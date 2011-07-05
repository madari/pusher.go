package pusher

import (
	"container/list"
	"container/vector"
	"fmt"
	"http"
	"os"
	"strings"
	"sync"
	"time"
)

// Stats holds information about a channel.
type Stats struct {
	Created       int64 // The time the channel was created.
	Delivered     int64 // The amonut of messages delivered.
	LastPublished int64 // The time the last message was published.
	LastRequested int64 // The time the last message was requested.
	Published     int64 // The amount of messages published.
	Subscribers   int   // The amount of active subscribers.
	Queued        int   // The amount of messages queued.
}

// ChannelSlice provides sort.Interface to sort by channel activities in ascending order
// i.e. where least active channel is first.
type channelSlice []*channel

func (cs channelSlice) Len() int {
	return len(cs)
}

func (cs channelSlice) Less(i, j int) bool {
	return cs[i].stamp() < cs[j].stamp()
}

func (cs channelSlice) Swap(i, j int) {
	cs[i], cs[j] = cs[i], cs[j]
}

// Channel represents a gateway for messages to pass from publishers to
// subscribers.
type channel struct {
	subscribers *list.List     // The active subscribers to this channel.
	config      *Configuration // The configuration options.
	lock        sync.RWMutex   // Protects the state.
	lastMessage *Message       // The most recent message that delivered.
	stats       Stats          // The statistics of the channel
	id          string         // The name of the channel.
	queue       vector.Vector  // The messages, newest first.
}

// NewChannel creates a new channel.
func newChannel(id string, config *Configuration) (c *channel) {
	c = &channel{
		subscribers: list.New(),
		config:      config,
		stats:       Stats{Created: time.Seconds()},
		id:          id,
	}
	return
}

// Stamp return the time of the last activity on this channel.
func (c *channel) stamp() int64 {
	if c.stats.LastRequested == 0 && c.stats.LastPublished == 0 {
		return c.stats.Created
	} else if c.stats.LastRequested > c.stats.LastPublished {
		return c.stats.LastRequested
	}
	return c.stats.LastPublished
}

// WriteStats writes statistics about this channel straight to rw. It
// will determine the encoding of the stats based on the request's Accept-header.
func (c *channel) writeStats(rw http.ResponseWriter, req *http.Request) os.Error {
	var typ, subtype string

	// Valid Accept-types are {text | application} / {statFormats...}.
	// If these conditions are not met, we will revert to text/plain.
	accept := strings.Split(strings.ToLower(req.Header.Get("Accept")), "/", 2)
	if len(accept) != 2 || (accept[0] != "text" && accept[0] != "application") {
		typ, subtype = "text", "plain"
	} else {
		typ, subtype = accept[0], accept[1]
	}

	format := statFormats[subtype]
	if format == "" {
		subtype = "plain"
		format = statFormats["plain"]
	}

	c.lock.RLock()
	stats := c.stats
	c.lock.RUnlock()

	rw.Header().Set("Content-Type", typ+"/"+subtype)

	// format plain mode stamps to ago
	if subtype == "plain" {
		if stats.LastRequested > 0 {
			stats.LastRequested = time.Seconds() - stats.LastRequested
		} else {
			stats.LastRequested = -1
		}
		if stats.LastPublished > 0 {
			stats.LastPublished = time.Seconds() - stats.LastPublished
		} else {
			stats.LastRequested = -1
		}
	}
	_, err := fmt.Fprintf(rw, format, stats.Queued, stats.LastRequested, stats.LastPublished,
		stats.Subscribers, stats.Published, stats.Delivered)
	return err
}

// Stats returns a snapshot of the current statistics.
func (c *channel) Stats() (stats Stats) {
	c.lock.RLock()
	stats = c.stats
	c.lock.RUnlock()
	return
}

// Publish takes the given message and sends it to all active subscribers. It
// can also queue the message for future requests.
func (c *channel) Publish(m *Message, queue bool) (n int) {
	c.lock.Lock()
	n = c.publish(m, queue)
	c.lock.Unlock()
	return
}

// PublishString takes the given string and sends it to all active subscribers along
// with a text/plain content-type and a 200 status. It can also queue the message for
// future requests.
func (c *channel) PublishString(s string, queue bool) int {
	m := &Message{Status: http.StatusOK, ContentType: "text/plain", Payload: []byte(s)}
	return c.Publish(m, queue)
}

func (c *channel) publish(m *Message, queue bool) (n int) {
	m.time = time.Seconds()
	m.etag = 0

	if c.lastMessage != nil && c.lastMessage.time == m.time {
		m.etag = c.lastMessage.etag + 1
	}

	c.lastMessage = m
	c.stats.Published++
	c.stats.LastPublished = time.Seconds()

	for e := c.subscribers.Front(); e != nil; e = e.Next() {
		client := e.Value.(chan *Message)
		select {
		case client <- m:
			n++
		default:
		}
		close(client)
	}
	c.subscribers.Init()
	c.stats.Subscribers = 0
	c.stats.Delivered += int64(n)

	if queue && c.config.ChannelCapacity > 0 {
		if c.queue.Len() >= c.config.ChannelCapacity {
			c.queue.Pop()
		} else {
			c.stats.Queued++
		}
		c.queue.Insert(0, m)
	}

	return
}

// Unsubscribe removes the given subscriber from subscribers.
func (c *channel) Unsubscribe(elem *list.Element) {
	c.lock.Lock()
	close(elem.Value.(chan *Message))
	c.subscribers.Remove(elem)
	c.stats.Subscribers = c.subscribers.Len()
	c.lock.Unlock()
}

// Subscribe registers a new subscriber. It takes If-Modified-Since and Etag
// arguments to determine the requested message. If a suitable message is
// immediately available (or a conflict has occured), only the message will be
// returned. If the interval polling mechanism is used, it will return
// immediately but with zero'd return values. Otherwise a list.Element is
// returned, whose value is a channel of *Message type, that might eventually
// receive the desired message.
func (c *channel) Subscribe(since int64, etag int) (*list.Element, *Message) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.stats.LastRequested = time.Seconds()

	switch c.config.ConcurrencyMode {
	case ConcurrencyModeLIFO:
		c.publish(conflictMessage, false)
	case ConcurrencyModeFILO:
		if c.stats.Subscribers > 0 {
			return nil, conflictMessage
		}
	}

	for i := c.queue.Len() - 1; i >= 0; i-- {
		m := c.queue.At(i).(*Message)
		if m.time >= since {
			if m.time == since && m.etag <= etag {
				continue
			}
			c.stats.Delivered++
			return nil, m
		}
	}

	if c.config.PollingMechanism == PollingMechanismInterval {
		return nil, nil
	}

	ch := make(chan *Message, 0)
	elem := c.subscribers.PushBack((chan *Message)(ch))
	c.stats.Subscribers++
	return elem, nil
}

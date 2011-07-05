package pusher

import (
	"bytes"
	"http"
	"sync"
	"sort"
	"strconv"
	"time"
)

// Pusher represents a set of channels that share the same
// behaviour e.g. the same configuration options, acceptor and
// garbage collector.
//
// Once a pusher has been initialized using New(), it can be muxed
// into any http ServeMux by passing PublisherHandler and/or SubscriberHandler
// to ServeMux.Handle.
type pusher struct {
	acceptor          Acceptor
	channels          map[string]*channel
	config            Configuration
	lock              sync.RWMutex // Protects channels.
	PublisherHandler  http.Handler // The handler for publisher locations.
	SubscriberHandler http.Handler // The handler for subscriber locations.
}

// New creates a new pusher that is ready to be muxed into any ServeMux.
// The new pusher (and any channel in it's context) will behave according
// to the given configuration options are acceptor logic.
func New(acceptor Acceptor, config Configuration) (p *pusher) {
	p = &pusher{
		acceptor: acceptor,
		channels: make(map[string]*channel),
		config:   config,
	}

	p.PublisherHandler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		p.handlePublisher(rw, req)
	})
	p.SubscriberHandler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		p.handleSubscriber(rw, req)
	})

	if config.GCInterval > 0 && (config.MaxChannelIdleTime > 0 || config.MaxChannels > 0) {
		go func() {
			for _ = range time.Tick(config.GCInterval) {
				p.GC()
			}
		}()
	}

	return
}

// Channel returns the channel identified with the given channel id. If the channel
// does not yet exists, it will be created.
func (p *pusher) Channel(cid string) (c *channel, created bool) {
	p.lock.Lock()
	c, ok := p.channels[cid]
	if !ok {
		created = true
		c = newChannel(cid, &p.config)
		p.channels[cid] = c
	}
	p.lock.Unlock()
	return
}

// GC does garbage collection by collecting stale channels (see MaxChannelIdleTime
// configuration option) and purges them. It also removes as many channels (least
// active first) as needed until there are no more than MaxChannels (configuration option)
// channels.
//
// TODO: This is a really naive implementation and will not scale if there are billions
// of channels. We could do better.
func (p *pusher) GC() int {
	var i int
	var c *channel

	start := time.Nanoseconds()
	limit := (start - p.config.MaxChannelIdleTime) / 1e9
	count := len(p.channels)

	Logger.Printf("GC: Started with %d channels", count)

	p.lock.Lock()
	sorted := make(channelSlice, len(p.channels))
	for _, c = range p.channels {
		sorted[i] = c
		i++
	}
	sort.Sort(sorted)
	gc := sorted[:0]
	for i, c = range sorted {
		if (p.config.MaxChannels == 0 || count <= p.config.MaxChannels) && c.stamp() >= limit {
			break
		}
		gc = sorted[:i+1]
		p.channels[c.id] = nil, false
		count--
	}
	p.lock.Unlock()

	for _, c := range gc {
		c.Publish(goneMessage, false)
		Logger.Printf("GC: Channel %q was garbage collected", c.id)
	}

	Logger.Printf("GC: Ended in %d ns with %d channels garbage collected", time.Nanoseconds()-start, len(gc))
	return len(gc)
}

// HandlePublisher is responsible for answering requests to the publisher locations. It will use
// the pusher's acceptor to extract the channel. If acceptor does not provide a non-empty channel id,
// then a 404 will be returned. Otherwise the handler will take actions based on the http method of
// the request. All 200-level responses will be paired with information about the channel requested
// encoded in a format requested via the Accept-header.
//
// - GET     Yields a 404 if the channel does not exists, 200 otherwise
// - PUT     Tries to create the channel and yield 200
// - POST    Creates a new message using the request's body and content-type (unless the content-type is
//           explictly overridden using the ContentType configuration option). It will create the channel
//           if needed and it yields a 201 if the message was immediately delivered to atleast one
//           subscriber and 202 otherwise.
// - DELETE  Deletes the channel. Active subscribers will receive a 410. If the channel existed, a 200
//           will be responded, 404 otherwise.
// 
// Any other request using a method other than those that were described above will be responded with a
// 405 response.
func (p *pusher) handlePublisher(rw http.ResponseWriter, req *http.Request) {
	cid := p.acceptor(req)
	if cid == "" {
		Logger.Printf("Pub/404: Acceptor denied access to URL %q [%s]", req.RawURL, req.RemoteAddr)
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	status := http.StatusMethodNotAllowed
	var c *channel
	var ok bool

	switch req.Method {
	case "GET":
		p.lock.RLock()
		c, ok = p.channels[cid]
		p.lock.RUnlock()

		if ok {
			Logger.Printf("Pub/200: Channel information retrieved for %q [%s]", cid, req.RemoteAddr)
			status = http.StatusOK
		} else {
			Logger.Printf("Pub/404: Channel information retrieved for %q [%s]", cid, req.RemoteAddr)
			status = http.StatusNotFound
		}

	case "PUT":
		c, ok = p.Channel(cid)
		if ok {
			Logger.Printf("Pub/200: Channel %q created [%s]", cid, req.RemoteAddr)
		} else {
			Logger.Printf("Pub/200: Channel %q was already created [%s]", cid, req.RemoteAddr)
		}
		status = http.StatusOK

	case "POST":
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(req.Body); err != nil {
			Logger.Print("ReadFrom(req.Body):", err)
			status = http.StatusInternalServerError
			break
		}

		ctype := p.config.ContentType
		if ctype == "" {
			ctype = req.Header.Get("Content-Type")
		}

		c, _ = p.Channel(cid)

		if c.Publish(&Message{Status: http.StatusOK, ContentType: ctype, Payload: buf.Bytes()}, true) > 0 {
			Logger.Printf("Pub/201: A message was published to channel %q and delivered simultaneously to some clients [%s]", cid, req.RemoteAddr)
			status = http.StatusCreated
		} else {
			Logger.Printf("Pub/202: A message was queued to channel %q [%s]", cid, req.RemoteAddr)
			status = http.StatusAccepted
		}

	case "DELETE":
		p.lock.Lock()
		c, ok = p.channels[cid]
		if ok {
			p.channels[cid] = nil, false
			p.lock.Unlock()
			c.Publish(goneMessage, false)
			Logger.Printf("Pub/200: Channel %q was deleted [%s]", cid, req.RemoteAddr)
			status = http.StatusOK
		} else {
			p.lock.Unlock()
			Logger.Printf("Pub/404: Trying to delete a non-existent channel %q [%s]", cid, req.RemoteAddr)
			status = http.StatusNotFound
		}
	}

	rw.WriteHeader(status)
	if status >= 200 && status < 300 && c != nil {
		if err := c.writeStats(rw, req); err != nil {
			Logger.Print("writeStats:", err)
		}
	}
	return
}

// HandleSubscriber is responsible for answering requests to the subscriber locations. It will use
// the pusher's acceptor to extract the channel. If acceptor does not provide a non-empty channel id,
// then a 404 will be returned. If the request method is other than GET then a 405 will be returned.
// If the channel does not exists, the handler will either reject or create the channel depending on
// the AllowChannelCreation configuration option.
//
// The handler uses If-Modified-Since and If-None-Match headers to determine which message the client
// requested. If these are omitted, then the oldest available message is used. All 200-level responses
// will contain Etag and Last-Modified headers for the client to use during it's next request.
//
// The PollingMechanism and ConcurrencyMode configuration options affect the behavior of this handler.
// If long-polling is used, the response is delayed until a message has become available or a period
// defined by the configuration option PollingTimeout has passed. The request will be responded with
// a 304 if no message was available or with a 200 along with the ContentType and Payload from the
// message. Additionally a 409 might be responded depending on the used ConcurrencyMode. See the
// documentation for ConcurrencyModeBroadcast, ConcurrencyModeFILO and ConcurrencyModeLIFO for
// details.
func (p *pusher) handleSubscriber(rw http.ResponseWriter, req *http.Request) {
	cid := p.acceptor(req)
	var status int
	var since int64

	rw.Header().Set("Vary", "If-None-Match, If-Modified-Since")

	if req.Method != "GET" {
		Logger.Printf("Sub/405: A non GET request to channel %q [%s]", cid, req.RemoteAddr)
		status = http.StatusMethodNotAllowed
	} else if cid == "" {
		Logger.Printf("Sub/404: Acceptor denied access to URL %q [%s]", req.RawURL, req.RemoteAddr)
		status = http.StatusNotFound
	}

	if status != 0 {
		rw.WriteHeader(status)
		return
	}

	if ifsince, _ := time.Parse(http.TimeFormat, req.Header.Get("If-Modified-Since")); ifsince != nil {
		since = ifsince.Seconds()
	}
	etag, _ := strconv.Atoi(req.Header.Get("If-None-Match"))

	p.lock.Lock()
	c, ok := p.channels[cid]
	if !ok {
		if !p.config.AllowChannelCreation {
			p.lock.Unlock()
			Logger.Printf("Sub/403: Trying to subscribe to a non-existent channel %q [%s]", cid, req.RemoteAddr)
			rw.WriteHeader(http.StatusForbidden)
			return
		} else {
			Logger.Printf("Sub: Channel %q created [%s]", cid, req.RemoteAddr)
			c = newChannel(cid, &p.config)
			p.channels[cid] = c
		}
	}

	Logger.Printf("Sub: New subscription to channel %q [%s]", cid, req.RemoteAddr)
	sub, message := c.Subscribe(since, etag)
	p.lock.Unlock()

	if sub != nil {
		if p.config.PollingTimeout > 0 {
			select {
			case message = <-sub.Value.(chan *Message):
			case <-time.After(p.config.PollingTimeout):
				c.Unsubscribe(sub)
			}
		} else {
			message = <-sub.Value.(chan *Message)
		}
	}
	if message == nil {
		Logger.Printf("Sub/304: Subscription to channel %q timed out (probably) [%s]", cid, req.RemoteAddr)
		rw.WriteHeader(http.StatusNotModified)
		return
	}

	rw.Header().Set("Etag", strconv.Itoa(message.etag))
	rw.Header().Set("Last-Modified", time.SecondsToUTC(message.time).Format(http.TimeFormat))

	if message.ContentType != "" {
		rw.Header().Set("Content-Type", message.ContentType)
	}

	rw.WriteHeader(message.Status)
	if message.Payload != nil {
		rw.Write(message.Payload)
	}

	Logger.Printf("Sub/%d: Delivered message in channel %q [%s]", message.Status, cid, req.RemoteAddr)
}

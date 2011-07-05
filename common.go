package pusher

import (
	"http"
	"log"
	"os"
)

// Concurrency mode defines the behaviour of channels when there are
// multiple subscribers. If conflicts occur in FILO and LIFO modes, a
// 409 Conflict message will be broadcasted to the clients that were kicked
// out.
const (
	ConcurrencyModeBroadcast = iota // Broadcasting
	ConcurrencyModeFILO             // First-in, last-out
	ConcurrencyModeLIFO             // Last-in, first-out
)

// Polling mechanism defines the behaviour of response-cycles.
const (
	PollingMechanismLong     = iota // Long-polling
	PollingMechanismInterval        // Interval-polling
)

// Logger is the logging facility used by Pusher
var Logger = log.New(os.Stderr, "", log.LstdFlags)

// Configuration holds various parameters for the server.
type Configuration struct {
	AllowChannelCreation bool   // Can channels be created through subscriber locations.
	ChannelCapacity      int    // The capacity of the channels (queue length, 0=unlimited).
	ConcurrencyMode      int    // The behaviour of channels under concurrent subscribers
	ContentType          string // Override outgoing Content-Type headers.
	GCInterval           int64  // The interval between collecting stale channels (0=disable).
	MaxChannels          int    // Maximum amount of channels (0=unlimited).
	MaxChannelIdleTime   int64  // Maximum idle time for a channel (0=unlimited).
	PollingMechanism     int    // The behaviour of response-cycles.
	PollingTimeout       int64  // Maximum time for a long-polling connection (0=unlimited).
}

// DefaultConfiguration holds some sensible defaults.
var DefaultConfiguration = Configuration{
	ChannelCapacity:    20,
	GCInterval:         60e9,
	MaxChannelIdleTime: 600e9,
	PollingTimeout:     20e9,
}

// A Message defines a single application specific data packet,
// containing the original content-type and body along with
// a HTTP status code to use when delivering it.
type Message struct {
	ContentType string // HTTP content-type to use
	Payload     []byte // the body to use
	Status      int    // HTTP status code to use
	etag        int    // HTTP Etag to use
	time        int64  // HTTP Last-Modified e.g. the time the message was created
}

var (
	conflictMessage = &Message{Status: http.StatusConflict}
	goneMessage     = &Message{Status: http.StatusGone}

	statFormats = map[string]string{
		"plain": `queued messages: %d
last requested: %d sec. ago (-1=never)
last published: %d sec. ago (-1=never)
active subscribers: %d
total published: %d
total delivered: %d`,
		"json": `{"queued":%d,"lastRequested":%d,"lastPublished":%d,"subscribers":%d,"published":%d,"delivered":%d}`,
	}
)

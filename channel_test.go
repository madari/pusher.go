package pusher

import (
	"testing"
	"time"
)

var intervalConf = Configuration{
	PollingMechanism: PollingMechanismInterval,
	ChannelCapacity:  3,
}

var longConf = Configuration{
	PollingMechanism: PollingMechanismLong,
	ChannelCapacity:  3,
}

// Empty channel tests
func TestEmptyChannel(t *testing.T) {
	channel := newChannel("test", &intervalConf)
	if e, m := channel.Subscribe(0, 0); e != nil || m != nil {
		t.Error("Expected nothing (1)")
	}
	tm1 := &Message{Status: 1, ContentType: "tm1.ctype", Payload: []byte("tm1.payload")}
	channel.Publish(tm1, false)
	if e, m := channel.Subscribe(0, 0); e != nil || m != nil {
		t.Error("Expected nothing (2)")
	}

	s := channel.Stats()
	if s.Created == 0 || s.LastRequested == 0 || s.LastPublished == 0 || channel.stamp() != s.LastRequested {
		t.Errorf("Invalid stamps %#v", s)
	}
	if s.Queued != 0 || s.Subscribers != 0 || s.Delivered != 0 || s.Published != 1 {
		t.Errorf("Invalid counters %#v", s)
	}
}

// 1 message tests
func TestSimpleChannel(t *testing.T) {
	channel := newChannel("test", &intervalConf)
	tm1 := &Message{Status: 1, ContentType: "tm1.ctype", Payload: []byte("tm1.payload")}
	channel.Publish(tm1, true)
	if e, m := channel.Subscribe(0, 0); e != nil || m != tm1 {
		t.Error("Expected tm1 (1)")
	}
	if e, m := channel.Subscribe(0, 0); e != nil || m != tm1 {
		t.Error("Expected tm1 (2)")
	}
	if tm1.etag != 0 || tm1.time == 0 {
		t.Error("Invalid etag or time")
	}
}

// 2 message tests
func TestMediumChannel(t *testing.T) {
	channel := newChannel("test", &intervalConf)
	tm1 := &Message{Status: 1, ContentType: "tm1.ctype", Payload: []byte("tm1.payload")}
	tm2 := &Message{Status: 2, ContentType: "tm2.ctype", Payload: []byte("tm2.payload")}

	channel.Publish(tm1, true)
	channel.Publish(tm2, true)
	if e, m := channel.Subscribe(0, 0); e != nil || m != tm1 {
		t.Error("Expected tm1 (1)")
	}
	if e, m := channel.Subscribe(tm1.time, tm1.etag); e != nil || m != tm2 {
		t.Error("Expected tm2 (1)")
	}

	// drained
	if e, m := channel.Subscribe(tm2.time, tm2.etag); e != nil || m != nil {
		t.Error("Expected nothing")
	}

	// again
	time, etag := tm2.time, tm2.etag

	tm1 = &Message{Status: 1, ContentType: "tm1.ctype", Payload: []byte("tm1.payload")}
	tm2 = &Message{Status: 2, ContentType: "tm2.ctype", Payload: []byte("tm2.payload")}
	channel.Publish(tm1, true)
	channel.Publish(tm2, true)
	if e, m := channel.Subscribe(time, etag); e != nil || m != tm1 {
		t.Errorf("Expected tm1 (2) %#v", m)
	}
	if e, m := channel.Subscribe(tm1.time, tm1.etag); e != nil || m != tm2 {
		t.Error("Expected tm2 (2)")
	}

	// drained
	if e, m := channel.Subscribe(tm2.time, tm2.etag); e != nil || m != nil {
		t.Error("Expected nothing")
	}
}

// full channel tests
func TestFullChannel(t *testing.T) {
	channel := newChannel("test", &intervalConf)
	tm1 := &Message{Status: 1, ContentType: "tm1.ctype", Payload: []byte("tm1.payload")}
	tm2 := &Message{Status: 2, ContentType: "tm2.ctype", Payload: []byte("tm2.payload")}
	tm3 := &Message{Status: 3, ContentType: "tm3.ctype", Payload: []byte("tm3.payload")}
	tm4 := &Message{Status: 4, ContentType: "tm4.ctype", Payload: []byte("tm4.payload")}

	channel.Publish(tm1, true)
	channel.Publish(tm2, true)
	channel.Publish(tm3, true)
	channel.Publish(tm4, true)
	if e, m := channel.Subscribe(0, 0); e != nil || m != tm2 {
		t.Error("Expected tm2 (1)")
	}
	if e, m := channel.Subscribe(tm2.time, tm2.etag); e != nil || m != tm3 {
		t.Error("Expected tm3 (1)")
	}
	if e, m := channel.Subscribe(tm3.time, tm3.etag); e != nil || m != tm4 {
		t.Error("Expected tm3 (1)")
	}
	if e, m := channel.Subscribe(tm3.time, tm3.etag); e != nil || m != tm4 {
		t.Error("Expected tm3 (2)")
	}

	// drained
	if e, m := channel.Subscribe(tm4.time, tm4.etag); e != nil || m != nil {
		t.Error("Expected nothing")
	}

	s := channel.Stats()
	if s.Created == 0 || s.LastRequested == 0 || s.LastPublished == 0 || channel.stamp() != s.LastRequested {
		t.Errorf("Invalid stamps %#v", s)
	}
	if s.Queued != 3 || s.Subscribers != 0 || s.Delivered != 4 || s.Published != 4 {
		t.Errorf("Invalid counters %#v", s)
	}
}

// 1 message tests
func TestLongSimpleChannel(t *testing.T) {
	channel := newChannel("test", &longConf)
	tm1 := &Message{Status: 1, ContentType: "tm1.ctype", Payload: []byte("tm1.payload")}
	go func() {
		time.Sleep(1e9 / 2)
		channel.Publish(tm1, true)
	}()
	if e, m := channel.Subscribe(0, 0); e == nil || m != nil {
		t.Error("Expected channel")
	} else {
		m = <-e.Value.(chan *Message)
		if m != tm1 {
			t.Error("Expected tm1 (1)")
		}
	}
	if e, m := channel.Subscribe(0, 0); e != nil || m != tm1 {
		t.Error("Expected tm1 (2)")
	}
}

// 2 message tests
func TestLongMediumChannel(t *testing.T) {
	channel := newChannel("test", &longConf)
	tm1 := &Message{Status: 1, ContentType: "tm1.ctype", Payload: []byte("tm1.payload")}
	tm2 := &Message{Status: 2, ContentType: "tm2.ctype", Payload: []byte("tm2.payload")}

	go func() {
		time.Sleep(1e9 / 2)
		if s := channel.Stats(); s.Subscribers != 1 {
			t.Errorf("Invalid amount of subscribers %#v", s)
		}
		channel.Publish(tm1, true)
		time.Sleep(1e9 / 2)
		channel.Publish(tm2, true)
	}()

	if e, m := channel.Subscribe(0, 0); e == nil || m != nil {
		t.Error("Expected channel")
	} else {
		m = <-e.Value.(chan *Message)
		if m != tm1 {
			t.Error("Expected tm1 (1)")
		}
	}
	if e, m := channel.Subscribe(0, 0); e != nil || m != tm1 {
		t.Error("Expected tm1 (2)")
	}

	if e, m := channel.Subscribe(tm1.time, tm1.etag); e == nil || m != nil {
		t.Error("Expected channel")
	} else {
		m = <-e.Value.(chan *Message)
		if m != tm2 {
			t.Error("Expected tm2 (1)")
		}
	}
	if e, m := channel.Subscribe(tm1.time, tm1.etag); e != nil || m != tm2 {
		t.Error("Expected tm2 (2)")
	}

	// just for fun
	channel.Publish(tm1, true)
	channel.Publish(tm1, true)
	channel.Publish(tm1, true)

	s := channel.Stats()
	if s.Created == 0 || s.LastRequested == 0 || s.LastPublished == 0 || channel.stamp() != s.LastPublished {
		t.Errorf("Invalid stamps %#v", s)
	}
	if s.Queued != 3 || s.Subscribers != 0 || s.Delivered != 4 || s.Published != 5 {
		t.Errorf("Invalid counters %#v", s)
	}
}

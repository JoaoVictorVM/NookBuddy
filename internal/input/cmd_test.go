package input

import (
	"testing"
	"time"
)

func preloaded(n int) *Source {
	source := newSource(newFakeBackend(), testReadyTimeout)
	for i := range n {
		kind := Click
		if i%2 == 1 {
			kind = Key
		}
		source.events <- Event{Type: kind, Timestamp: time.Now()}
	}
	return source
}

func TestCmd_DrainsUpToBatchLimit(t *testing.T) {
	source := preloaded(300)

	first, ok := Cmd(source)().(InputEventMsg)
	if !ok {
		t.Fatal("first Cmd did not return an InputEventMsg")
	}
	if got := len(first.Events); got != batchLimit {
		t.Errorf("first batch = %d events, want %d", got, batchLimit)
	}

	second, ok := Cmd(source)().(InputEventMsg)
	if !ok {
		t.Fatal("second Cmd did not return an InputEventMsg")
	}
	if got := len(second.Events); got != 300-batchLimit {
		t.Errorf("second batch = %d events, want %d", got, 300-batchLimit)
	}

	if first.Events[0].Type != Click || first.Events[1].Type != Key {
		t.Error("batch does not preserve capture order")
	}
}

func TestCmd_ReturnsNoMsgOnEmptyChannel(t *testing.T) {
	if msg := Cmd(preloaded(0))(); msg != nil {
		t.Errorf("Cmd on an empty channel returned %#v, want nil", msg)
	}
}

func TestCmd_StopsAtClosedChannel(t *testing.T) {
	source := preloaded(3)
	close(source.events)

	msg, ok := Cmd(source)().(InputEventMsg)
	if !ok || len(msg.Events) != 3 {
		t.Fatalf("Cmd = %#v, want the 3 buffered events", msg)
	}
	if again := Cmd(source)(); again != nil {
		t.Errorf("Cmd on a closed, drained channel returned %#v, want nil", again)
	}
}

func TestCmd_PipelineLimits(t *testing.T) {
	if batchLimit != 256 {
		t.Errorf("batchLimit = %d, want 256", batchLimit)
	}
	if channelCapacity != 1024 {
		t.Errorf("channelCapacity = %d, want 1024", channelCapacity)
	}
}

func TestSource_EventsFlowThroughCmdUnderRace(t *testing.T) {
	source, fake := startReady(t)

	const total = 2000
	go func() {
		for i := range total {
			if i%2 == 0 {
				fake.emit(rawClick)
			} else {
				fake.emit(rawKey)
			}
		}
	}()

	clicks, keys := 0, 0
	deadline := time.Now().Add(3 * time.Second)
	for clicks+keys+int(source.dropped.Load()) < total {
		if time.Now().After(deadline) {
			t.Fatalf("received %d clicks + %d keys, want %d", clicks, keys, total)
		}
		msg, ok := Cmd(source)().(InputEventMsg)
		if !ok {
			time.Sleep(time.Millisecond)
			continue
		}
		for _, ev := range msg.Events {
			switch ev.Type {
			case Click:
				clicks++
			case Key:
				keys++
			}
		}
	}
	if clicks == 0 || keys == 0 {
		t.Errorf("clicks = %d, keys = %d, want both to flow through", clicks, keys)
	}
}

type channelSource struct {
	events chan Event
}

func (c channelSource) Events() <-chan Event { return c.events }

func TestCmd_AcceptsFakeEventSource(t *testing.T) {
	source := channelSource{events: make(chan Event, 300)}
	for i := range 300 {
		kind := Click
		if i%3 == 0 {
			kind = Key
		}
		source.events <- Event{Type: kind, Timestamp: time.Now()}
	}

	first, ok := Cmd(source)().(InputEventMsg)
	if !ok || len(first.Events) != batchLimit {
		t.Fatalf("first drain = %#v, want %d events", first, batchLimit)
	}
	if first.Events[0].Type != Key || first.Events[1].Type != Click {
		t.Error("drain from a fake source does not preserve order")
	}
	second, ok := Cmd(source)().(InputEventMsg)
	if !ok || len(second.Events) != 300-batchLimit {
		t.Fatalf("second drain = %d events, want %d", len(second.Events), 300-batchLimit)
	}
	if Cmd(source)() != nil {
		t.Error("drain of an empty fake source returned a message, want nil")
	}
}

func TestCmd_SourceSatisfiesEventSource(t *testing.T) {
	var _ EventSource = NewSource()
}

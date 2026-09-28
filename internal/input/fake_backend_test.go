package input

import (
	"sync"
	"sync/atomic"
	"time"
)

type fakeBackend struct {
	startErr    error
	events      chan rawEvent
	stopped     chan struct{}
	stopOnce    sync.Once
	stopCalls   atomic.Int32
	panicOnNext atomic.Bool
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{
		events:  make(chan rawEvent, 4*channelCapacity),
		stopped: make(chan struct{}),
	}
}

func (f *fakeBackend) start() error {
	return f.startErr
}

func (f *fakeBackend) next() (rawEvent, bool) {
	select {
	case ev := <-f.events:
		if f.panicOnNext.Load() {
			panic("fake backend exploded")
		}
		return ev, true
	case <-f.stopped:
		return rawEvent{}, false
	}
}

func (f *fakeBackend) stop() {
	f.stopCalls.Add(1)
	f.stopOnce.Do(func() { close(f.stopped) })
}

func (f *fakeBackend) emit(kind rawKind) {
	f.events <- rawEvent{kind: kind, when: time.Now()}
}

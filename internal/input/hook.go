package input

import (
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	hook "github.com/robotn/gohook"
)

const (
	channelCapacity  = 1024
	readyTimeout     = 3 * time.Second
	dropWarnInterval = time.Minute
)

const (
	stateStarting int32 = iota
	stateAvailable
	stateUnavailable
)

type gohookBackend struct {
	events chan hook.Event
}

func (b *gohookBackend) start() error {
	b.events = hook.Start()
	return nil
}

func (b *gohookBackend) next() (rawEvent, bool) {
	ev, ok := <-b.events
	if !ok {
		return rawEvent{}, false
	}
	return classify(ev), true
}

func (b *gohookBackend) stop() {
	hook.End()
}

func classify(ev hook.Event) rawEvent {
	raw := rawEvent{kind: rawIgnored, when: ev.When}
	switch ev.Kind {
	case hook.HookEnabled:
		raw.kind = rawReady
	case hook.KeyDown:
		raw.kind = rawKey
	case hook.MouseDown:
		if ev.Button >= 1 && ev.Button <= 3 {
			raw.kind = rawClick
		}
	}
	return raw
}

type Source struct {
	backend      backend
	readyTimeout time.Duration

	events chan Event
	ready  chan struct{}
	done   chan struct{}

	started  atomic.Bool
	state    atomic.Int32
	dropped  atomic.Uint64
	lastWarn time.Time

	readyOnce sync.Once
	stopOnce  sync.Once
}

func NewSource() *Source {
	return newSource(&gohookBackend{}, readyTimeout)
}

func newSource(b backend, timeout time.Duration) *Source {
	return &Source{
		backend:      b,
		readyTimeout: timeout,
		events:       make(chan Event, channelCapacity),
		ready:        make(chan struct{}),
		done:         make(chan struct{}),
	}
}

func (s *Source) Start() error {
	s.started.Store(true)

	if err := s.backend.start(); err != nil {
		slog.Error("nookbuddy: input hook failed to start", "error", err)
		s.state.Store(stateUnavailable)
		s.stopOnce.Do(func() {})
		close(s.events)
		close(s.done)
		return nil
	}

	go s.run()

	select {
	case <-s.ready:
		s.state.CompareAndSwap(stateStarting, stateAvailable)
	case <-s.done:
		s.state.Store(stateUnavailable)
	case <-time.After(s.readyTimeout):
		slog.Error("nookbuddy: input hook did not become ready in time", "timeout", s.readyTimeout)
		s.state.Store(stateUnavailable)
		s.stopBackend()
	}

	return nil
}

func (s *Source) Events() <-chan Event {
	return s.events
}

func (s *Source) Status() Status {
	if s.state.Load() == stateAvailable {
		return Available
	}
	return Unavailable
}

func (s *Source) Stop() {
	if !s.started.Load() {
		return
	}
	s.state.Store(stateUnavailable)
	s.stopBackend()
	<-s.done
}

func (s *Source) stopBackend() {
	s.stopOnce.Do(s.backend.stop)
}

func (s *Source) run() {
	defer close(s.done)
	defer close(s.events)
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("nookbuddy: input hook goroutine panicked", "panic", recovered, "stack", string(debug.Stack()))
			s.state.Store(stateUnavailable)
			s.stopBackend()
		}
	}()

	for {
		raw, ok := s.backend.next()
		if !ok {
			return
		}
		s.dispatch(raw)
	}
}

func (s *Source) dispatch(raw rawEvent) {
	switch raw.kind {
	case rawReady:
		s.readyOnce.Do(func() { close(s.ready) })
	case rawClick:
		s.forward(Event{Type: Click, Timestamp: raw.when})
	case rawKey:
		s.forward(Event{Type: Key, Timestamp: raw.when})
	}
}

func (s *Source) forward(ev Event) {
	select {
	case s.events <- ev:
	default:
		total := s.dropped.Add(1)
		if now := time.Now(); now.Sub(s.lastWarn) >= dropWarnInterval {
			s.lastWarn = now
			slog.Warn("nookbuddy: input channel saturated, dropping events", "dropped_total", total)
		}
	}
}

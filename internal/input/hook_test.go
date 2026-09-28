package input

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	hook "github.com/robotn/gohook"
)

const testReadyTimeout = 100 * time.Millisecond

type recordingHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *recordingHandler) WithGroup(string) slog.Handler { return h }

func (h *recordingHandler) count(level slog.Level, contains string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, r := range h.records {
		if r.Level == level && strings.Contains(r.Message, contains) {
			n++
		}
	}
	return n
}

func captureLogs(t *testing.T) *recordingHandler {
	t.Helper()
	handler := &recordingHandler{}
	previous := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return handler
}

func startReady(t *testing.T) (*Source, *fakeBackend) {
	t.Helper()
	fake := newFakeBackend()
	fake.emit(rawReady)
	source := newSource(fake, testReadyTimeout)
	if err := source.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(source.Stop)
	return source, fake
}

func receive(t *testing.T, source *Source) Event {
	t.Helper()
	select {
	case ev, ok := <-source.Events():
		if !ok {
			t.Fatal("events channel closed unexpectedly")
		}
		return ev
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for an event")
	}
	return Event{}
}

func waitFor(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestSource_StartBecomesAvailableOnReadySignal(t *testing.T) {
	began := time.Now()
	source, _ := startReady(t)

	if got := source.Status(); got != Available {
		t.Fatalf("Status = %v, want Available", got)
	}
	if elapsed := time.Since(began); elapsed >= testReadyTimeout {
		t.Errorf("Start took %v, want it to return on the ready signal, before the %v timeout", elapsed, testReadyTimeout)
	}
}

func TestSource_StartTimesOutToUnavailable(t *testing.T) {
	logs := captureLogs(t)
	fake := newFakeBackend()
	source := newSource(fake, testReadyTimeout)

	began := time.Now()
	if err := source.Start(); err != nil {
		t.Fatalf("Start returned %v, want nil so the app keeps running", err)
	}
	t.Cleanup(source.Stop)

	if elapsed := time.Since(began); elapsed < testReadyTimeout {
		t.Errorf("Start returned after %v, want it to wait the full %v", elapsed, testReadyTimeout)
	}
	if got := source.Status(); got != Unavailable {
		t.Errorf("Status = %v, want Unavailable", got)
	}
	if fake.stopCalls.Load() == 0 {
		t.Error("backend was not stopped after the readiness timeout")
	}
	if logs.count(slog.LevelError, "did not become ready") != 1 {
		t.Error("readiness timeout was not logged")
	}
}

func TestSource_DefaultReadyTimeoutIsThreeSeconds(t *testing.T) {
	if readyTimeout != 3*time.Second {
		t.Errorf("readyTimeout = %v, want 3s", readyTimeout)
	}
	if got := NewSource().readyTimeout; got != 3*time.Second {
		t.Errorf("NewSource().readyTimeout = %v, want 3s", got)
	}
}

func TestSource_StartReturnsUnavailableOnBackendError(t *testing.T) {
	logs := captureLogs(t)
	fake := newFakeBackend()
	fake.startErr = errors.New("no desktop session")
	source := newSource(fake, testReadyTimeout)

	if err := source.Start(); err != nil {
		t.Fatalf("Start returned %v, want nil so the app keeps running", err)
	}
	if got := source.Status(); got != Unavailable {
		t.Errorf("Status = %v, want Unavailable", got)
	}
	if logs.count(slog.LevelError, "failed to start") != 1 {
		t.Error("backend start failure was not logged")
	}

	source.Stop()
	if fake.stopCalls.Load() != 0 {
		t.Error("Stop called backend.stop on a backend that never started")
	}
	if _, ok := <-source.Events(); ok {
		t.Error("events channel is open, want it closed when the hook never started")
	}
}

func TestSource_ClicksAndKeysAreClassifiedCorrectly(t *testing.T) {
	source, fake := startReady(t)

	fake.emit(rawClick)
	fake.emit(rawKey)

	first := receive(t, source)
	second := receive(t, source)
	if first.Type != Click {
		t.Errorf("first event = %v, want Click", first.Type)
	}
	if second.Type != Key {
		t.Errorf("second event = %v, want Key", second.Type)
	}
	if first.Timestamp.IsZero() || second.Timestamp.IsZero() {
		t.Error("events must carry the capture timestamp")
	}
}

func TestSource_MouseMoveAndScrollAreIgnored(t *testing.T) {
	source, fake := startReady(t)

	for range 50 {
		fake.emit(rawIgnored)
	}
	fake.emit(rawKey)

	if got := receive(t, source); got.Type != Key {
		t.Fatalf("first delivered event = %v, want the Key sentinel (ignored events leaked)", got.Type)
	}
	if pending := len(source.Events()); pending != 0 {
		t.Errorf("%d extra events pending, want 0", pending)
	}
}

func TestSource_ChannelDropsOnSaturation(t *testing.T) {
	captureLogs(t)
	source, fake := startReady(t)

	const emitted = channelCapacity + 500
	emitDone := make(chan struct{})
	go func() {
		for range emitted {
			fake.emit(rawClick)
		}
		close(emitDone)
	}()

	select {
	case <-emitDone:
	case <-time.After(2 * time.Second):
		t.Fatal("emitter blocked, the hook callback path must never block")
	}

	waitFor(t, "all events to be delivered or dropped", func() bool {
		return uint64(len(source.Events()))+source.dropped.Load() == emitted
	})
	if got := len(source.Events()); got != channelCapacity {
		t.Errorf("buffered events = %d, want %d", got, channelCapacity)
	}
	if got := source.dropped.Load(); got != emitted-channelCapacity {
		t.Errorf("dropped = %d, want %d", got, emitted-channelCapacity)
	}
}

func TestSource_DropWarningIsRateLimited(t *testing.T) {
	logs := captureLogs(t)
	source, fake := startReady(t)

	burst := func(size int) {
		before := source.dropped.Load()
		for range size {
			fake.emit(rawKey)
		}
		waitFor(t, "burst to be processed", func() bool {
			return len(source.Events()) == channelCapacity && source.dropped.Load() > before
		})
	}

	burst(channelCapacity + 10)
	burst(10)

	if got := logs.count(slog.LevelWarn, "saturated"); got != 1 {
		t.Errorf("saturation warnings = %d, want exactly 1 within the same minute", got)
	}
}

func TestSource_GoroutinePanicSetsUnavailable(t *testing.T) {
	logs := captureLogs(t)
	source, fake := startReady(t)
	if source.Status() != Available {
		t.Fatalf("precondition: Status = %v, want Available", source.Status())
	}

	fake.panicOnNext.Store(true)
	fake.emit(rawClick)

	waitFor(t, "status to become Unavailable", func() bool { return source.Status() == Unavailable })
	<-source.done

	if logs.count(slog.LevelError, "panicked") != 1 {
		t.Error("goroutine panic was not logged")
	}
	if fake.stopCalls.Load() == 0 {
		t.Error("backend was not stopped after the panic")
	}
}

func TestSource_PanicBeforeReadyIsUnavailable(t *testing.T) {
	captureLogs(t)
	fake := newFakeBackend()
	fake.panicOnNext.Store(true)
	fake.emit(rawReady)
	source := newSource(fake, testReadyTimeout)

	if err := source.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(source.Stop)

	if got := source.Status(); got != Unavailable {
		t.Errorf("Status = %v, want Unavailable", got)
	}
}

func TestSource_StopClosesChannelAndStopsBackend(t *testing.T) {
	source, fake := startReady(t)

	source.Stop()

	if fake.stopCalls.Load() != 1 {
		t.Errorf("backend stop calls = %d, want 1", fake.stopCalls.Load())
	}
	select {
	case <-source.done:
	default:
		t.Fatal("capture goroutine is still running after Stop")
	}
	if _, ok := <-source.Events(); ok {
		t.Error("events channel is open after Stop, want it closed")
	}
	if got := source.Status(); got != Unavailable {
		t.Errorf("Status after Stop = %v, want Unavailable", got)
	}

	source.Stop()
	if fake.stopCalls.Load() != 1 {
		t.Error("a second Stop must be a no-op")
	}
}

func TestSource_StopBeforeStartDoesNotBlock(t *testing.T) {
	source := newSource(newFakeBackend(), testReadyTimeout)

	stopped := make(chan struct{})
	go func() {
		source.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop before Start blocked")
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		ev   hook.Event
		want rawKind
	}{
		{"hook enabled is the ready signal", hook.Event{Kind: hook.HookEnabled}, rawReady},
		{"key down counts as a key", hook.Event{Kind: hook.KeyDown}, rawKey},
		{"key typed is ignored", hook.Event{Kind: hook.KeyHold}, rawIgnored},
		{"key up is ignored", hook.Event{Kind: hook.KeyUp}, rawIgnored},
		{"left button down is a click", hook.Event{Kind: hook.MouseDown, Button: 1}, rawClick},
		{"right button down is a click", hook.Event{Kind: hook.MouseDown, Button: 2}, rawClick},
		{"middle button down is a click", hook.Event{Kind: hook.MouseDown, Button: 3}, rawClick},
		{"extra button down is ignored", hook.Event{Kind: hook.MouseDown, Button: 4}, rawIgnored},
		{"mouse up is ignored", hook.Event{Kind: hook.MouseUp, Button: 1}, rawIgnored},
		{"mouse release is ignored", hook.Event{Kind: hook.MouseHold, Button: 1}, rawIgnored},
		{"mouse move is ignored", hook.Event{Kind: hook.MouseMove}, rawIgnored},
		{"mouse drag is ignored", hook.Event{Kind: hook.MouseDrag}, rawIgnored},
		{"mouse wheel is ignored", hook.Event{Kind: hook.MouseWheel}, rawIgnored},
		{"hook disabled is ignored", hook.Event{Kind: hook.HookDisabled}, rawIgnored},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classify(tc.ev).kind; got != tc.want {
				t.Errorf("classify(Kind=%d, Button=%d) = %v, want %v", tc.ev.Kind, tc.ev.Button, got, tc.want)
			}
		})
	}
}

func TestClassify_KeepsCaptureTimestamp(t *testing.T) {
	when := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	if got := classify(hook.Event{Kind: hook.KeyDown, When: when}).when; !got.Equal(when) {
		t.Errorf("when = %v, want %v", got, when)
	}
}

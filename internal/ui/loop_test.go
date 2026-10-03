package ui

import (
	"errors"
	"log/slog"
	"nookbuddy/internal/config"
	"nookbuddy/internal/input"
	"nookbuddy/internal/storage"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type fakeSource struct {
	events chan input.Event
	mu     sync.Mutex
	status input.Status
}

func newFakeSource(status input.Status) *fakeSource {
	return &fakeSource{events: make(chan input.Event, 1024), status: status}
}

func (f *fakeSource) Events() <-chan input.Event { return f.events }

func (f *fakeSource) Status() input.Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

func (f *fakeSource) setStatus(status input.Status) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = status
}

func (f *fakeSource) emit(kind input.EventType, n int) {
	for range n {
		f.events <- input.Event{Type: kind, Timestamp: time.Now()}
	}
}

type scheduled struct {
	after time.Duration
	msg   tea.Msg
}

type scheduler struct {
	mu    sync.Mutex
	calls []scheduled
}

func (s *scheduler) schedule(after time.Duration, msg tea.Msg) tea.Cmd {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, scheduled{after: after, msg: msg})
	return func() tea.Msg { return msg }
}

func (s *scheduler) count(msg tea.Msg) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, call := range s.calls {
		if call.msg == msg {
			n++
		}
	}
	return n
}

func (s *scheduler) last(msg tea.Msg) (time.Duration, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.calls) - 1; i >= 0; i-- {
		if s.calls[i].msg == msg {
			return s.calls[i].after, true
		}
	}
	return 0, false
}

var loopClock = time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)

func loopModel(saver Saver, source InputSource, state storage.PlayerState) (Model, *scheduler) {
	m := NewModel(saver, source, state, threeScreens())
	sched := &scheduler{}
	m.schedule = sched.schedule
	m.now = func() time.Time { return loopClock }
	return m, sched
}

func contains(msgs []tea.Msg, want tea.Msg) bool {
	for _, msg := range msgs {
		if msg == want {
			return true
		}
	}
	return false
}

func TestModel_InitSchedulesInputTickWhenSourceIsPresent(t *testing.T) {
	m, sched := loopModel(nil, newFakeSource(input.Available), storage.PlayerState{})

	if !contains(collect(m.Init()), inputTickMsg{}) {
		t.Fatal("Init did not schedule the input tick")
	}
	if after, _ := sched.last(inputTickMsg{}); after != 250*time.Millisecond {
		t.Errorf("input tick interval = %v, want 250ms", after)
	}
}

func TestModel_InitDoesNotScheduleInputTickWhenSourceIsNil(t *testing.T) {
	m, _ := loopModel(nil, nil, storage.PlayerState{})
	msgs := collect(m.Init())
	if contains(msgs, inputTickMsg{}) || contains(msgs, autosaveTickMsg{}) {
		t.Errorf("Init without source or store scheduled %v, want neither input nor autosave ticks", msgs)
	}
}

func TestModel_InitSchedulesAutosaveWhenSaverIsPresent(t *testing.T) {
	m, sched := loopModel(&countingSaver{}, nil, storage.PlayerState{})

	msgs := collect(m.Init())
	if !contains(msgs, autosaveTickMsg{}) || contains(msgs, inputTickMsg{}) {
		t.Fatalf("Init messages = %v, want only the autosave tick", msgs)
	}
	if after, _ := sched.last(autosaveTickMsg{}); after != config.AutosaveInterval {
		t.Errorf("autosave interval = %v, want 30s", after)
	}
}

func TestModel_HookStatusIsReadAtConstruction(t *testing.T) {
	if m, _ := loopModel(nil, newFakeSource(input.Available), storage.PlayerState{}); m.hookStatus != input.Available {
		t.Error("hook status from an available source was not read")
	}
	if m, _ := loopModel(nil, nil, storage.PlayerState{}); m.hookStatus != input.Unavailable {
		t.Error("a model without a source must report the hook as unavailable")
	}
}

func TestModel_InputTickAlwaysReschedulesItself(t *testing.T) {
	m, sched := loopModel(nil, newFakeSource(input.Available), storage.PlayerState{})

	for i := 1; i <= 3; i++ {
		var cmd tea.Cmd
		m, cmd = send(t, m, inputTickMsg{})
		if !contains(collect(cmd), inputTickMsg{}) {
			t.Fatalf("tick %d on an empty source did not reschedule", i)
		}
	}
	if got := sched.count(inputTickMsg{}); got != 3 {
		t.Errorf("rescheduled %d times, want 3", got)
	}
}

func TestModel_InputTickAppliesPendingEvents(t *testing.T) {
	source := newFakeSource(input.Available)
	source.emit(input.Click, 3)
	source.emit(input.Key, 5)
	m, _ := loopModel(nil, source, storage.PlayerState{})

	m, cmd := send(t, m, inputTickMsg{})
	if m.state.ClicksProgress != 3 || m.state.KeysProgress != 5 {
		t.Errorf("bars = %d/%d, want 3/5", m.state.ClicksProgress, m.state.KeysProgress)
	}
	if !m.lastInputAt.Equal(loopClock) {
		t.Errorf("lastInputAt = %v, want %v", m.lastInputAt, loopClock)
	}
	if !contains(collect(cmd), inputTickMsg{}) {
		t.Error("tick with events did not reschedule")
	}
}

func TestModel_InputTickDrainsAtMostOneBatch(t *testing.T) {
	source := newFakeSource(input.Available)
	source.emit(input.Key, 300)
	m, _ := loopModel(nil, source, storage.PlayerState{})

	m, _ = send(t, m, inputTickMsg{})
	if m.state.KeysProgress != 256 {
		t.Fatalf("first tick applied %d keys, want one batch of 256", m.state.KeysProgress)
	}
	m, _ = send(t, m, inputTickMsg{})
	if m.state.KeysProgress != 300 {
		t.Errorf("second tick reached %d keys, want 300", m.state.KeysProgress)
	}
}

func TestModel_InputTickReadsHookStatus(t *testing.T) {
	source := newFakeSource(input.Available)
	m, _ := loopModel(nil, source, storage.PlayerState{})

	source.setStatus(input.Unavailable)
	m, _ = send(t, m, inputTickMsg{})
	if m.hookStatus != input.Unavailable {
		t.Error("a hook that became unavailable was not picked up on the next tick")
	}
}

func TestModel_InputTickWithoutSourceDoesNothing(t *testing.T) {
	m, _ := loopModel(nil, nil, storage.PlayerState{})
	m, cmd := send(t, m, inputTickMsg{})
	if cmd != nil || m.state.ClicksProgress != 0 {
		t.Error("an input tick without a source did something")
	}
}

func TestModel_InputEventMsgAppliesEventsAndUpdatesLastInputTime(t *testing.T) {
	m, _ := loopModel(nil, nil, storage.PlayerState{UpgradeClicksLevel: 2})

	m, cmd := send(t, m, input.InputEventMsg{Events: []input.Event{{Type: input.Click}}})
	if cmd != nil {
		t.Errorf("applying events returned a command: %v", cmd)
	}
	if m.state.ClicksProgress != 3 {
		t.Errorf("ClicksProgress = %d, want 3 (contribution at Clicks Lv 2)", m.state.ClicksProgress)
	}
	if !m.lastInputAt.Equal(loopClock) || !m.dirty {
		t.Errorf("lastInputAt=%v dirty=%v, want %v/true", m.lastInputAt, m.dirty, loopClock)
	}
}

func TestModel_InputEventMsgSetsHighlightOnProjectCompletion(t *testing.T) {
	logs := captureLogs(t)
	m, _ := loopModel(nil, nil, storage.PlayerState{ClicksProgress: 99, KeysProgress: 300, ProjectsReady: 1})

	m, _ = send(t, m, input.InputEventMsg{Events: []input.Event{{Type: input.Click}, {Type: input.Key}}})
	if m.state.ProjectsReady != 2 {
		t.Fatalf("ProjectsReady = %d, want 2", m.state.ProjectsReady)
	}
	if m.state.ClicksProgress != 0 || m.state.KeysProgress != 1 {
		t.Errorf("bars after completion = %d/%d, want 0/1 (reset, then the next key counts)", m.state.ClicksProgress, m.state.KeysProgress)
	}
	if want := loopClock.Add(time.Second); !m.highlightUntil.Equal(want) {
		t.Errorf("highlightUntil = %v, want %v", m.highlightUntil, want)
	}
	if !logs.has(slog.LevelInfo, "project completed") {
		t.Error("project completion was not logged")
	}
}

func TestModel_NoHighlightWithoutCompletion(t *testing.T) {
	m, _ := loopModel(nil, nil, storage.PlayerState{})
	m, _ = send(t, m, input.InputEventMsg{Events: []input.Event{{Type: input.Click}}})
	if !m.highlightUntil.IsZero() {
		t.Errorf("highlightUntil = %v, want unset without a completed project", m.highlightUntil)
	}
}

func TestModel_InputEventMsgWithEmptyBatchChangesNothing(t *testing.T) {
	start := storage.PlayerState{ClicksProgress: 5, KeysProgress: 6}
	m, _ := loopModel(nil, nil, start)

	m, _ = send(t, m, input.InputEventMsg{})
	if m.state.ClicksProgress != 5 || m.state.KeysProgress != 6 || !m.lastInputAt.IsZero() || !m.highlightUntil.IsZero() || m.dirty {
		t.Errorf("empty batch changed the model: state=%+v lastInput=%v highlight=%v dirty=%v", m.state, m.lastInputAt, m.highlightUntil, m.dirty)
	}
}

func TestModel_AutosaveSavesOnlyWhenDirty(t *testing.T) {
	saver := &countingSaver{}
	m, _ := loopModel(saver, nil, storage.PlayerState{})

	m, cmd := send(t, m, autosaveTickMsg{})
	msgs := collect(cmd)
	if len(saveResults(msgs)) != 0 || saver.calls() != 0 {
		t.Fatal("autosave ran without any change")
	}
	if !contains(msgs, autosaveTickMsg{}) {
		t.Fatal("autosave did not reschedule itself")
	}

	m, _ = send(t, m, input.InputEventMsg{Events: []input.Event{{Type: input.Key}, {Type: input.Key}}})
	m, cmd = send(t, m, autosaveTickMsg{})
	msgs = collect(cmd)
	results := saveResults(msgs)
	if len(results) != 1 || saver.calls() != 1 || saver.states[0].KeysProgress != 2 {
		t.Fatalf("autosave after input: results=%d calls=%d, want one save with KeysProgress 2", len(results), saver.calls())
	}
	if !contains(msgs, autosaveTickMsg{}) {
		t.Error("autosave with a save did not reschedule itself")
	}

	m, _ = send(t, m, results[0])
	if m.dirty {
		t.Error("model still dirty after a successful autosave")
	}
	m, cmd = send(t, m, autosaveTickMsg{})
	if len(saveResults(collect(cmd))) != 0 {
		t.Error("autosave ran again with nothing new to save")
	}
}

func TestModel_AutosaveRetriesAfterFailure(t *testing.T) {
	captureLogs(t)
	saver := &countingSaver{err: errors.New("database is locked")}
	m, _ := loopModel(saver, nil, storage.PlayerState{})
	m, _ = send(t, m, input.InputEventMsg{Events: []input.Event{{Type: input.Click}}})

	m, cmd := send(t, m, autosaveTickMsg{})
	m, _ = send(t, m, saveResults(collect(cmd))[0])
	if !m.dirty || !m.saveFailed {
		t.Fatalf("after a failed autosave dirty=%v saveFailed=%v, want true/true", m.dirty, m.saveFailed)
	}

	saver.err = nil
	m, cmd = send(t, m, autosaveTickMsg{})
	results := saveResults(collect(cmd))
	if len(results) != 1 || saver.calls() != 2 {
		t.Fatalf("next tick did not retry: results=%d calls=%d", len(results), saver.calls())
	}
	m, _ = send(t, m, results[0])
	if m.saveFailed || m.dirty {
		t.Error("warning or dirty flag stayed after the retry succeeded")
	}
}

func TestModel_InputDuringSaveIsSavedByTheNextAutosave(t *testing.T) {
	saver := &countingSaver{}
	m, _ := loopModel(saver, nil, storage.PlayerState{})
	m, _ = send(t, m, input.InputEventMsg{Events: []input.Event{{Type: input.Click}}})

	m, cmd := send(t, m, autosaveTickMsg{})
	m, _ = send(t, m, input.InputEventMsg{Events: []input.Event{{Type: input.Click}}})
	m, _ = send(t, m, saveResults(collect(cmd))[0])
	if !m.dirty {
		t.Fatal("input that arrived during a save was forgotten")
	}

	_, cmd = send(t, m, autosaveTickMsg{})
	collect(cmd)
	if saver.calls() != 2 || saver.states[1].ClicksProgress != 2 {
		t.Errorf("second autosave wrote %+v, want ClicksProgress 2", saver.states)
	}
}

func TestModel_AutosaveIsSkippedWhileQuitting(t *testing.T) {
	saver := &countingSaver{}
	m, _ := loopModel(saver, nil, storage.PlayerState{})
	m, _ = send(t, m, input.InputEventMsg{Events: []input.Event{{Type: input.Click}}})
	m, final := send(t, m, key("q"))

	_, cmd := send(t, m, autosaveTickMsg{})
	collect(final)
	collect(cmd)
	if saver.calls() != 1 {
		t.Errorf("Save calls = %d, want only the final save", saver.calls())
	}
}

func TestModel_UpgradeLevelAppliesToTheNextEvent(t *testing.T) {
	m, _ := loopModel(nil, nil, storage.PlayerState{})
	m, _ = send(t, m, input.InputEventMsg{Events: []input.Event{{Type: input.Click}}})
	m.state.UpgradeClicksLevel = 1
	m, _ = send(t, m, input.InputEventMsg{Events: []input.Event{{Type: input.Click}}})
	if m.state.ClicksProgress != 3 {
		t.Errorf("ClicksProgress = %d, want 1 + 2 after the level changed", m.state.ClicksProgress)
	}
}

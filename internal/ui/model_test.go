package ui

import (
	"context"
	"errors"
	"log/slog"
	"nookbuddy/internal/storage"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type fakeScreen struct {
	title         string
	items         int
	activateCmd   tea.Cmd
	panicOnEnter  bool
	panicOnView   bool
	goldDelta     int
	seenGold      []int
	mu            sync.Mutex
	activatedWith []int
}

func (f *fakeScreen) Title() string  { return f.title }
func (f *fakeScreen) ItemCount() int { return f.items }

func (f *fakeScreen) View(selected int, state storage.PlayerState) string {
	if f.panicOnView {
		panic("view exploded")
	}
	return f.title + " body"
}

func (f *fakeScreen) Activate(selected int, state storage.PlayerState) (storage.PlayerState, bool, tea.Cmd) {
	if f.panicOnEnter {
		panic("activate exploded")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.activatedWith = append(f.activatedWith, selected)
	f.seenGold = append(f.seenGold, state.Gold)
	if f.goldDelta != 0 {
		state.Gold += f.goldDelta
		return state, true, f.activateCmd
	}
	return state, false, f.activateCmd
}

func (f *fakeScreen) activations() []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int(nil), f.activatedWith...)
}

func threeScreens() []Screen {
	return []Screen{
		&fakeScreen{title: "Sell", items: 5},
		&fakeScreen{title: "Upgrades", items: 3},
		&fakeScreen{title: "Customize", items: 0},
	}
}

func key(name string) tea.KeyMsg {
	switch name {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
}

func send(t *testing.T, m Model, msgs ...tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	for _, msg := range msgs {
		var next tea.Model
		next, cmd = m.Update(msg)
		m = next.(Model)
	}
	return m, cmd
}

func press(t *testing.T, m Model, keys ...string) Model {
	t.Helper()
	for _, k := range keys {
		m, _ = send(t, m, key(k))
	}
	return m
}

func newTestModel() Model {
	return NewModel(nil, nil, storage.PlayerState{}, threeScreens())
}

func openTempStore(t *testing.T) (*storage.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nookbuddy.db")
	store, _, err := storage.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, path
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

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

func (h *recordingHandler) has(level slog.Level, contains string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.records {
		if r.Level == level && strings.Contains(r.Message, contains) {
			return true
		}
	}
	return false
}

func captureLogs(t *testing.T) *recordingHandler {
	t.Helper()
	handler := &recordingHandler{}
	previous := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return handler
}

func TestModel_LaunchesOnSell(t *testing.T) {
	m := newTestModel()
	if m.active != 0 || m.activeScreen().Title() != "Sell" {
		t.Errorf("launch screen = %q, want Sell", m.activeScreen().Title())
	}
	if m.selected != 0 {
		t.Errorf("launch selection = %d, want 0", m.selected)
	}
}

func TestModel_NumberKeysSwitchScreenAndResetSelection(t *testing.T) {
	m := press(t, newTestModel(), "down", "down")
	if m.selected != 2 {
		t.Fatalf("precondition: selected = %d, want 2", m.selected)
	}

	for _, tc := range []struct {
		key   string
		title string
	}{{"2", "Upgrades"}, {"3", "Customize"}, {"1", "Sell"}} {
		m = press(t, m, "down")
		m = press(t, m, tc.key)
		if got := m.activeScreen().Title(); got != tc.title {
			t.Errorf("after %q active = %q, want %q", tc.key, got, tc.title)
		}
		if m.selected != 0 {
			t.Errorf("after %q selected = %d, want 0", tc.key, m.selected)
		}
	}
}

func TestModel_TabAndShiftTabCycleBothDirections(t *testing.T) {
	m := newTestModel()
	var seen []string
	for range 3 {
		m = press(t, m, "tab")
		seen = append(seen, m.activeScreen().Title())
	}
	if strings.Join(seen, ",") != "Upgrades,Customize,Sell" {
		t.Errorf("tab cycle = %v, want Upgrades,Customize,Sell", seen)
	}

	m = press(t, m, "shift+tab")
	if got := m.activeScreen().Title(); got != "Customize" {
		t.Errorf("shift+tab from Sell = %q, want Customize", got)
	}
}

func TestModel_TabResetsSelection(t *testing.T) {
	m := press(t, newTestModel(), "down", "tab")
	if m.selected != 0 {
		t.Errorf("selected after tab = %d, want 0", m.selected)
	}
}

func TestModel_UnknownScreenKeyFourIsIgnored(t *testing.T) {
	before := press(t, newTestModel(), "2", "down")
	after := press(t, before, "4")
	if after.active != before.active || after.selected != before.selected {
		t.Errorf("4 changed state: active %d→%d, selected %d→%d", before.active, after.active, before.selected, after.selected)
	}
}

func TestModel_UpDownMoveAndClampSelection(t *testing.T) {
	m := newTestModel()

	m = press(t, m, "down", "down", "down", "down", "down", "down", "down")
	if m.selected != 4 {
		t.Errorf("selected after 7× down on 5 items = %d, want 4 (no wrap)", m.selected)
	}

	m = press(t, m, "up", "up", "up", "up", "up", "up", "up")
	if m.selected != 0 {
		t.Errorf("selected after 7× up = %d, want 0 (no wrap)", m.selected)
	}
}

func TestModel_JAndKMoveLikeArrows(t *testing.T) {
	m := press(t, newTestModel(), "j", "j", "k")
	if m.selected != 1 {
		t.Errorf("selected after j j k = %d, want 1", m.selected)
	}
}

func TestModel_SelectionStaysAtZeroOnEmptyScreen(t *testing.T) {
	m := press(t, newTestModel(), "3", "down", "down", "up")
	if m.selected != 0 {
		t.Errorf("selected on an empty screen = %d, want 0", m.selected)
	}
}

func TestModel_EnterCallsActivateOnActiveScreen(t *testing.T) {
	screens := threeScreens()
	upgrades := screens[1].(*fakeScreen)
	upgrades.activateCmd = func() tea.Msg { return "bought" }
	m := NewModel(nil, nil, storage.PlayerState{}, screens)

	m = press(t, m, "2", "down", "down")
	_, cmd := send(t, m, key("enter"))

	if got := upgrades.activations(); len(got) != 1 || got[0] != 2 {
		t.Fatalf("Activate calls = %v, want [2]", got)
	}
	if cmd == nil || cmd() != "bought" {
		t.Error("the screen's command was not returned to Bubbletea")
	}
	if got := screens[0].(*fakeScreen).activations(); len(got) != 0 {
		t.Errorf("inactive screen was activated: %v", got)
	}
}

func TestModel_EnterOnEmptyScreenDoesNotActivate(t *testing.T) {
	screens := threeScreens()
	m := press(t, NewModel(nil, nil, storage.PlayerState{}, screens), "3", "enter")
	if got := screens[2].(*fakeScreen).activations(); len(got) != 0 {
		t.Errorf("Activate called on an empty screen: %v", got)
	}
	_ = m
}

func TestModel_QuitKeySetsQuittingAndReturnsSaveCmd(t *testing.T) {
	store, path := openTempStore(t)
	state := storage.PlayerState{Gold: 77, ClicksProgress: 12, KeysProgress: 34, ProjectsReady: 2, UpgradeValueLevel: 1, OwnedCosmetics: []string{"rug"}}
	m := NewModel(store, nil, state, threeScreens())

	m, cmd := send(t, m, key("q"))
	if !m.quitting {
		t.Fatal("quitting = false after q")
	}
	if cmd == nil {
		t.Fatal("q returned no command, want the save command")
	}

	saved, ok := cmd().(saveResultMsg)
	if !ok {
		t.Fatalf("save command returned %T, want saveResultMsg", cmd())
	}
	if saved.err != nil {
		t.Fatalf("save failed: %v", saved.err)
	}

	_, next := send(t, m, saved)
	if !isQuit(next) {
		t.Error("a finished save did not lead to tea.Quit")
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reopened, _, err := storage.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	got, err := reopened.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Gold != 77 || got.ClicksProgress != 12 || got.KeysProgress != 34 || got.ProjectsReady != 2 ||
		got.UpgradeValueLevel != 1 || len(got.OwnedCosmetics) != 1 || got.OwnedCosmetics[0] != "rug" {
		t.Errorf("persisted state = %+v, want the exact in-memory state %+v", got, state)
	}
}

func TestModel_CtrlCTriggersSameQuitPathAsQ(t *testing.T) {
	store, _ := openTempStore(t)
	m := NewModel(store, nil, storage.PlayerState{Gold: 5}, threeScreens())

	m, cmd := send(t, m, key("ctrl+c"))
	if !m.quitting {
		t.Fatal("quitting = false after ctrl+c")
	}
	saved, ok := cmd().(saveResultMsg)
	if !ok || saved.err != nil {
		t.Fatalf("ctrl+c save = %#v, want a successful saveResultMsg", cmd())
	}
	if _, next := send(t, m, saved); !isQuit(next) {
		t.Error("ctrl+c did not lead to tea.Quit")
	}
}

func TestModel_KeysAreIgnoredWhileSaving(t *testing.T) {
	store, _ := openTempStore(t)
	m, _ := send(t, NewModel(store, nil, storage.PlayerState{}, threeScreens()), key("q"))

	after, cmd := send(t, m, key("2"))
	if after.active != 0 || cmd != nil {
		t.Error("a key press during the final save changed state or issued a command")
	}
	if _, cmd = send(t, m, key("q")); cmd != nil {
		t.Error("a second q during the final save started another save")
	}
}

type blockingSaver struct {
	release chan struct{}
}

func (b blockingSaver) Save(context.Context, storage.PlayerState) (storage.SaveResult, error) {
	<-b.release
	return storage.SaveResult{OK: true}, nil
}

func TestModel_SaveTimeoutStillQuits(t *testing.T) {
	logs := captureLogs(t)
	saver := blockingSaver{release: make(chan struct{})}
	t.Cleanup(func() { close(saver.release) })

	m := NewModel(saver, nil, storage.PlayerState{}, threeScreens())
	m.saveTimeout = 50 * time.Millisecond

	m, cmd := send(t, m, key("q"))
	began := time.Now()
	saved := cmd().(saveResultMsg)
	if elapsed := time.Since(began); elapsed > time.Second {
		t.Fatalf("save command took %v, want it bounded by the timeout", elapsed)
	}
	if !errors.Is(saved.err, context.DeadlineExceeded) {
		t.Errorf("save error = %v, want context.DeadlineExceeded", saved.err)
	}

	if _, next := send(t, m, saved); !isQuit(next) {
		t.Error("a timed-out save did not lead to tea.Quit")
	}
	if !logs.has(slog.LevelError, "save failed") {
		t.Error("the failed final save was not logged")
	}
}

func TestModel_DefaultSaveTimeoutIsTwoSeconds(t *testing.T) {
	if got := newTestModel().saveTimeout; got != 2*time.Second {
		t.Errorf("saveTimeout = %v, want 2s", got)
	}
}

func TestModel_QuitWithoutStoreQuitsImmediately(t *testing.T) {
	m, cmd := send(t, newTestModel(), key("q"))
	if !m.quitting || !isQuit(cmd) {
		t.Error("q without a store did not quit immediately")
	}
}

func TestModel_WindowSizeMsgUpdatesDimensions(t *testing.T) {
	m, _ := send(t, newTestModel(), tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.width != 120 || m.height != 40 {
		t.Errorf("size = %d×%d, want 120×40", m.width, m.height)
	}
}

func TestModel_ViewRendersActiveScreenAndFooter(t *testing.T) {
	m, _ := send(t, newTestModel(), tea.WindowSizeMsg{Width: 120, Height: 30})
	out := plain(m.View())

	for _, want := range []string{"Sell body", "Sell │ Upgrades │ Customize", "Room view coming soon", "[1] Sell", "q quit"} {
		if !strings.Contains(out, want) {
			t.Errorf("view is missing %q", want)
		}
	}
	if strings.Contains(out, "Upgrades body") {
		t.Error("view renders an inactive screen")
	}
}

func TestModel_ViewFitsTheTerminal(t *testing.T) {
	m, _ := send(t, newTestModel(), tea.WindowSizeMsg{Width: 120, Height: 30})
	lines := strings.Split(plain(m.View()), "\n")
	if len(lines) != 30 {
		t.Errorf("view has %d lines, want 30", len(lines))
	}
	for i, line := range lines {
		if w := len([]rune(line)); w > 120 {
			t.Errorf("line %d is %d columns wide, want at most 120", i, w)
		}
	}
}

func TestModel_ViewShowsSavingWhileQuitting(t *testing.T) {
	store, _ := openTempStore(t)
	m, _ := send(t, NewModel(store, nil, storage.PlayerState{}, threeScreens()), key("q"))
	out := plain(m.View())
	if !strings.Contains(out, "Saving…") || strings.Contains(out, "q quit") {
		t.Error("footer does not switch to Saving… during the final save")
	}
}

func TestModel_BatchedRunesAreHandledOneByOne(t *testing.T) {
	m, _ := send(t, newTestModel(), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2jj")})
	if got := m.activeScreen().Title(); got != "Upgrades" || m.selected != 2 {
		t.Errorf("after batched \"2jj\": active=%q selected=%d, want Upgrades/2", got, m.selected)
	}
}

func TestModel_BatchedRunesStopAfterQuit(t *testing.T) {
	m, cmd := send(t, newTestModel(), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q2")})
	if !m.quitting || m.active != 0 {
		t.Errorf("after batched \"q2\": quitting=%v active=%d, want true/0", m.quitting, m.active)
	}
	if !isQuit(cmd) {
		t.Error("batched q did not quit")
	}
}

func collect(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, inner := range batch {
			out = append(out, collect(inner)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

func saveResults(msgs []tea.Msg) []saveResultMsg {
	var out []saveResultMsg
	for _, msg := range msgs {
		if result, ok := msg.(saveResultMsg); ok {
			out = append(out, result)
		}
	}
	return out
}

type countingSaver struct {
	mu        sync.Mutex
	states    []storage.PlayerState
	deadlines []time.Time
	err       error
}

func (c *countingSaver) Save(ctx context.Context, state storage.PlayerState) (storage.SaveResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	deadline, _ := ctx.Deadline()
	c.states = append(c.states, state)
	c.deadlines = append(c.deadlines, deadline)
	return storage.SaveResult{OK: c.err == nil}, c.err
}

func (c *countingSaver) calls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.states)
}

func goldScreens(delta int) []Screen {
	screens := threeScreens()
	screens[0].(*fakeScreen).goldDelta = delta
	return screens
}

func TestModel_ActivateReceivesCurrentState(t *testing.T) {
	screens := threeScreens()
	m := NewModel(nil, nil, storage.PlayerState{Gold: 33}, screens)
	_, _ = send(t, m, key("enter"))
	if got := screens[0].(*fakeScreen).seenGold; len(got) != 1 || got[0] != 33 {
		t.Errorf("Activate saw gold %v, want [33]", got)
	}
}

func TestModel_EnterWithChangedStateTriggersSaveCmd(t *testing.T) {
	store, path := openTempStore(t)
	m := NewModel(store, nil, storage.PlayerState{Gold: 60}, goldScreens(-50))

	m, cmd := send(t, m, key("enter"))
	if m.state.Gold != 10 {
		t.Fatalf("model gold = %d, want 10 right after the change", m.state.Gold)
	}
	results := saveResults(collect(cmd))
	if len(results) != 1 || results[0].err != nil {
		t.Fatalf("save results = %+v, want one successful save", results)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reopened, _, err := storage.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	got, err := reopened.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Gold != 10 {
		t.Errorf("persisted gold = %d, want 10 without quitting", got.Gold)
	}
}

func TestModel_EnterWithUnchangedStateDoesNotTriggerSaveCmd(t *testing.T) {
	saver := &countingSaver{}
	m := NewModel(saver, nil, storage.PlayerState{Gold: 60}, goldScreens(0))

	_, cmd := send(t, m, key("enter"))
	collect(cmd)
	if saver.calls() != 0 {
		t.Errorf("Save called %d times for an unchanged state, want 0", saver.calls())
	}
}

func TestModel_SaveAfterChangeDoesNotQuit(t *testing.T) {
	saver := &countingSaver{}
	m, cmd := send(t, NewModel(saver, nil, storage.PlayerState{Gold: 60}, goldScreens(-50)), key("enter"))

	m, next := send(t, m, saveResults(collect(cmd))[0])
	if isQuit(next) || m.quitting {
		t.Fatal("finishing a purchase save quit the app")
	}
	if m.saving {
		t.Error("model still marked as saving after the result arrived")
	}
}

func TestModel_SavesAreSerializedAndUseTheLatestState(t *testing.T) {
	saver := &countingSaver{}
	m := NewModel(saver, nil, storage.PlayerState{Gold: 100}, goldScreens(-10))

	m, first := send(t, m, key("enter"))
	m, second := send(t, m, key("enter"))
	m, third := send(t, m, key("enter"))
	if len(saveResults(collect(second))) != 0 || len(saveResults(collect(third))) != 0 {
		t.Fatal("a second save started while one was in flight")
	}
	if !m.pendingSave {
		t.Fatal("changes during an in-flight save were not marked pending")
	}

	m, follow := send(t, m, saveResults(collect(first))[0])
	results := saveResults(collect(follow))
	if len(results) != 1 {
		t.Fatalf("follow-up saves = %d, want exactly 1", len(results))
	}
	if saver.calls() != 2 {
		t.Fatalf("Save calls = %d, want 2", saver.calls())
	}
	if got := saver.states[1].Gold; got != 70 {
		t.Errorf("follow-up save wrote gold %d, want the latest 70", got)
	}

	m, last := send(t, m, results[0])
	if last != nil || m.saving || m.pendingSave {
		t.Error("model kept saving after the latest state was written")
	}
}

func TestModel_QuitWaitsForInFlightSave(t *testing.T) {
	saver := &countingSaver{}
	m, purchase := send(t, NewModel(saver, nil, storage.PlayerState{Gold: 60}, goldScreens(-50)), key("enter"))

	m, cmd := send(t, m, key("q"))
	if !m.quitting || cmd != nil {
		t.Fatal("q during an in-flight save started a concurrent final save")
	}

	m, final := send(t, m, saveResults(collect(purchase))[0])
	finalMsgs := collect(final)
	for _, msg := range finalMsgs {
		if _, quit := msg.(tea.QuitMsg); quit {
			t.Fatal("app quit before the final save")
		}
	}
	results := saveResults(finalMsgs)
	if len(results) != 1 || saver.calls() != 2 {
		t.Fatalf("final save did not run: results=%d calls=%d", len(results), saver.calls())
	}
	if !saver.deadlines[1].Equal(m.quitDeadline) && saver.deadlines[1].After(m.quitDeadline) {
		t.Errorf("final save deadline %v is after the quit deadline %v", saver.deadlines[1], m.quitDeadline)
	}

	if _, done := send(t, m, results[0]); !isQuit(done) {
		t.Error("app did not quit after the final save")
	}
}

func TestModel_FailedSaveShowsWarningUntilNextSuccess(t *testing.T) {
	logs := captureLogs(t)
	saver := &countingSaver{err: errors.New("disk is read-only")}
	m, cmd := send(t, NewModel(saver, nil, storage.PlayerState{Gold: 60}, goldScreens(-5)), key("enter"))

	m, _ = send(t, m, saveResults(collect(cmd))[0])
	if !m.saveFailed || !strings.Contains(plain(m.View()), "Last save failed — retrying") {
		t.Fatal("a failed save did not show the warning")
	}
	if m.state.Gold != 55 {
		t.Errorf("in-memory gold = %d, want 55 kept after a failed save", m.state.Gold)
	}
	if !logs.has(slog.LevelError, "save failed") {
		t.Error("the failed save was not logged")
	}

	saver.err = nil
	m, cmd = send(t, m, key("enter"))
	m, _ = send(t, m, saveResults(collect(cmd))[0])
	if m.saveFailed || strings.Contains(plain(m.View()), "Last save failed") {
		t.Error("the warning stayed after a successful save")
	}
}

func TestModel_ChangeWithoutStoreKeepsStateInMemory(t *testing.T) {
	m, cmd := send(t, NewModel(nil, nil, storage.PlayerState{Gold: 60}, goldScreens(-50)), key("enter"))
	if m.state.Gold != 10 || m.saving {
		t.Errorf("memory-only change: gold=%d saving=%v, want 10/false", m.state.Gold, m.saving)
	}
	if len(saveResults(collect(cmd))) != 0 {
		t.Error("memory-only mode tried to save")
	}
}

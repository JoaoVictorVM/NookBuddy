package screens

import (
	"context"
	"log/slog"
	"math"
	"nookbuddy/internal/game"
	"nookbuddy/internal/input"
	"nookbuddy/internal/storage"
	"nookbuddy/internal/ui"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestSell() (*Sell, *fakeClock) {
	clock := &fakeClock{at: time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)}
	s := NewSell()
	s.now = clock.now
	return s, clock
}

func sellAllLine(view string) string {
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(plain(line), "Sell all") {
			return line
		}
	}
	return ""
}

func TestSell_TitleAndItemCount(t *testing.T) {
	s := NewSell()
	if s.Title() != "Sell" {
		t.Errorf("Title() = %q, want Sell", s.Title())
	}
	if s.ItemCount() != 1 {
		t.Errorf("ItemCount() = %d, want 1 even with zero projects", s.ItemCount())
	}
}

func TestSell_ViewShowsProjectsValueAndTotal(t *testing.T) {
	s, _ := newTestSell()
	out := plain(s.View(0, storage.PlayerState{ProjectsReady: 3, Gold: 8}))
	for _, want := range []string{"Gold: 8", "Projects ready: 3", "Value per project: 10 gold", "Total: 3 × 10 = 30 gold", "> Sell all"} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q:\n%s", want, out)
		}
	}
}

func TestSell_ViewTotalFollowsValueUpgrade(t *testing.T) {
	s, _ := newTestSell()
	out := plain(s.View(0, storage.PlayerState{ProjectsReady: 2, UpgradeValueLevel: 2}))
	if !strings.Contains(out, "Value per project: 20 gold") || !strings.Contains(out, "Total: 2 × 20 = 40 gold") {
		t.Errorf("view does not use the Value level 2 sale value:\n%s", out)
	}
}

func TestSell_ViewRendersSellAllDimWhenNoProjects(t *testing.T) {
	forceANSI(t)
	s, _ := newTestSell()

	empty := sellAllLine(s.View(0, storage.PlayerState{}))
	if empty != ui.Dim.Bold(true).Render("> Sell all") {
		t.Errorf("Sell all with 0 projects = %q, want dim", empty)
	}
	ready := sellAllLine(s.View(0, storage.PlayerState{ProjectsReady: 1}))
	if ready != ui.SelectedRow.Render("> Sell all") {
		t.Errorf("Sell all with projects = %q, want the selected-row style", ready)
	}
	if !slices.Contains(attributes(ready), "1") {
		t.Error("enabled Sell all is not bold")
	}
}

func TestSell_ActivateSellsEverything(t *testing.T) {
	s, _ := newTestSell()
	next, changed, cmd := s.Activate(0, storage.PlayerState{ProjectsReady: 3})
	if !changed || cmd == nil {
		t.Fatalf("changed=%v cmd=%v, want a sale with a refresh command", changed, cmd)
	}
	if next.Gold != 30 || next.ProjectsReady != 0 {
		t.Errorf("after selling: gold=%d projects=%d, want 30/0", next.Gold, next.ProjectsReady)
	}
}

func TestSell_ActivateWithZeroProjectsIsANoOp(t *testing.T) {
	s, _ := newTestSell()
	state := storage.PlayerState{Gold: 44}
	next, changed, cmd := s.Activate(0, state)
	if changed || next.Gold != 44 || next.ProjectsReady != 0 {
		t.Errorf("empty sale changed state: changed=%v next=%+v", changed, next)
	}
	if cmd == nil {
		t.Error("no command returned to expire the message")
	}
	if !strings.Contains(plain(s.View(0, next)), "No projects ready") {
		t.Error("view does not show No projects ready")
	}
}

func TestSell_ConfirmationPluralizes(t *testing.T) {
	s, _ := newTestSell()
	next, _, _ := s.Activate(0, storage.PlayerState{ProjectsReady: 1})
	if !strings.Contains(plain(s.View(0, next)), "Sold 1 project for 10 gold") {
		t.Error("singular confirmation missing")
	}
	next, _, _ = s.Activate(0, storage.PlayerState{ProjectsReady: 3})
	if !strings.Contains(plain(s.View(0, next)), "Sold 3 projects for 30 gold") {
		t.Error("plural confirmation missing")
	}
}

type warnCounter struct {
	mu    sync.Mutex
	warns int
}

func (w *warnCounter) Enabled(context.Context, slog.Level) bool { return true }

func (w *warnCounter) Handle(_ context.Context, r slog.Record) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if r.Level == slog.LevelWarn {
		w.warns++
	}
	return nil
}

func (w *warnCounter) WithAttrs([]slog.Attr) slog.Handler { return w }

func (w *warnCounter) WithGroup(string) slog.Handler { return w }

func (w *warnCounter) count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.warns
}

func countWarnings(t *testing.T) *warnCounter {
	t.Helper()
	counter := &warnCounter{}
	previous := slog.Default()
	slog.SetDefault(slog.New(counter))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return counter
}

func TestSell_CappedSaleIsLoggedAndReportsTheGoldActuallyAdded(t *testing.T) {
	warnings := countWarnings(t)
	s, _ := newTestSell()

	s.Activate(0, storage.PlayerState{ProjectsReady: 2})
	if warnings.count() != 0 {
		t.Fatal("an ordinary sale was logged as capped")
	}

	next, _, _ := s.Activate(0, storage.PlayerState{Gold: math.MaxInt - 5, ProjectsReady: 3})
	if next.Gold != math.MaxInt {
		t.Errorf("gold = %d, want capped at the maximum", next.Gold)
	}
	if !strings.Contains(plain(s.View(0, next)), "Sold 3 projects for 5 gold") {
		t.Error("confirmation must report the gold actually added")
	}
	if warnings.count() != 1 {
		t.Errorf("capped sale logged %d warnings, want 1", warnings.count())
	}
}

func TestSell_MessageExpiresAfterTwoSeconds(t *testing.T) {
	s, clock := newTestSell()
	next, _, _ := s.Activate(0, storage.PlayerState{ProjectsReady: 2})

	clock.at = clock.at.Add(1999 * time.Millisecond)
	if !strings.Contains(plain(s.View(0, next)), "Sold 2 projects") {
		t.Fatal("message disappeared before 2 seconds")
	}
	clock.at = clock.at.Add(time.Millisecond)
	if strings.Contains(plain(s.View(0, next)), "Sold") {
		t.Error("message still shown after 2 seconds")
	}
}

func TestSell_DoubleEnterSellsOnlyOnce(t *testing.T) {
	s, _ := newTestSell()
	first, firstChanged, _ := s.Activate(0, storage.PlayerState{ProjectsReady: 4, Gold: 1})
	second, secondChanged, _ := s.Activate(0, first)
	if !firstChanged || secondChanged {
		t.Fatalf("changed = %v then %v, want true then false", firstChanged, secondChanged)
	}
	if second.Gold != 41 {
		t.Errorf("gold after two presses = %d, want 41 credited once", second.Gold)
	}
	if !strings.Contains(plain(s.View(0, second)), "No projects ready") {
		t.Error("the second press must land on No projects ready")
	}
}

func TestSell_ActivateOutOfRangeIsIgnored(t *testing.T) {
	s, _ := newTestSell()
	for _, selected := range []int{-1, 1} {
		if _, changed, cmd := s.Activate(selected, storage.PlayerState{ProjectsReady: 5}); changed || cmd != nil {
			t.Errorf("Activate(%d) changed=%v cmd=%v, want an ignored press", selected, changed, cmd)
		}
	}
}

func TestSell_ProjectsFromCompletedWorkDriveTheTotal(t *testing.T) {
	state := storage.PlayerState{}
	for range 100 {
		state = game.ApplyEvent(state, input.Event{Type: input.Click})
	}
	for range 300 {
		state = game.ApplyEvent(state, input.Event{Type: input.Key})
	}
	s, _ := newTestSell()
	out := plain(s.View(0, state))
	if !strings.Contains(out, "Projects ready: 1") || !strings.Contains(out, "Total: 1 × 10 = 10 gold") {
		t.Errorf("a completed project does not drive the total:\n%s", out)
	}
}

func TestSell_TotalUpdatesLiveWhileTheScreenIsOpen(t *testing.T) {
	model := tea.Model(ui.NewModel(&recordingSaver{}, nil, storage.PlayerState{ClicksProgress: 99, KeysProgress: 300}, []ui.Screen{NewSell(), NewUpgrades(), NewCustomize()}))
	model, _ = drive(model, tea.WindowSizeMsg{Width: 120, Height: 30})
	if !strings.Contains(plain(model.View()), "Total: 0 × 10 = 0 gold") {
		t.Fatal("total is not zero before the project completes")
	}

	model, _ = drive(model, input.InputEventMsg{Events: []input.Event{{Type: input.Click}}})
	if !strings.Contains(plain(model.View()), "Total: 1 × 10 = 10 gold") {
		t.Error("total did not update on the render after the project completed")
	}
}

func TestSell_SaleUpdatesTheRoomGoldAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nookbuddy.db")
	store, _, err := storage.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}

	model := tea.Model(ui.NewModel(store, nil, storage.PlayerState{Gold: 5, ProjectsReady: 3, UpgradeValueLevel: 1}, []ui.Screen{NewSell(), NewUpgrades(), NewCustomize()}))
	model, _ = drive(model, tea.WindowSizeMsg{Width: 120, Height: 30})
	model, sale := drive(model, tea.KeyMsg{Type: tea.KeyEnter})

	frame := plain(model.View())
	if !strings.Contains(frame, "Sold 3 projects for 45 gold") {
		t.Fatalf("confirmation missing:\n%s", frame)
	}
	if strings.Count(frame, "Gold: 50") < 2 || !strings.Contains(frame, "Projects ready: 0") {
		t.Errorf("the room panel and the screen must both show 50 gold and 0 projects on the same frame:\n%s", frame)
	}

	settle(model, sale, 300*time.Millisecond)
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	relaunched, _, err := storage.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = relaunched.Close() })
	loaded, err := relaunched.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Gold != 50 || loaded.ProjectsReady != 0 {
		t.Errorf("after relaunch: gold=%d projects=%d, want 50/0", loaded.Gold, loaded.ProjectsReady)
	}
}

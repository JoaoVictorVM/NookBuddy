package screens

import (
	"context"
	"nookbuddy/internal/storage"
	"nookbuddy/internal/ui"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestCustomize() (*Customize, *fakeClock) {
	clock := &fakeClock{at: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)}
	c := NewCustomize()
	c.now = clock.now
	return c, clock
}

func cosmeticRows(view string) []string {
	var rows []string
	for _, line := range strings.Split(view, "\n") {
		text := plain(line)
		for _, name := range []string{"Window", "Bookshelf", "Flower", "Painting", "Rug"} {
			if strings.Contains(text, name+" ") && !strings.Contains(text, "added to your room") {
				rows = append(rows, line)
			}
		}
	}
	return rows
}

func TestCustomize_TitleAndItemCount(t *testing.T) {
	c := NewCustomize()
	if c.Title() != "Customize" {
		t.Errorf("Title() = %q, want Customize", c.Title())
	}
	if c.ItemCount() != 5 {
		t.Errorf("ItemCount() = %d, want 5", c.ItemCount())
	}
}

func TestCustomize_ViewListsAllFiveItemsInStorageOrder(t *testing.T) {
	c, _ := newTestCustomize()
	rows := cosmeticRows(c.View(0, storage.PlayerState{}))
	want := []string{"Window", "Bookshelf", "Flower", "Painting", "Rug"}
	if len(rows) != len(want) {
		t.Fatalf("rendered %d rows, want %d", len(rows), len(want))
	}
	for i, name := range want {
		if !strings.Contains(plain(rows[i]), name) || !strings.Contains(plain(rows[i]), "100 gold") {
			t.Errorf("row %d = %q, want %s at 100 gold", i, plain(rows[i]), name)
		}
	}
}

func TestCustomize_ViewShowsGoldBalance(t *testing.T) {
	c, _ := newTestCustomize()
	if !strings.Contains(plain(c.View(0, storage.PlayerState{Gold: 321})), "Gold: 321") {
		t.Error("view does not show the gold balance")
	}
}

func TestCustomize_ViewShowsOwnedItemsWithCheckmarkAndNoPrice(t *testing.T) {
	c, _ := newTestCustomize()
	rows := cosmeticRows(c.View(0, storage.PlayerState{Gold: 500, OwnedCosmetics: []string{"bookshelf"}}))

	bookshelf := plain(rows[1])
	if !strings.Contains(bookshelf, "✓ Bookshelf") || !strings.Contains(bookshelf, "Owned") || strings.Contains(bookshelf, "gold") {
		t.Errorf("owned row = %q, want ✓, Owned and no price", bookshelf)
	}
	for _, i := range []int{0, 2, 3, 4} {
		if row := plain(rows[i]); !strings.Contains(row, "100 gold") || strings.Contains(row, "✓") {
			t.Errorf("unowned row %d = %q, want the price and no checkmark", i, row)
		}
	}
}

func TestCustomize_OwnedRowsRenderDim(t *testing.T) {
	forceANSI(t)
	c, _ := newTestCustomize()
	rows := cosmeticRows(c.View(0, storage.PlayerState{Gold: 500, OwnedCosmetics: []string{"flower"}}))

	if rows[2] != ui.Dim.Render(plain(rows[2])) {
		t.Errorf("owned Flower row is not dim: %q", rows[2])
	}
	if strings.Contains(rows[3], "\x1b[") {
		t.Errorf("affordable unowned Painting row is styled: %q", rows[3])
	}
}

func TestCustomize_ViewShowsUnaffordableRowsDim(t *testing.T) {
	forceANSI(t)
	c, _ := newTestCustomize()
	rows := cosmeticRows(c.View(5, storage.PlayerState{Gold: 50}))

	for i, row := range rows {
		if row != ui.Dim.Render(plain(row)) {
			t.Errorf("unaffordable row %d is not dim: %q", i, row)
		}
	}
}

func TestCustomize_AffordabilityFollowsGold(t *testing.T) {
	forceANSI(t)
	c, _ := newTestCustomize()

	if row := cosmeticRows(c.View(5, storage.PlayerState{Gold: 99}))[0]; !strings.Contains(row, "\x1b[") {
		t.Error("Window with 99 gold is not dim")
	}
	if row := cosmeticRows(c.View(5, storage.PlayerState{Gold: 100}))[0]; strings.Contains(row, "\x1b[") {
		t.Error("Window with 100 gold is still dim")
	}
}

func TestCustomize_ViewMarksSelectedRow(t *testing.T) {
	forceANSI(t)
	c, _ := newTestCustomize()
	rows := cosmeticRows(c.View(2, storage.PlayerState{Gold: 500}))

	for i, row := range rows {
		if marked := strings.HasPrefix(plain(row), "> "); marked != (i == 2) {
			t.Errorf("row %d selection marker = %v, want %v", i, marked, i == 2)
		}
	}
	if !slices.Contains(attributes(rows[2]), "1") {
		t.Error("selected row is not bold")
	}
}

func TestCustomize_ActivatePurchasesWhenAffordableAndUnowned(t *testing.T) {
	c, _ := newTestCustomize()
	state := storage.PlayerState{Gold: 150, OwnedCosmetics: []string{"rug"}}

	next, changed, cmd := c.Activate(0, state)
	if !changed || cmd == nil {
		t.Fatalf("changed=%v cmd=%v, want a purchase with a refresh command", changed, cmd)
	}
	if next.Gold != 50 {
		t.Errorf("Gold = %d, want 50", next.Gold)
	}
	if !slices.Contains(next.OwnedCosmetics, "window") || !slices.Contains(next.OwnedCosmetics, "rug") || len(next.OwnedCosmetics) != 2 {
		t.Errorf("OwnedCosmetics = %v, want rug and window", next.OwnedCosmetics)
	}
	if state.Gold != 150 || len(state.OwnedCosmetics) != 1 {
		t.Error("Activate mutated its input state")
	}
}

func TestCustomize_PurchaseDoesNotShareTheInputArray(t *testing.T) {
	c, _ := newTestCustomize()
	owned := make([]string, 1, 8)
	owned[0] = "rug"
	state := storage.PlayerState{Gold: 300, OwnedCosmetics: owned}

	first, _, _ := c.Activate(0, state)
	second, _, _ := c.Activate(1, state)
	if !slices.Contains(first.OwnedCosmetics, "window") || slices.Contains(first.OwnedCosmetics, "bookshelf") {
		t.Errorf("first purchase result was overwritten by the second: %v", first.OwnedCosmetics)
	}
	if !slices.Contains(second.OwnedCosmetics, "bookshelf") {
		t.Errorf("second purchase = %v, want bookshelf", second.OwnedCosmetics)
	}
}

func TestCustomize_ActivateWithInsufficientGoldLeavesStateUnchanged(t *testing.T) {
	c, _ := newTestCustomize()
	state := storage.PlayerState{Gold: 50}

	next, changed, _ := c.Activate(3, state)
	if changed || next.Gold != 50 || len(next.OwnedCosmetics) != 0 {
		t.Errorf("insufficient gold changed state: changed=%v next=%+v", changed, next)
	}
	if !strings.Contains(plain(c.View(3, next)), "Insufficient gold (need 50 more)") {
		t.Error("view does not show Insufficient gold (need 50 more)")
	}
}

func TestCustomize_ActivateOnAlreadyOwnedItemIsANoOp(t *testing.T) {
	c, _ := newTestCustomize()
	state := storage.PlayerState{Gold: 150, OwnedCosmetics: []string{"window"}}

	next, changed, _ := c.Activate(0, state)
	if changed || next.Gold != 150 || len(next.OwnedCosmetics) != 1 {
		t.Errorf("buying an owned item changed state: changed=%v next=%+v", changed, next)
	}
	if !strings.Contains(plain(c.View(0, next)), "Already owned") {
		t.Error("view does not show Already owned")
	}
}

func TestCustomize_AlreadyOwnedWinsOverInsufficientGold(t *testing.T) {
	c, _ := newTestCustomize()
	c.Activate(0, storage.PlayerState{Gold: 0, OwnedCosmetics: []string{"window"}})
	if view := plain(c.View(0, storage.PlayerState{})); !strings.Contains(view, "Already owned") || strings.Contains(view, "Insufficient") {
		t.Error("an owned item with no gold must say Already owned")
	}
}

func TestCustomize_ActivateSetsAddedToRoomMessage(t *testing.T) {
	c, _ := newTestCustomize()
	next, _, _ := c.Activate(4, storage.PlayerState{Gold: 100})
	if !strings.Contains(plain(c.View(4, next)), "Rug added to your room") {
		t.Error("view does not show Rug added to your room")
	}
}

func TestCustomize_MessageExpiresAfterTwoSeconds(t *testing.T) {
	c, clock := newTestCustomize()
	next, _, _ := c.Activate(0, storage.PlayerState{Gold: 100})

	clock.at = clock.at.Add(1999 * time.Millisecond)
	if !strings.Contains(plain(c.View(0, next)), "Window added to your room") {
		t.Fatal("message disappeared before 2 seconds")
	}
	clock.at = clock.at.Add(time.Millisecond)
	if strings.Contains(plain(c.View(0, next)), "added to your room") {
		t.Error("message still shown after 2 seconds")
	}
}

func TestCustomize_DoublePurchaseWithinOneUpdateCycleOnlyDeductsOnce(t *testing.T) {
	c, _ := newTestCustomize()

	first, firstChanged, _ := c.Activate(2, storage.PlayerState{Gold: 300})
	second, secondChanged, _ := c.Activate(2, first)
	if !firstChanged || secondChanged {
		t.Fatalf("changed = %v then %v, want true then false", firstChanged, secondChanged)
	}
	if second.Gold != 200 || len(second.OwnedCosmetics) != 1 {
		t.Errorf("after two presses: gold=%d owned=%v, want 200 and one flower", second.Gold, second.OwnedCosmetics)
	}
}

func TestCustomize_ActivateOutOfRangeIsIgnored(t *testing.T) {
	c, _ := newTestCustomize()
	for _, selected := range []int{-1, 5} {
		if _, changed, cmd := c.Activate(selected, storage.PlayerState{Gold: 999}); changed || cmd != nil {
			t.Errorf("Activate(%d) changed=%v cmd=%v, want an ignored press", selected, changed, cmd)
		}
	}
}

func TestCustomize_PurchasePersistsAcrossRelaunch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nookbuddy.db")
	store, _, err := storage.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}

	var model tea.Model = ui.NewModel(store, nil, storage.PlayerState{Gold: 250}, []ui.Screen{NewSell(), NewUpgrades(), NewCustomize()})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model, purchase := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = settle(model, purchase, 300*time.Millisecond)

	if view := plain(model.View()); !strings.Contains(view, "✓ Bookshelf") || !strings.Contains(view, "Gold: 150") {
		t.Fatalf("purchase not reflected in the view:\n%s", view)
	}
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
	if loaded.Gold != 150 || !slices.Equal(loaded.OwnedCosmetics, []string{"bookshelf"}) {
		t.Errorf("after relaunch: gold=%d owned=%v, want 150 and [bookshelf]", loaded.Gold, loaded.OwnedCosmetics)
	}

	view := plain(NewCustomize().View(0, loaded))
	if !strings.Contains(view, "✓ Bookshelf") {
		t.Error("relaunched Customize screen does not show Bookshelf as owned")
	}
}

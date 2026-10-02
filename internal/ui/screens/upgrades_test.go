package screens

import (
	"context"
	"nookbuddy/internal/game"
	"nookbuddy/internal/storage"
	"nookbuddy/internal/ui"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var sgr = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

func plain(s string) string {
	return sgr.ReplaceAllString(s, "")
}

func forceANSI(t *testing.T) {
	t.Helper()
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })
}

func attributes(line string) []string {
	match := sgr.FindStringSubmatch(line)
	if match == nil {
		return nil
	}
	return strings.Split(match[1], ";")
}

type fakeClock struct {
	at time.Time
}

func (c *fakeClock) now() time.Time { return c.at }

func newTestUpgrades() (*Upgrades, *fakeClock) {
	clock := &fakeClock{at: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)}
	u := NewUpgrades()
	u.now = clock.now
	return u, clock
}

func rowLines(view string) []string {
	var rows []string
	for _, line := range strings.Split(view, "\n") {
		text := strings.TrimSpace(plain(line))
		for _, name := range []string{"Clicks", "Keys", "Value"} {
			if strings.HasPrefix(strings.TrimPrefix(text, "> "), name+" ") {
				rows = append(rows, line)
			}
		}
	}
	return rows
}

func TestUpgrades_ItemCountIsThree(t *testing.T) {
	if got := NewUpgrades().ItemCount(); got != 3 {
		t.Errorf("ItemCount() = %d, want 3", got)
	}
	if got := NewUpgrades().Title(); got != "Upgrades" {
		t.Errorf("Title() = %q, want Upgrades", got)
	}
}

func TestUpgrades_ViewShowsLevelEffectsAndInitialCosts(t *testing.T) {
	u, _ := newTestUpgrades()
	rows := rowLines(u.View(0, storage.PlayerState{}))
	if len(rows) != 3 {
		t.Fatalf("rendered %d rows, want 3", len(rows))
	}

	want := [][]string{
		{"Clicks", "Lv 0", "+1/click → +2/click", "50 gold"},
		{"Keys", "Lv 0", "+1/key → +2/key", "50 gold"},
		{"Value", "Lv 0", "10g/project → 15g/project", "50 gold"},
	}
	for i, fields := range want {
		for _, field := range fields {
			if !strings.Contains(plain(rows[i]), field) {
				t.Errorf("row %d %q is missing %q", i, plain(rows[i]), field)
			}
		}
	}
}

func TestUpgrades_ViewShowsGoldBalance(t *testing.T) {
	u, _ := newTestUpgrades()
	if !strings.Contains(plain(u.View(0, storage.PlayerState{Gold: 1234})), "Gold: 1234") {
		t.Error("view does not show the gold balance")
	}
}

func TestUpgrades_ViewCostsFollowTheLevels(t *testing.T) {
	u, _ := newTestUpgrades()
	for level, cost := range []string{"50 gold", "75 gold", "112 gold", "168 gold", "253 gold"} {
		rows := rowLines(u.View(0, storage.PlayerState{UpgradeKeysLevel: level}))
		if !strings.Contains(plain(rows[1]), cost) {
			t.Errorf("Keys at Lv %d shows %q, want %q", level, plain(rows[1]), cost)
		}
	}
}

func TestUpgrades_ViewShowsAffordableRowNormalAndUnaffordableDim(t *testing.T) {
	forceANSI(t)
	u, _ := newTestUpgrades()
	state := storage.PlayerState{Gold: 60, UpgradeKeysLevel: 1}

	rows := rowLines(u.View(2, state))
	if strings.Contains(rows[0], "\x1b[") {
		t.Errorf("affordable Clicks row is styled: %q", rows[0])
	}
	if rows[1] != ui.Dim.Render(plain(rows[1])) {
		t.Errorf("unaffordable Keys row (cost 75, gold 60) is not dim: %q", rows[1])
	}
	if slices.Contains(attributes(rows[1]), "1") {
		t.Error("unaffordable, unselected row is bold")
	}
}

func TestUpgrades_AffordabilityUpdatesLiveWithGold(t *testing.T) {
	forceANSI(t)
	u, _ := newTestUpgrades()

	poor := rowLines(u.View(2, storage.PlayerState{Gold: 49}))
	rich := rowLines(u.View(2, storage.PlayerState{Gold: 50}))
	if !strings.Contains(poor[0], "\x1b[") {
		t.Error("Clicks row with 49 gold (cost 50) is not dim")
	}
	if strings.Contains(rich[0], "\x1b[") {
		t.Error("Clicks row with 50 gold (cost 50) is still dim")
	}
}

func TestUpgrades_ViewMarksSelectedRow(t *testing.T) {
	forceANSI(t)
	u, _ := newTestUpgrades()
	rows := rowLines(u.View(1, storage.PlayerState{Gold: 500}))

	for i, row := range rows {
		hasMarker := strings.HasPrefix(plain(row), "> ")
		if hasMarker != (i == 1) {
			t.Errorf("row %d selection marker = %v, want %v", i, hasMarker, i == 1)
		}
	}
	if attrs := attributes(rows[1]); !slices.Contains(attrs, "1") {
		t.Errorf("selected row attributes = %v, want bold", attrs)
	}
	if !strings.HasPrefix(plain(rows[0]), "  ") {
		t.Errorf("unselected row %q is not indented", plain(rows[0]))
	}
}

func TestUpgrades_SelectedUnaffordableRowStaysDim(t *testing.T) {
	forceANSI(t)
	u, _ := newTestUpgrades()
	rows := rowLines(u.View(0, storage.PlayerState{Gold: 10}))
	if rows[0] != ui.Dim.Bold(true).Render(plain(rows[0])) {
		t.Errorf("selected unaffordable row = %q, want dim and bold", rows[0])
	}
}

func TestUpgrades_ActivateDeductsCostAndIncrementsLevel(t *testing.T) {
	u, _ := newTestUpgrades()
	state := storage.PlayerState{Gold: 60, OwnedCosmetics: []string{"rug"}}

	next, changed, cmd := u.Activate(0, state)
	if !changed {
		t.Fatal("changed = false after an affordable purchase")
	}
	if next.Gold != 10 || next.UpgradeClicksLevel != 1 {
		t.Errorf("after buying Clicks: gold=%d level=%d, want 10/1", next.Gold, next.UpgradeClicksLevel)
	}
	if next.UpgradeKeysLevel != 0 || next.UpgradeValueLevel != 0 || len(next.OwnedCosmetics) != 1 {
		t.Errorf("purchase touched unrelated fields: %+v", next)
	}
	if got := game.ClickContribution(next.UpgradeClicksLevel); got != 2 {
		t.Errorf("next click contribution = %d, want +2", got)
	}
	if cmd == nil {
		t.Error("no command returned to refresh the confirmation message")
	}
	if state.Gold != 60 || state.UpgradeClicksLevel != 0 {
		t.Error("Activate mutated its input state")
	}
}

func TestUpgrades_ActivateKeysAndValueRaiseOnlyTheirLevel(t *testing.T) {
	u, _ := newTestUpgrades()

	keys, _, _ := u.Activate(1, storage.PlayerState{Gold: 50})
	if keys.UpgradeKeysLevel != 1 || keys.UpgradeClicksLevel != 0 || keys.UpgradeValueLevel != 0 || keys.Gold != 0 {
		t.Errorf("after buying Keys: %+v", keys)
	}

	value, _, _ := u.Activate(2, storage.PlayerState{Gold: 50})
	if value.UpgradeValueLevel != 1 || value.UpgradeClicksLevel != 0 || value.UpgradeKeysLevel != 0 {
		t.Errorf("after buying Value: %+v", value)
	}
	if got := game.SaleValue(value.UpgradeValueLevel); got != 15 {
		t.Errorf("next sale value = %d gold per project, want 15", got)
	}
}

func TestUpgrades_ActivateWithInsufficientGoldLeavesStateUnchanged(t *testing.T) {
	u, _ := newTestUpgrades()
	state := storage.PlayerState{Gold: 10}

	next, changed, cmd := u.Activate(0, state)
	if changed || next.Gold != 10 || next.UpgradeClicksLevel != 0 {
		t.Errorf("insufficient gold changed state: changed=%v next=%+v", changed, next)
	}
	if cmd == nil {
		t.Error("no command returned to expire the message")
	}
	if !strings.Contains(plain(u.View(0, next)), "Insufficient gold (need 40 more)") {
		t.Error("view does not show the insufficient gold message with N = 40")
	}
}

func TestUpgrades_InsufficientGoldUsesTheSelectedRowCost(t *testing.T) {
	u, _ := newTestUpgrades()
	u.Activate(1, storage.PlayerState{Gold: 100, UpgradeKeysLevel: 3})
	if !strings.Contains(plain(u.View(1, storage.PlayerState{})), "Insufficient gold (need 68 more)") {
		t.Error("N must be cost(3) = 168 minus gold 100 = 68")
	}
}

func TestUpgrades_ActivateSetsUpgradedMessage(t *testing.T) {
	u, _ := newTestUpgrades()
	next, _, _ := u.Activate(0, storage.PlayerState{Gold: 60})
	if !strings.Contains(plain(u.View(0, next)), "Upgraded to Lv 1") {
		t.Error("view does not show Upgraded to Lv 1")
	}
}

func TestUpgrades_MessageExpiresAfterTwoSeconds(t *testing.T) {
	u, clock := newTestUpgrades()
	next, _, _ := u.Activate(0, storage.PlayerState{Gold: 60})

	clock.at = clock.at.Add(1999 * time.Millisecond)
	if !strings.Contains(plain(u.View(0, next)), "Upgraded to Lv 1") {
		t.Fatal("message disappeared before 2 seconds")
	}

	clock.at = clock.at.Add(time.Millisecond)
	if strings.Contains(plain(u.View(0, next)), "Upgraded to Lv 1") {
		t.Error("message still shown after 2 seconds")
	}
}

func TestUpgrades_CappedCostShowsMaxAndCannotBeBought(t *testing.T) {
	u, _ := newTestUpgrades()
	state := storage.PlayerState{Gold: 1_000_000, UpgradeClicksLevel: 500}

	if !strings.Contains(plain(rowLines(u.View(1, state))[0]), "MAX") {
		t.Error("capped cost row does not show MAX")
	}
	next, changed, _ := u.Activate(0, state)
	if changed || next.UpgradeClicksLevel != 500 {
		t.Error("an upgrade at the capped cost was bought")
	}
	if !strings.Contains(plain(u.View(0, next)), "Max level reached") {
		t.Error("view does not explain why the capped upgrade cannot be bought")
	}
}

func TestUpgrades_ActivateOutOfRangeIsIgnored(t *testing.T) {
	u, _ := newTestUpgrades()
	for _, selected := range []int{-1, 3} {
		if _, changed, cmd := u.Activate(selected, storage.PlayerState{Gold: 999}); changed || cmd != nil {
			t.Errorf("Activate(%d) changed=%v cmd=%v, want an ignored press", selected, changed, cmd)
		}
	}
}

func runLeaves(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, inner := range batch {
			go runLeaves(inner)
		}
	}
}

func TestUpgrades_PurchaseThroughModelIsSavedImmediately(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nookbuddy.db")
	store, _, err := storage.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	var model tea.Model = ui.NewModel(store, nil, storage.PlayerState{Gold: 60}, []ui.Screen{Sell{}, NewUpgrades(), Customize{}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	runLeaves(cmd)

	deadline := time.Now().Add(2 * time.Second)
	for {
		got, err := store.Load(context.Background())
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.UpgradeClicksLevel == 1 && got.Gold == 10 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("persisted state = %+v, want Clicks Lv 1 and 10 gold saved right after the purchase", got)
		}
		time.Sleep(5 * time.Millisecond)
	}

	if !strings.Contains(plain(model.View()), "Gold: 10") {
		t.Error("the model view does not reflect the purchase")
	}
}

package screens

import (
	"context"
	"nookbuddy/internal/input"
	"nookbuddy/internal/storage"
	"nookbuddy/internal/ui"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type recordingSaver struct {
	mu    sync.Mutex
	saved []storage.PlayerState
}

func (r *recordingSaver) Save(_ context.Context, state storage.PlayerState) (storage.SaveResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved = append(r.saved, state)
	return storage.SaveResult{OK: true}, nil
}

func (r *recordingSaver) last() storage.PlayerState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saved[len(r.saved)-1]
}

func drive(model tea.Model, msgs ...tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	for _, msg := range msgs {
		model, cmd = model.Update(msg)
	}
	return model, cmd
}

func settle(model tea.Model, cmd tea.Cmd, within time.Duration) tea.Model {
	if cmd == nil {
		return model
	}
	msgs := make(chan tea.Msg, 16)
	var leaves []tea.Cmd
	if batch, ok := cmd().(tea.BatchMsg); ok {
		leaves = batch
	} else {
		leaves = []tea.Cmd{cmd}
	}
	for _, leaf := range leaves {
		if leaf != nil {
			go func() { msgs <- leaf() }()
		}
	}
	deadline := time.After(within)
	for {
		select {
		case msg := <-msgs:
			if msg != nil {
				model, _ = model.Update(msg)
			}
		case <-deadline:
			return model
		}
	}
}

func TestUpgradePurchaseAppliesToTheNextClick(t *testing.T) {
	saver := &recordingSaver{}
	click := input.InputEventMsg{Events: []input.Event{{Type: input.Click}}}
	model := tea.Model(ui.NewModel(saver, nil, storage.PlayerState{Gold: 60}, []ui.Screen{NewSell(), NewUpgrades(), NewCustomize()}))

	model, purchase := drive(model, click, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")}, tea.KeyMsg{Type: tea.KeyEnter})
	if view := plain(model.View()); !strings.Contains(view, "Gold: 10") {
		t.Fatalf("the Clicks upgrade was not bought:\n%s", view)
	}
	model = settle(model, purchase, 300*time.Millisecond)

	model, _ = drive(model, click)
	_, quit := drive(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if quit == nil {
		t.Fatal("q returned no save command")
	}
	quit()

	if got := saver.last(); got.ClicksProgress != 3 || got.UpgradeClicksLevel != 1 {
		t.Errorf("saved ClicksProgress = %d at Lv %d, want 1 before the purchase + 2 after it at Lv 1", got.ClicksProgress, got.UpgradeClicksLevel)
	}
}

func TestCosmeticPurchaseRendersInTheRoomOnTheSameFrame(t *testing.T) {
	model := tea.Model(ui.NewModel(&recordingSaver{}, nil, storage.PlayerState{Gold: 100}, []ui.Screen{NewSell(), NewUpgrades(), NewCustomize()}))
	model, _ = drive(model, tea.WindowSizeMsg{Width: 120, Height: 30}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	rug := "░▒▓▓▓▓▓▓▓▓▓▓▓▓▒░"
	if strings.Contains(plain(model.View()), rug) {
		t.Fatal("the rug is drawn before it was bought")
	}

	model, _ = drive(model,
		tea.KeyMsg{Type: tea.KeyDown}, tea.KeyMsg{Type: tea.KeyDown}, tea.KeyMsg{Type: tea.KeyDown}, tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeyEnter})

	frame := plain(model.View())
	if !strings.Contains(frame, "Rug added to your room") || !strings.Contains(frame, rug) {
		t.Errorf("confirmation and rug must appear in the same frame:\n%s", frame)
	}
}

package screens

import (
	"nookbuddy/internal/storage"
	"nookbuddy/internal/ui"
	"strings"
	"testing"
)

func assertPlaceholder(t *testing.T, screen ui.Screen, title string) {
	t.Helper()
	if got := screen.Title(); got != title {
		t.Errorf("Title() = %q, want %q", got, title)
	}
	if got := screen.ItemCount(); got != 0 {
		t.Errorf("ItemCount() = %d, want 0", got)
	}
	state := storage.PlayerState{Gold: 9}
	if got := screen.View(0, state); !strings.Contains(got, title+" — not yet implemented") {
		t.Errorf("View(0) = %q, want the not-yet-implemented line", got)
	}
	next, changed, cmd := screen.Activate(0, state)
	if changed || cmd != nil || next.Gold != state.Gold {
		t.Errorf("Activate(0) = (%+v, %v, %v), want the state unchanged and no command", next, changed, cmd)
	}
}

func TestSellPlaceholder_ReportsZeroItemsAndPlaceholderText(t *testing.T) {
	assertPlaceholder(t, Sell{}, "Sell")
}

func TestCustomizePlaceholder_ReportsZeroItemsAndPlaceholderText(t *testing.T) {
	assertPlaceholder(t, Customize{}, "Customize")
}

package screens

import (
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
	if got := screen.View(0); !strings.Contains(got, title+" — not yet implemented") {
		t.Errorf("View(0) = %q, want the not-yet-implemented line", got)
	}
	if screen.Activate(0) != nil {
		t.Error("Activate(0) returned a command, want nil")
	}
}

func TestSellPlaceholder_ReportsZeroItemsAndPlaceholderText(t *testing.T) {
	assertPlaceholder(t, Sell{}, "Sell")
}

func TestUpgradesPlaceholder_ReportsZeroItemsAndPlaceholderText(t *testing.T) {
	assertPlaceholder(t, Upgrades{}, "Upgrades")
}

func TestCustomizePlaceholder_ReportsZeroItemsAndPlaceholderText(t *testing.T) {
	assertPlaceholder(t, Customize{}, "Customize")
}

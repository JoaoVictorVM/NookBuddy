package ui

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var sgrParams = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

func forceANSI(t *testing.T) {
	t.Helper()
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })
}

func attributesOf(rendered string) []string {
	match := sgrParams.FindStringSubmatch(rendered)
	if match == nil {
		return nil
	}
	return strings.Split(match[1], ";")
}

func TestHeader_ListsEveryScreenInOrder(t *testing.T) {
	out := plain(renderTabs(threeScreens(), 0))
	if out != "Sell │ Upgrades │ Customize" {
		t.Errorf("header = %q, want %q", out, "Sell │ Upgrades │ Customize")
	}
}

func TestHeader_ActiveScreenIsBoldAndUnderlined(t *testing.T) {
	forceANSI(t)

	for active, title := range []string{"Sell", "Upgrades", "Customize"} {
		out := renderTabs(threeScreens(), active)
		segment := activeTab.Render(title)
		if !strings.Contains(out, segment) {
			t.Fatalf("header for active %q does not contain its highlighted title", title)
		}
		attrs := attributesOf(segment)
		if !slices.Contains(attrs, "1") || !slices.Contains(attrs, "4") {
			t.Errorf("active %q attributes = %v, want bold (1) and underline (4)", title, attrs)
		}
	}
}

func TestHeader_InactiveScreensAreNotHighlighted(t *testing.T) {
	forceANSI(t)

	attrs := attributesOf(inactiveTab.Render("Sell"))
	if slices.Contains(attrs, "1") || slices.Contains(attrs, "4") {
		t.Errorf("inactive attributes = %v, want neither bold nor underline", attrs)
	}
}

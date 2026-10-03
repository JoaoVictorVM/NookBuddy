package ui

import (
	"nookbuddy/internal/storage"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderRoom_ComposesSceneCountersAndNotice(t *testing.T) {
	snap := roomSnapshot{
		state:       storage.PlayerState{ClicksProgress: 10, KeysProgress: 30, ProjectsReady: 2, Gold: 77, OwnedCosmetics: []string{"window"}},
		now:         frameBase,
		lastInputAt: frameBase,
		notice:      recoveryNotice,
	}
	out := plain(renderRoom(27, snap))

	for _, want := range []string{cosmeticArt["window"], "Clicks", "10/100", "30/300", "Projects ready: 2", "Gold: 77", "Previous save was unreadable"} {
		if !strings.Contains(out, want) {
			t.Errorf("room panel missing %q", want)
		}
	}
	if strings.Index(out, cosmeticArt["window"]) >= strings.Index(out, "Clicks") || strings.Index(out, "Gold: 77") >= strings.Index(out, "Previous save") {
		t.Error("room panel order must be scene, counters, then notice")
	}
}

func TestRenderRoom_FitsTheFixedPanel(t *testing.T) {
	snap := roomSnapshot{state: storage.PlayerState{Gold: 999999999}, now: frameBase, notice: recoveryNotice}
	out := renderRoom(27, snap)
	if w := lipgloss.Width(out); w != LeftPanelWidth {
		t.Errorf("room panel is %d columns, want %d", w, LeftPanelWidth)
	}
	if h := lipgloss.Height(out); h != 27 {
		t.Errorf("room panel is %d rows at height 27, want 27", h)
	}
}

func TestRenderRoom_ContentFitsTheMinimumTerminal(t *testing.T) {
	content := lipgloss.JoinVertical(lipgloss.Left,
		renderScene(frameBase, frameBase.Add(-time.Hour), nil), "",
		renderCounters(storage.PlayerState{}, frameBase, time.Time{}), "",
		renderNotice(memoryOnlyNotice))
	if h := lipgloss.Height(content); h > minHeight-3 {
		t.Errorf("panel content is %d rows, want at most %d to fit a %d-row terminal", h, minHeight-3, minHeight)
	}
}

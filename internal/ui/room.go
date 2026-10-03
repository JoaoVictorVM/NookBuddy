package ui

import (
	"nookbuddy/internal/storage"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type roomSnapshot struct {
	state          storage.PlayerState
	now            time.Time
	lastInputAt    time.Time
	highlightUntil time.Time
	notice         string
}

func renderRoom(height int, snap roomSnapshot) string {
	content := lipgloss.JoinVertical(lipgloss.Left,
		renderScene(snap.now, snap.lastInputAt, snap.state.OwnedCosmetics),
		"",
		renderCounters(snap.state, snap.now, snap.highlightUntil),
		"",
		renderNotice(snap.notice),
	)
	style := Panel.Width(LeftPanelWidth - 2)
	if height > 2 {
		style = style.Height(height - 2)
	}
	return style.Render(content)
}

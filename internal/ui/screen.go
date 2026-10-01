package ui

import (
	"nookbuddy/internal/storage"

	tea "github.com/charmbracelet/bubbletea"
)

type Screen interface {
	Title() string
	ItemCount() int
	View(selected int, state storage.PlayerState) string
	Activate(selected int, state storage.PlayerState) (next storage.PlayerState, changed bool, cmd tea.Cmd)
}

func clamp(selected, itemCount int) int {
	if itemCount <= 0 || selected < 0 {
		return 0
	}
	if selected >= itemCount {
		return itemCount - 1
	}
	return selected
}

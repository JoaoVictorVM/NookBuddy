package ui

import tea "github.com/charmbracelet/bubbletea"

type Screen interface {
	Title() string
	ItemCount() int
	View(selected int) string
	Activate(selected int) tea.Cmd
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

package screens

import (
	"nookbuddy/internal/storage"

	tea "github.com/charmbracelet/bubbletea"
)

type Customize struct{}

func (Customize) Title() string { return "Customize" }

func (Customize) ItemCount() int { return 0 }

func (Customize) View(int, storage.PlayerState) string { return placeholder("Customize") }

func (Customize) Activate(_ int, state storage.PlayerState) (storage.PlayerState, bool, tea.Cmd) {
	return state, false, nil
}

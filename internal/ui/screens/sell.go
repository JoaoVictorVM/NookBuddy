package screens

import (
	"nookbuddy/internal/storage"

	tea "github.com/charmbracelet/bubbletea"
)

type Sell struct{}

func (Sell) Title() string { return "Sell" }

func (Sell) ItemCount() int { return 0 }

func (Sell) View(int, storage.PlayerState) string { return placeholder("Sell") }

func (Sell) Activate(_ int, state storage.PlayerState) (storage.PlayerState, bool, tea.Cmd) {
	return state, false, nil
}

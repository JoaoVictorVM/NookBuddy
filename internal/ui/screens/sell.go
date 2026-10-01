package screens

import tea "github.com/charmbracelet/bubbletea"

type Sell struct{}

func (Sell) Title() string { return "Sell" }

func (Sell) ItemCount() int { return 0 }

func (Sell) View(int) string { return placeholder("Sell") }

func (Sell) Activate(int) tea.Cmd { return nil }

package screens

import tea "github.com/charmbracelet/bubbletea"

type Upgrades struct{}

func (Upgrades) Title() string { return "Upgrades" }

func (Upgrades) ItemCount() int { return 0 }

func (Upgrades) View(int) string { return placeholder("Upgrades") }

func (Upgrades) Activate(int) tea.Cmd { return nil }

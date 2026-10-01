package screens

import tea "github.com/charmbracelet/bubbletea"

type Customize struct{}

func (Customize) Title() string { return "Customize" }

func (Customize) ItemCount() int { return 0 }

func (Customize) View(int) string { return placeholder("Customize") }

func (Customize) Activate(int) tea.Cmd { return nil }

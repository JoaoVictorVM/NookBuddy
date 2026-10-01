package ui

import "github.com/charmbracelet/lipgloss"

const LeftPanelWidth = 40

var (
	AccentColor = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}
	DimColor    = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	BorderColor = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#4B5563"}

	Dim    = lipgloss.NewStyle().Foreground(DimColor)
	Accent = lipgloss.NewStyle().Foreground(AccentColor)
	Panel  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(BorderColor).Padding(0, 1)

	activeTab   = Accent.Bold(true).Underline(true)
	inactiveTab = Dim
)

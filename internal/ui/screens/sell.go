package screens

import (
	"fmt"
	"log/slog"
	"nookbuddy/internal/game"
	"nookbuddy/internal/storage"
	"nookbuddy/internal/ui"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sellMessageTickMsg struct{}

type Sell struct {
	message      string
	messageStyle lipgloss.Style
	expiresAt    time.Time
	now          func() time.Time
}

func NewSell() *Sell {
	return &Sell{now: time.Now}
}

func (*Sell) Title() string { return "Sell" }

func (*Sell) ItemCount() int { return 1 }

func (s *Sell) View(selected int, state storage.PlayerState) string {
	value := game.SaleValue(state.UpgradeValueLevel)
	lines := []string{
		fmt.Sprintf("Gold: %d", state.Gold),
		"",
		fmt.Sprintf("Projects ready: %d", state.ProjectsReady),
		fmt.Sprintf("Value per project: %d gold", value),
		fmt.Sprintf("Total: %d × %d = %d gold", state.ProjectsReady, value, game.SaleTotal(state.ProjectsReady, state.UpgradeValueLevel)),
		"",
		renderSellAll(selected == 0, state.ProjectsReady > 0),
	}
	if s.message != "" && s.now().Before(s.expiresAt) {
		lines = append(lines, "", s.messageStyle.Render(s.message))
	}
	return strings.Join(lines, "\n")
}

func (s *Sell) Activate(selected int, state storage.PlayerState) (storage.PlayerState, bool, tea.Cmd) {
	if selected != 0 {
		return state, false, nil
	}
	tick := tea.Tick(messageDuration, func(time.Time) tea.Msg { return sellMessageTickMsg{} })

	if state.ProjectsReady <= 0 {
		s.show("No projects ready", ui.Warning)
		return state, false, tick
	}

	sale := game.Sell(state)
	if sale.Capped {
		slog.Warn("nookbuddy: sale capped at the maximum gold", "sold", sale.Sold, "earned", sale.Earned)
	}
	noun := "projects"
	if sale.Sold == 1 {
		noun = "project"
	}
	s.show(fmt.Sprintf("Sold %d %s for %d gold", sale.Sold, noun, sale.Earned), ui.Accent)
	return sale.Next, true, tick
}

func (s *Sell) show(message string, style lipgloss.Style) {
	s.message, s.messageStyle, s.expiresAt = message, style, s.now().Add(messageDuration)
}

func renderSellAll(selected, enabled bool) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}
	text := prefix + "Sell all"
	switch {
	case selected && enabled:
		return ui.SelectedRow.Render(text)
	case selected:
		return ui.Dim.Bold(true).Render(text)
	case enabled:
		return text
	default:
		return ui.Dim.Render(text)
	}
}

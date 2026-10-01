package screens

import (
	"fmt"
	"math"
	"nookbuddy/internal/game"
	"nookbuddy/internal/storage"
	"nookbuddy/internal/ui"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const messageDuration = 2 * time.Second

type upgradeMessageTickMsg struct{}

type upgrade struct {
	name   string
	level  func(storage.PlayerState) int
	raise  func(*storage.PlayerState)
	effect func(level int) string
}

var upgrades = []upgrade{
	{
		name:   "Clicks",
		level:  func(s storage.PlayerState) int { return s.UpgradeClicksLevel },
		raise:  func(s *storage.PlayerState) { s.UpgradeClicksLevel++ },
		effect: func(level int) string { return fmt.Sprintf("+%d/click", game.ClickContribution(level)) },
	},
	{
		name:   "Keys",
		level:  func(s storage.PlayerState) int { return s.UpgradeKeysLevel },
		raise:  func(s *storage.PlayerState) { s.UpgradeKeysLevel++ },
		effect: func(level int) string { return fmt.Sprintf("+%d/key", game.KeyContribution(level)) },
	},
	{
		name:   "Value",
		level:  func(s storage.PlayerState) int { return s.UpgradeValueLevel },
		raise:  func(s *storage.PlayerState) { s.UpgradeValueLevel++ },
		effect: func(level int) string { return fmt.Sprintf("%dg/project", game.SaleValue(level)) },
	},
}

type Upgrades struct {
	message      string
	messageStyle lipgloss.Style
	expiresAt    time.Time
	now          func() time.Time
}

func NewUpgrades() *Upgrades {
	return &Upgrades{now: time.Now}
}

func (*Upgrades) Title() string { return "Upgrades" }

func (*Upgrades) ItemCount() int { return len(upgrades) }

func (u *Upgrades) View(selected int, state storage.PlayerState) string {
	lines := []string{fmt.Sprintf("Gold: %d", state.Gold), ""}
	for i, up := range upgrades {
		lines = append(lines, renderUpgrade(up, state, i == selected))
	}
	if u.message != "" && u.now().Before(u.expiresAt) {
		lines = append(lines, "", u.messageStyle.Render(u.message))
	}
	return strings.Join(lines, "\n")
}

func (u *Upgrades) Activate(selected int, state storage.PlayerState) (storage.PlayerState, bool, tea.Cmd) {
	if selected < 0 || selected >= len(upgrades) {
		return state, false, nil
	}
	up := upgrades[selected]
	level := up.level(state)
	cost := game.UpgradeCost(level)

	next, changed := state, false
	switch {
	case cost == math.MaxInt64:
		u.show("Max level reached", ui.Warning)
	case int64(state.Gold) < cost:
		u.show(fmt.Sprintf("Insufficient gold (need %d more)", cost-int64(state.Gold)), ui.Warning)
	default:
		next.Gold -= int(cost)
		up.raise(&next)
		changed = true
		u.show(fmt.Sprintf("Upgraded to Lv %d", level+1), ui.Accent)
	}

	return next, changed, tea.Tick(messageDuration, func(time.Time) tea.Msg { return upgradeMessageTickMsg{} })
}

func (u *Upgrades) show(message string, style lipgloss.Style) {
	u.message, u.messageStyle, u.expiresAt = message, style, u.now().Add(messageDuration)
}

func renderUpgrade(up upgrade, state storage.PlayerState, selected bool) string {
	level := up.level(state)
	cost := game.UpgradeCost(level)

	costText := fmt.Sprintf("%d gold", cost)
	if cost == math.MaxInt64 {
		costText = "MAX"
	}
	effect := up.effect(level) + " → " + up.effect(level+1)
	text := fmt.Sprintf("%-7s Lv %-4d %-25s %s", up.name, level, effect, costText)

	affordable := cost != math.MaxInt64 && int64(state.Gold) >= cost
	switch {
	case selected && affordable:
		return ui.SelectedRow.Render("> " + text)
	case selected:
		return ui.Dim.Bold(true).Render("> " + text)
	case affordable:
		return "  " + text
	default:
		return ui.Dim.Render("  " + text)
	}
}

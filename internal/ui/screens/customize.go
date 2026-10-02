package screens

import (
	"fmt"
	"nookbuddy/internal/game"
	"nookbuddy/internal/storage"
	"nookbuddy/internal/ui"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type customizeMessageTickMsg struct{}

var cosmeticNames = map[string]string{
	"window":    "Window",
	"bookshelf": "Bookshelf",
	"flower":    "Flower",
	"painting":  "Painting",
	"rug":       "Rug",
}

type Customize struct {
	message      string
	messageStyle lipgloss.Style
	expiresAt    time.Time
	now          func() time.Time
}

func NewCustomize() *Customize {
	return &Customize{now: time.Now}
}

func (*Customize) Title() string { return "Customize" }

func (*Customize) ItemCount() int { return len(storage.CosmeticIDs) }

func (c *Customize) View(selected int, state storage.PlayerState) string {
	lines := []string{fmt.Sprintf("Gold: %d", state.Gold), ""}
	for i, id := range storage.CosmeticIDs {
		lines = append(lines, renderCosmetic(id, state, i == selected))
	}
	if c.message != "" && c.now().Before(c.expiresAt) {
		lines = append(lines, "", c.messageStyle.Render(c.message))
	}
	return strings.Join(lines, "\n")
}

func (c *Customize) Activate(selected int, state storage.PlayerState) (storage.PlayerState, bool, tea.Cmd) {
	if selected < 0 || selected >= len(storage.CosmeticIDs) {
		return state, false, nil
	}
	id := storage.CosmeticIDs[selected]
	price := game.CosmeticPrice()

	next, changed := state, false
	switch {
	case slices.Contains(state.OwnedCosmetics, id):
		c.show("Already owned", ui.Warning)
	case state.Gold < price:
		c.show(fmt.Sprintf("Insufficient gold (need %d more)", price-state.Gold), ui.Warning)
	default:
		next.Gold -= price
		next.OwnedCosmetics = append(slices.Clone(state.OwnedCosmetics), id)
		changed = true
		c.show(cosmeticNames[id]+" added to your room", ui.Accent)
	}

	return next, changed, tea.Tick(messageDuration, func(time.Time) tea.Msg { return customizeMessageTickMsg{} })
}

func (c *Customize) show(message string, style lipgloss.Style) {
	c.message, c.messageStyle, c.expiresAt = message, style, c.now().Add(messageDuration)
}

func renderCosmetic(id string, state storage.PlayerState, selected bool) string {
	owned := slices.Contains(state.OwnedCosmetics, id)
	mark, status := "  ", fmt.Sprintf("%d gold", game.CosmeticPrice())
	if owned {
		mark, status = "✓ ", "Owned"
	}

	prefix := "  "
	if selected {
		prefix = "> "
	}
	text := fmt.Sprintf("%s%s%-10s %s", prefix, mark, cosmeticNames[id], status)

	affordable := !owned && state.Gold >= game.CosmeticPrice()
	switch {
	case selected && affordable:
		return ui.SelectedRow.Render(text)
	case selected:
		return ui.Dim.Bold(true).Render(text)
	case affordable:
		return text
	default:
		return ui.Dim.Render(text)
	}
}

package ui

import (
	"context"
	"log/slog"
	"nookbuddy/internal/storage"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const saveTimeout = 2 * time.Second

type Saver interface {
	Save(ctx context.Context, state storage.PlayerState) (storage.SaveResult, error)
}

type savedMsg struct {
	err error
}

type Model struct {
	screens     []Screen
	active      int
	selected    int
	width       int
	height      int
	quitting    bool
	saver       Saver
	state       storage.PlayerState
	saveTimeout time.Duration
	crash       *crashRecorder
}

func NewModel(saver Saver, state storage.PlayerState, screens []Screen) Model {
	return Model{
		screens:     screens,
		saver:       saver,
		state:       state,
		saveTimeout: saveTimeout,
		crash:       &crashRecorder{},
	}
}

func (m Model) Crash() (value any, stack []byte, ok bool) {
	return m.crash.report()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	defer func() {
		if r := recover(); r != nil {
			m.crash.record(r)
			panic(r)
		}
	}()
	next, cmd := m.update(msg)
	return next, m.crash.guard(cmd)
}

func (m Model) View() string {
	defer func() {
		if r := recover(); r != nil {
			m.crash.record(r)
			panic(r)
		}
	}()
	return m.view()
}

func (m Model) update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case savedMsg:
		if msg.err != nil {
			slog.Error("nookbuddy: final save failed", "error", msg.err)
		}
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 1 {
			return m.handleRunes(msg.Runes)
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleRunes(runes []rune) (Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, len(runes))
	for _, r := range runes {
		var cmd tea.Cmd
		m, cmd = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		cmds = append(cmds, m.crash.guard(cmd))
	}
	return m, tea.Batch(cmds...)
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.quitting || len(m.screens) == 0 {
		return m, nil
	}

	switch key := msg.String(); key {
	case "q", "ctrl+c":
		return m.quit()
	case "1", "2", "3":
		return m.switchTo(int(key[0] - '1')), nil
	case "tab":
		return m.switchTo((m.active + 1) % len(m.screens)), nil
	case "shift+tab":
		return m.switchTo((m.active - 1 + len(m.screens)) % len(m.screens)), nil
	case "up", "k":
		m.selected = clamp(m.selected-1, m.activeScreen().ItemCount())
	case "down", "j":
		m.selected = clamp(m.selected+1, m.activeScreen().ItemCount())
	case "enter":
		if m.activeScreen().ItemCount() > 0 {
			return m, m.activeScreen().Activate(m.selected)
		}
	}
	return m, nil
}

func (m Model) switchTo(index int) Model {
	if index < 0 || index >= len(m.screens) {
		return m
	}
	m.active, m.selected = index, 0
	return m
}

func (m Model) activeScreen() Screen {
	return m.screens[m.active]
}

func (m Model) quit() (Model, tea.Cmd) {
	m.quitting = true
	if m.saver == nil {
		return m, tea.Quit
	}
	return m, saveCmd(m.saver, m.state, m.saveTimeout)
}

func saveCmd(saver Saver, state storage.PlayerState, timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			_, err := saver.Save(ctx, state)
			done <- err
		}()

		select {
		case err := <-done:
			return savedMsg{err: err}
		case <-ctx.Done():
			return savedMsg{err: ctx.Err()}
		}
	}
}

func (m Model) view() string {
	bodyHeight := 0
	if m.height > 1 {
		bodyHeight = m.height - 1
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, renderRoom(bodyHeight), m.renderScreen(bodyHeight))
	return lipgloss.JoinVertical(lipgloss.Left, body, renderFooter(m.quitting))
}

func (m Model) renderScreen(height int) string {
	style := Panel
	if m.width > LeftPanelWidth+2 {
		style = style.Width(m.width - LeftPanelWidth - 2)
	}
	if height > 2 {
		style = style.Height(height - 2)
	}

	content := ""
	if len(m.screens) > 0 {
		content = renderTabs(m.screens, m.active) + "\n\n" + m.activeScreen().View(m.selected)
	}
	return style.Render(content)
}

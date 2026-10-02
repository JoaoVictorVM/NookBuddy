package ui

import (
	"context"
	"log/slog"
	"nookbuddy/internal/input"
	"nookbuddy/internal/storage"
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const saveTimeout = 2 * time.Second

type Saver interface {
	Save(ctx context.Context, state storage.PlayerState) (storage.SaveResult, error)
}

type saveResultMsg struct {
	err error
}

type Model struct {
	screens        []Screen
	active         int
	selected       int
	width          int
	height         int
	quitting       bool
	saver          Saver
	state          storage.PlayerState
	saveTimeout    time.Duration
	saving         bool
	pendingSave    bool
	saveFailed     bool
	quitDeadline   time.Time
	dirty          bool
	source         InputSource
	hookStatus     input.Status
	lastInputAt    time.Time
	highlightUntil time.Time
	now            func() time.Time
	schedule       func(time.Duration, tea.Msg) tea.Cmd
	crash          *crashRecorder
}

func NewModel(saver Saver, source InputSource, state storage.PlayerState, screens []Screen) Model {
	m := Model{
		screens:     screens,
		saver:       saver,
		source:      source,
		state:       state,
		saveTimeout: saveTimeout,
		hookStatus:  input.Unavailable,
		now:         time.Now,
		schedule:    scheduleTick,
		crash:       &crashRecorder{},
	}
	if source != nil {
		m.hookStatus = source.Status()
	}
	return m
}

func (m Model) Crash() (value any, stack []byte, ok bool) {
	return m.crash.report()
}

func (m Model) Init() tea.Cmd {
	return m.startTicks()
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
	case saveResultMsg:
		return m.saveFinished(msg.err)
	case inputTickMsg:
		if m.source != nil {
			return m.pollInput()
		}
	case input.InputEventMsg:
		return m.applyEvents(msg.Events), nil
	case autosaveTickMsg:
		return m.autosave()
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
			return m.activate()
		}
	}
	return m, nil
}

func (m Model) activate() (Model, tea.Cmd) {
	next, changed, cmd := m.activeScreen().Activate(m.selected, m.state)
	if !changed {
		return m, cmd
	}
	m.state = next
	m, save := m.requestSave()
	return m, tea.Batch(cmd, save)
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
	m.quitDeadline = time.Now().Add(m.saveTimeout)
	if m.saving {
		m.pendingSave = true
		return m, nil
	}
	return m.startSave()
}

func (m Model) requestSave() (Model, tea.Cmd) {
	if m.saver == nil {
		return m, nil
	}
	if m.saving {
		m.pendingSave = true
		return m, nil
	}
	return m.startSave()
}

func (m Model) startSave() (Model, tea.Cmd) {
	m.saving = true
	m.dirty = false
	deadline := time.Now().Add(m.saveTimeout)
	if m.quitting && deadline.After(m.quitDeadline) {
		deadline = m.quitDeadline
	}
	snapshot := m.state
	snapshot.OwnedCosmetics = slices.Clone(m.state.OwnedCosmetics)
	return m, saveCmd(m.saver, snapshot, deadline)
}

func (m Model) saveFinished(err error) (Model, tea.Cmd) {
	m.saving = false
	m.saveFailed = err != nil
	if err != nil {
		m.dirty = true
		slog.Error("nookbuddy: save failed", "error", err, "final", m.quitting)
	}
	if m.pendingSave {
		m.pendingSave = false
		return m.startSave()
	}
	if m.quitting {
		return m, tea.Quit
	}
	return m, nil
}

func saveCmd(saver Saver, state storage.PlayerState, deadline time.Time) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			_, err := saver.Save(ctx, state)
			done <- err
		}()

		select {
		case err := <-done:
			return saveResultMsg{err: err}
		case <-ctx.Done():
			return saveResultMsg{err: ctx.Err()}
		}
	}
}

func (m Model) view() string {
	bodyHeight := 0
	if m.height > 1 {
		bodyHeight = m.height - 1
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, renderRoom(bodyHeight), m.renderScreen(bodyHeight))
	return lipgloss.JoinVertical(lipgloss.Left, body, renderFooter(m.quitting, m.saveFailed))
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
		content = renderTabs(m.screens, m.active) + "\n\n" + m.activeScreen().View(m.selected, m.state)
	}
	return style.Render(content)
}

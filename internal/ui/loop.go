package ui

import (
	"log/slog"
	"nookbuddy/internal/config"
	"nookbuddy/internal/game"
	"nookbuddy/internal/input"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	inputPollInterval = 250 * time.Millisecond
	animationInterval = 250 * time.Millisecond
	highlightDuration = time.Second
)

type InputSource interface {
	input.EventSource
	Status() input.Status
}

type (
	inputTickMsg     struct{}
	autosaveTickMsg  struct{}
	animationTickMsg struct{}
)

func scheduleTick(after time.Duration, msg tea.Msg) tea.Cmd {
	return tea.Tick(after, func(time.Time) tea.Msg { return msg })
}

func (m Model) startTicks() tea.Cmd {
	cmds := []tea.Cmd{m.schedule(animationInterval, animationTickMsg{})}
	if m.source != nil {
		cmds = append(cmds, m.schedule(inputPollInterval, inputTickMsg{}))
	}
	if m.saver != nil {
		cmds = append(cmds, m.schedule(config.AutosaveInterval, autosaveTickMsg{}))
	}
	return tea.Batch(cmds...)
}

func (m Model) pollInput() (Model, tea.Cmd) {
	m.hookStatus = m.source.Status()
	if batch, ok := input.Cmd(m.source)().(input.InputEventMsg); ok {
		m = m.applyEvents(batch.Events)
	}
	return m, m.schedule(inputPollInterval, inputTickMsg{})
}

func (m Model) applyEvents(events []input.Event) Model {
	if len(events) == 0 {
		return m
	}

	before := m.state.ProjectsReady
	for _, event := range events {
		m.state = game.ApplyEvent(m.state, event)
	}

	now := m.now()
	m.lastInputAt = now
	m.dirty = true
	if m.state.ProjectsReady > before {
		m.highlightUntil = now.Add(highlightDuration)
		slog.Info("nookbuddy: project completed", "projects_ready", m.state.ProjectsReady)
	}
	return m
}

func (m Model) autosave() (Model, tea.Cmd) {
	next := m.schedule(config.AutosaveInterval, autosaveTickMsg{})
	if m.quitting || !m.dirty {
		return m, next
	}
	m, save := m.requestSave()
	return m, tea.Batch(next, save)
}

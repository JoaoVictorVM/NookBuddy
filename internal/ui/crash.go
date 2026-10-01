package ui

import (
	"runtime/debug"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

type crashRecorder struct {
	mu       sync.Mutex
	recorded bool
	value    any
	stack    []byte
}

func (c *crashRecorder) record(value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.recorded {
		return
	}
	c.recorded, c.value, c.stack = true, value, debug.Stack()
}

func (c *crashRecorder) report() (any, []byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value, c.stack, c.recorded
}

func (c *crashRecorder) guard(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		defer func() {
			if r := recover(); r != nil {
				c.record(r)
				panic(r)
			}
		}()
		return cmd()
	}
}

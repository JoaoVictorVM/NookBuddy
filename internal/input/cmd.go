package input

import tea "github.com/charmbracelet/bubbletea"

const batchLimit = 256

type InputEventMsg struct {
	Events []Event
}

func Cmd(source *Source) tea.Cmd {
	return func() tea.Msg {
		return drain(source.Events())
	}
}

func drain(events <-chan Event) tea.Msg {
	batch := make([]Event, 0, batchLimit)
	for len(batch) < batchLimit {
		select {
		case ev, ok := <-events:
			if !ok {
				return batchMsg(batch)
			}
			batch = append(batch, ev)
		default:
			return batchMsg(batch)
		}
	}
	return batchMsg(batch)
}

func batchMsg(batch []Event) tea.Msg {
	if len(batch) == 0 {
		return nil
	}
	return InputEventMsg{Events: batch}
}

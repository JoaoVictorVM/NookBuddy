package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestNotice_PriorityOrder(t *testing.T) {
	steps := []struct {
		memoryOnly, hook, recovered bool
		want                        string
	}{
		{true, true, true, memoryOnlyNotice},
		{false, true, true, hookNotice},
		{false, false, true, recoveryNotice},
		{false, false, false, ""},
	}
	for _, step := range steps {
		if got := activeNotice(step.memoryOnly, step.hook, step.recovered); got != step.want {
			t.Errorf("activeNotice(%v, %v, %v) = %q, want %q", step.memoryOnly, step.hook, step.recovered, got, step.want)
		}
	}
}

func TestNotice_HookUnavailableTextAndStyle(t *testing.T) {
	forceANSI(t)
	out := renderNotice(activeNotice(false, true, false))
	if got := strings.Join(strings.Fields(plain(out)), " "); got != hookNotice {
		t.Errorf("notice = %q, want %q", got, hookNotice)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Error("the hook notice is not styled as a warning")
	}
}

func TestNotice_LongestMessageWrapsWithinTwoRows(t *testing.T) {
	for _, text := range []string{memoryOnlyNotice, hookNotice, recoveryNotice} {
		out := renderNotice(text)
		if h := lipgloss.Height(out); h > noticeRows {
			t.Errorf("%q takes %d rows, want at most %d", text, h, noticeRows)
		}
		if w := lipgloss.Width(out); w > roomWidth {
			t.Errorf("%q is %d columns wide, want at most %d", text, w, roomWidth)
		}
		if got := strings.Join(strings.Fields(plain(out)), " "); got != text {
			t.Errorf("wrapped text = %q, want the full %q", got, text)
		}
	}
}

func TestNotice_EmptyAreaKeepsItsHeight(t *testing.T) {
	if h := lipgloss.Height(renderNotice("")); h != noticeRows {
		t.Errorf("empty notice area is %d rows, want %d so the layout stays stable", h, noticeRows)
	}
}

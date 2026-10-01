package ui

import (
	"regexp"
	"strings"
	"testing"
)

var ansiSequence = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plain(s string) string {
	return ansiSequence.ReplaceAllString(s, "")
}

func TestFooter_RendersShortcutsWhenNotQuitting(t *testing.T) {
	out := plain(renderFooter(false, false))
	for _, hint := range []string{"[1] Sell", "[2] Upgrades", "[3] Customize", "↑/↓ select", "Enter confirm", "q quit"} {
		if !strings.Contains(out, hint) {
			t.Errorf("footer %q is missing %q", out, hint)
		}
	}
}

func TestFooter_RendersSavingWhenQuitting(t *testing.T) {
	if got := plain(renderFooter(true, false)); got != "Saving…" {
		t.Errorf("footer = %q, want exactly %q", got, "Saving…")
	}
}

func TestFooter_RendersSaveFailureWarning(t *testing.T) {
	if got := plain(renderFooter(false, true)); got != "Last save failed — retrying" {
		t.Errorf("footer = %q, want the save failure warning", got)
	}
}

func TestFooter_SavingWinsOverSaveFailure(t *testing.T) {
	if got := plain(renderFooter(true, true)); got != "Saving…" {
		t.Errorf("footer = %q, want Saving… while quitting", got)
	}
}

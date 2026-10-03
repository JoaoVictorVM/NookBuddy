package ui

import (
	"nookbuddy/internal/storage"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestRenderBar_FillsProportionally(t *testing.T) {
	cases := []struct {
		progress, limit, filled int
		text                    string
	}{
		{0, 100, 0, "0/100"},
		{63, 100, 6, "63/100"},
		{100, 100, 10, "100/100"},
		{150, 300, 5, "150/300"},
		{140, 100, 10, "140/100"},
	}
	for _, tc := range cases {
		out := plain(renderBar("Clicks", tc.progress, tc.limit))
		if got := strings.Count(out, "█"); got != tc.filled {
			t.Errorf("%d/%d: %d filled cells, want %d (%q)", tc.progress, tc.limit, got, tc.filled, out)
		}
		if got := strings.Count(out, "░"); got != barCells-tc.filled {
			t.Errorf("%d/%d: %d empty cells, want %d", tc.progress, tc.limit, got, barCells-tc.filled)
		}
		if !strings.HasSuffix(out, "  "+tc.text) {
			t.Errorf("%d/%d: %q does not end with %q", tc.progress, tc.limit, out, tc.text)
		}
	}
}

func TestRenderBar_MatchesTheExampleLayout(t *testing.T) {
	if got := plain(renderBar("Clicks", 40, 100)); got != "Clicks  ████░░░░░░  40/100" {
		t.Errorf("Clicks bar = %q", got)
	}
	if got := plain(renderBar("Keys", 180, 300)); got != "Keys    ██████░░░░  180/300" {
		t.Errorf("Keys bar = %q", got)
	}
}

func TestCounters_ShowClicksKeysProjectsAndGold(t *testing.T) {
	state := storage.PlayerState{ClicksProgress: 40, KeysProgress: 180, ProjectsReady: 3, Gold: 250}
	out := plain(renderCounters(state, frameBase, time.Time{}))
	for _, want := range []string{"Clicks  ████░░░░░░  40/100", "Keys    ██████░░░░  180/300", "Projects ready: 3", "Gold: 250"} {
		if !strings.Contains(out, want) {
			t.Errorf("counters missing %q:\n%s", want, out)
		}
	}
}

func TestCounters_HighlightOnlyWhileWithinWindow(t *testing.T) {
	forceANSI(t)
	state := storage.PlayerState{ProjectsReady: 4}
	until := frameBase.Add(time.Second)

	projectsLine := func(now time.Time) string {
		return strings.Split(renderCounters(state, now, until), "\n")[2]
	}
	if attrs := attributesOf(projectsLine(frameBase)); !slices.Contains(attrs, "1") {
		t.Errorf("projects line during the highlight has attributes %v, want bold", attrs)
	}
	if line := projectsLine(until); strings.Contains(line, "\x1b[") {
		t.Errorf("projects line after the highlight is still styled: %q", line)
	}
}

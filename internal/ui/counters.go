package ui

import (
	"fmt"
	"nookbuddy/internal/config"
	"nookbuddy/internal/storage"
	"strings"
	"time"
)

const barCells = 10

func renderBar(label string, progress, limit int) string {
	filled := 0
	if limit > 0 {
		filled = min(max(progress, 0)*barCells/limit, barCells)
	}
	bar := BarFilled.Render(strings.Repeat("█", filled)) + BarEmpty.Render(strings.Repeat("░", barCells-filled))
	return fmt.Sprintf("%-8s%s  %d/%d", label, bar, progress, limit)
}

func renderCounters(state storage.PlayerState, now, highlightUntil time.Time) string {
	projects := fmt.Sprintf("Projects ready: %d", state.ProjectsReady)
	if now.Before(highlightUntil) {
		projects = Accent.Bold(true).Render(projects)
	}
	return strings.Join([]string{
		renderBar("Clicks", state.ClicksProgress, config.ClicksBarLimit),
		renderBar("Keys", state.KeysProgress, config.KeysBarLimit),
		projects,
		fmt.Sprintf("Gold: %d", state.Gold),
	}, "\n")
}

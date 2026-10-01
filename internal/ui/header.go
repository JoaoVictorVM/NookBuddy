package ui

import "strings"

func renderTabs(screens []Screen, active int) string {
	tabs := make([]string, len(screens))
	for i, screen := range screens {
		if i == active {
			tabs[i] = activeTab.Render(screen.Title())
		} else {
			tabs[i] = inactiveTab.Render(screen.Title())
		}
	}
	return strings.Join(tabs, Dim.Render(" │ "))
}

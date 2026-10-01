package ui

const (
	shortcutsText = "[1] Sell  [2] Upgrades  [3] Customize  ↑/↓ select  Enter confirm  q quit"
	savingText    = "Saving…"
	saveFailText  = "Last save failed — retrying"
)

func renderFooter(quitting, saveFailed bool) string {
	switch {
	case quitting:
		return Dim.Render(savingText)
	case saveFailed:
		return Warning.Render(saveFailText)
	}
	return Dim.Render(shortcutsText)
}

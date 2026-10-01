package ui

const (
	shortcutsText = "[1] Sell  [2] Upgrades  [3] Customize  ↑/↓ select  Enter confirm  q quit"
	savingText    = "Saving…"
)

func renderFooter(quitting bool) string {
	if quitting {
		return Dim.Render(savingText)
	}
	return Dim.Render(shortcutsText)
}

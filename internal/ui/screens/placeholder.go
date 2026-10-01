package screens

import "nookbuddy/internal/ui"

func placeholder(title string) string {
	return ui.Dim.Render(title + " — not yet implemented")
}

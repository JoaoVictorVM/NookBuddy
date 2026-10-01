package ui

func renderRoom(height int) string {
	style := Panel.Width(LeftPanelWidth - 2)
	if height > 2 {
		style = style.Height(height - 2)
	}
	return style.Render(Dim.Render("Room view coming soon"))
}

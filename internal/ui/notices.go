package ui

const (
	memoryOnlyNotice = "Cannot write save file — progress will not be saved"
	hookNotice       = "Input capture unavailable — progress paused"
	recoveryNotice   = "Previous save was unreadable and was moved aside — starting fresh"

	noticeRows = 2
)

func activeNotice(memoryOnly, hookUnavailable, recovered bool) string {
	switch {
	case memoryOnly:
		return memoryOnlyNotice
	case hookUnavailable:
		return hookNotice
	case recovered:
		return recoveryNotice
	}
	return ""
}

func renderNotice(text string) string {
	return Warning.Width(roomWidth).Height(noticeRows).MaxHeight(noticeRows).Render(text)
}

package tui

import "strings"

func renderScrollbar(total, height, offset int) string {
	if height <= 0 {
		return ""
	}
	if total <= height {
		return strings.Repeat(" ", height)
	}

	trackHeight := height
	thumbHeight := max(1, height*height/total)
	thumbTop := 0
	if total > height {
		thumbTop = offset * max(1, trackHeight-thumbHeight) / max(1, total-height)
	}

	lines := make([]string, height)
	for i := range lines {
		if i >= thumbTop && i < thumbTop+thumbHeight {
			lines[i] = titleStyle.Render("▏")
		} else {
			lines[i] = " "
		}
	}
	return strings.Join(lines, "\n")
}

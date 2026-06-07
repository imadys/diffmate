package tui

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
)

func overlayCommitBox(body, box string) string {
	bodyLines := strings.Split(body, "\n")
	boxLines := strings.Split(box, "\n")
	if len(bodyLines) == 0 || len(boxLines) == 0 {
		return body
	}

	startRow := max(0, (len(bodyLines)-len(boxLines))/2)
	for i, boxLine := range boxLines {
		row := startRow + i
		if row >= len(bodyLines) {
			break
		}
		lineWidth := lipgloss.Width(bodyLines[row])
		boxWidth := lipgloss.Width(boxLine)
		startCol := max(0, (lineWidth-boxWidth)/2)
		bodyLines[row] = replaceVisualSegment(bodyLines[row], boxLine, startCol)
	}

	return strings.Join(bodyLines, "\n")
}
func replaceVisualSegment(base, insert string, start int) string {
	if start <= 0 {
		return insert + truncate(base, max(0, lipgloss.Width(base)-lipgloss.Width(insert)))
	}

	prefix := truncate(base, start)
	suffixWidth := max(0, lipgloss.Width(base)-lipgloss.Width(prefix)-lipgloss.Width(insert))
	suffix := ""
	if suffixWidth > 0 {
		suffix = strings.Repeat(" ", suffixWidth)
	}
	return prefix + insert + suffix
}

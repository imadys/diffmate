package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderConsole(width, height int) string {
	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	contentWidth := max(1, innerWidth-2)

	title := subtleStyle.Render("Console")
	if len(m.consoleLines) > 0 {
		title += mutedStyle.Render(fmt.Sprintf("  %d", len(m.consoleLines)))
	}

	lines := []string{title}
	if len(m.consoleLines) == 0 {
		lines = append(lines, mutedStyle.Render("No errors logged."))
	} else {
		visibleCount := max(1, innerHeight-1)
		start := max(0, len(m.consoleLines)-visibleCount)
		for _, line := range m.consoleLines[start:] {
			lines = append(lines, truncate(line, contentWidth))
		}
	}

	box := lipgloss.NewStyle().
		Width(innerWidth).
		Height(innerHeight).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 1).
		Render(fitLines(lines, innerHeight))
	return fitLines(strings.Split(box, "\n"), height)
}

func (m *model) appendConsoleError(source string, err error) {
	if err == nil {
		return
	}
	message := fmt.Sprintf("%s  %s: %s", time.Now().Format("15:04:05"), source, firstLine(err.Error()))
	m.consoleLines = append(m.consoleLines, message)
	if len(m.consoleLines) > 200 {
		m.consoleLines = m.consoleLines[len(m.consoleLines)-200:]
	}
}

func (m model) rightPaneHeights(bodyHeight int) (int, int) {
	if !m.consoleVisible || bodyHeight < 12 {
		return bodyHeight, 0
	}
	consoleHeight := clamp(7, 5, max(5, bodyHeight/3))
	diffHeight := max(1, bodyHeight-consoleHeight-1)
	return diffHeight, consoleHeight
}

package tui

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
)

func fitLines(lines []string, height int) string {
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func indentBlock(value string, width int) string {
	if width <= 0 {
		return value
	}
	prefix := strings.Repeat(" ", width)
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func fitLineSlice(lines []string, height int) []string {
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

func wrapLine(value string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	if value == "" || lipgloss.Width(value) <= width {
		return []string{value}
	}

	runes := []rune(value)
	lines := []string{}
	start := 0
	for start < len(runes) {
		for start < len(runes) && runes[start] == ' ' {
			start++
		}
		if start >= len(runes) {
			break
		}

		end := start
		for end < len(runes) && lipgloss.Width(string(runes[start:end+1])) <= width {
			end++
		}
		if end >= len(runes) {
			lines = append(lines, string(runes[start:]))
			break
		}
		if end == start {
			end++
		}

		breakAt := -1
		for i := end - 1; i > start; i-- {
			if runes[i] == ' ' {
				breakAt = i
				break
			}
		}
		if breakAt > start {
			lines = append(lines, strings.TrimRight(string(runes[start:breakAt]), " "))
			start = breakAt + 1
			continue
		}

		lines = append(lines, string(runes[start:end]))
		start = end
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func splitHeights(total, count int) []int {
	if count <= 0 {
		return nil
	}
	heights := make([]int, count)
	base := max(1, total/count)
	remainder := total - base*count
	for i := range heights {
		heights[i] = base
		if remainder > 0 {
			heights[i]++
			remainder--
		}
	}
	return heights
}
func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > width-1 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}
func normalizeBranchName(value string) string {
	parts := strings.Fields(strings.TrimSpace(value))
	return strings.Join(parts, "-")
}
func clamp(value, low, high int) int {
	return min(max(value, low), high)
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

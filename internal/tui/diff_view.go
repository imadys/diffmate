package tui

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

var diffHunkPattern = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func (m model) renderDiff(width, height int) string {
	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	contentWidth := max(1, innerWidth-2)
	lines := m.diffLines()
	if strings.TrimSpace(m.diff) == "" {
		lines = []string{mutedStyle.Render("Select a changed file to view its diff.")}
	}

	header := subtleStyle.Render("Diff")
	title := m.previewTitle()
	if title != "" {
		scroll := fmt.Sprintf("  %d/%d", min(m.diffViewport.YOffset+1, len(lines)), max(1, len(lines)))
		header = subtleStyle.Render(truncate(title, max(1, contentWidth-len(scroll)))) + mutedStyle.Render(scroll)
	}

	viewportWidth := max(1, contentWidth-1)
	content := formatDiffLines(lines, m.selectedFilePath(), viewportWidth)

	border := lipgloss.Color("238")
	if m.focus == diffFocus {
		border = lipgloss.Color("86")
	}

	vp := m.diffViewport
	if vp.Width == 0 || vp.Height == 0 {
		vp = viewport.New(viewportWidth, max(1, innerHeight-1))
	}
	vp.Width = viewportWidth
	vp.Height = max(1, innerHeight-1)
	vp.SetContent(strings.Join(content, "\n"))

	scrollbar := renderScrollbar(len(content), vp.Height, vp.YOffset)
	out := header + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, vp.View(), scrollbar)
	box := lipgloss.NewStyle().
		Width(innerWidth).
		Height(innerHeight).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Render(fitLines(strings.Split(out, "\n"), innerHeight))
	return fitLines(strings.Split(box, "\n"), height)
}
func (m model) diffHeight() int {
	return max(1, m.diffViewport.Height)
}

func (m *model) syncDiffViewport() {
	if m.width == 0 || m.height == 0 {
		return
	}

	sideInset := 1
	topGap := 0
	bodyGap := 1
	bodyWidth := max(1, m.width-sideInset*2)
	headerHeight := 1
	footerHeight := 1
	bodyHeight := max(1, m.height-headerHeight-footerHeight-topGap)
	sidebarWidth := clamp(34, 26, bodyWidth/2)
	diffWidth := max(1, bodyWidth-sidebarWidth-bodyGap)
	diffHeight, _ := m.rightPaneHeights(bodyHeight)

	innerWidth := max(1, diffWidth-4)
	innerHeight := max(1, diffHeight-2)
	m.diffViewport.Width = max(1, innerWidth-3)
	m.diffViewport.Height = max(1, innerHeight-1)
}

func (m *model) clampDiffOffset() {
	m.syncDiffViewport()
	m.diffViewport.SetYOffset(m.diffOffset)
	m.diffOffset = m.diffViewport.YOffset
}
func (m *model) scrollDiff(delta int) {
	m.syncDiffViewport()
	m.updateDiffViewportContent()
	if delta > 0 {
		m.diffViewport.ScrollDown(delta)
	} else if delta < 0 {
		m.diffViewport.ScrollUp(-delta)
	}
	m.diffOffset = m.diffViewport.YOffset
}
func (m model) diffLines() []string {
	if m.diff == "" {
		return []string{""}
	}
	// Expand tabs to spaces so lipgloss width measurements match what the
	// terminal actually renders (a raw tab counts as 1 column here but expands
	// to several on screen, which would wrap lines and break the layout).
	return strings.Split(strings.ReplaceAll(m.diff, "\t", "    "), "\n")
}
func (m *model) updateDiffViewportContent() {
	m.syncDiffViewport()
	m.diffViewport.SetContent(strings.Join(formatDiffLines(m.diffLines(), m.selectedFilePath(), max(1, m.diffViewport.Width)), "\n"))
}

func formatDiffLines(lines []string, path string, width int) []string {
	gutterWidth := diffGutterWidth(lines)
	codeWidth := max(1, width-gutterWidth-3)
	formatted := make([]string, 0, len(lines))
	oldLine := 0
	newLine := 0

	for _, line := range lines {
		if match := diffHunkPattern.FindStringSubmatch(line); match != nil {
			oldLine, _ = strconv.Atoi(match[1])
			newLine, _ = strconv.Atoi(match[2])
			formatted = append(formatted, diffBlankGutter(gutterWidth)+colorDiffLine(truncate(line, codeWidth), path))
			continue
		}

		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			formatted = append(formatted, diffBlankGutter(gutterWidth)+colorDiffLine(truncate(line, codeWidth), path))
		case strings.HasPrefix(line, "+"):
			formatted = append(formatted, diffLineGutter(newLine, gutterWidth)+colorDiffLine(truncate(line, codeWidth), path))
			newLine++
		case strings.HasPrefix(line, "-"):
			formatted = append(formatted, diffLineGutter(oldLine, gutterWidth)+colorDiffLine(truncate(line, codeWidth), path))
			oldLine++
		case strings.HasPrefix(line, " "):
			formatted = append(formatted, diffLineGutter(newLine, gutterWidth)+colorDiffLine(truncate(line, codeWidth), path))
			oldLine++
			newLine++
		default:
			formatted = append(formatted, diffBlankGutter(gutterWidth)+colorDiffLine(truncate(line, codeWidth), path))
		}
	}

	return formatted
}

func diffGutterWidth(lines []string) int {
	maxLine := 0
	for _, line := range lines {
		match := diffHunkPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		oldStart, _ := strconv.Atoi(match[1])
		newStart, _ := strconv.Atoi(match[2])
		maxLine = max(maxLine, max(oldStart, newStart))
	}
	return max(3, len(strconv.Itoa(maxLine)))
}

func diffLineGutter(line, width int) string {
	if line <= 0 {
		return diffBlankGutter(width)
	}
	return mutedStyle.Render(fmt.Sprintf("%*d │ ", width, line))
}

func diffBlankGutter(width int) string {
	return mutedStyle.Render(strings.Repeat(" ", width) + " │ ")
}
func keepIndexVisible(index, total, visibleCount int) int {
	offset := 0
	if index >= visibleCount {
		offset = index - visibleCount + 1
	}
	maxOffset := max(0, total-visibleCount)
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}
func (m model) selectedFilePath() string {
	if m.tab != changesTab || len(m.files) == 0 {
		return ""
	}
	return m.files[m.selected].Path
}
func (m model) previewTitle() string {
	switch m.tab {
	case changesTab:
		if len(m.files) == 0 {
			return ""
		}
		return m.files[m.selected].Path
	case branchesTab:
		if len(m.branches) == 0 {
			return ""
		}
		return m.branches[m.branchSelected].Name
	case commitsTab:
		if len(m.commits) == 0 {
			return ""
		}
		return m.commits[m.commitSelected].Hash
	case stashTab:
		if len(m.stashes) == 0 {
			return ""
		}
		return m.stashes[m.stashSelected].Name
	default:
		return ""
	}
}

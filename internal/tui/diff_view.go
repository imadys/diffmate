package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/imadys/diffmate/internal/git"
	"strings"
)

func (m model) renderDiff(width, height int) string {
	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	lines := m.diffLines()
	if strings.TrimSpace(m.diff) == "" {
		lines = []string{mutedStyle.Render("Select a changed file to view its diff.")}
	}

	m.clampDiffOffset()
	end := min(len(lines), m.diffOffset+innerHeight-1)
	if m.diffOffset > len(lines) {
		m.diffOffset = 0
	}
	visible := lines[m.diffOffset:end]

	header := subtleStyle.Render("Diff")
	title := m.previewTitle()
	if title != "" {
		scroll := fmt.Sprintf("  %d/%d", min(m.diffOffset+1, len(lines)), max(1, len(lines)))
		header = subtleStyle.Render(truncate(title, max(1, innerWidth-len(scroll)))) + mutedStyle.Render(scroll)
	}

	out := append([]string{header}, visible...)
	for i := 1; i < len(out); i++ {
		out[i] = colorDiffLine(truncate(out[i], innerWidth), m.selectedFilePath())
	}

	border := lipgloss.Color("238")
	if m.focus == diffFocus {
		border = lipgloss.Color("86")
	}

	return lipgloss.NewStyle().
		Width(innerWidth).
		Height(innerHeight).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Render(fitLines(out, innerHeight))
}
func (m model) diffHeight() int {
	return max(1, m.height-6)
}
func (m *model) clampDiffOffset() {
	maxOffset := max(0, len(m.diffLines())-m.diffHeight())
	if m.diffOffset > maxOffset {
		m.diffOffset = maxOffset
	}
	if m.diffOffset < 0 {
		m.diffOffset = 0
	}
}
func (m *model) scrollDiff(delta int) {
	m.diffOffset += delta
	m.clampDiffOffset()
}
func (m model) diffLines() []string {
	if m.diff == "" {
		return []string{""}
	}
	return strings.Split(m.diff, "\n")
}
func (m model) visibleFiles(count int) []git.FileStatus {
	if count <= 0 || len(m.files) == 0 {
		return nil
	}
	m.keepSelectedFileVisible()
	end := min(len(m.files), m.fileOffset+count)
	return m.files[m.fileOffset:end]
}
func (m *model) keepSelectedFileVisible() {
	if m.selected < m.fileOffset {
		m.fileOffset = m.selected
	}
	visibleCount := max(1, m.height-6)
	if m.selected >= m.fileOffset+visibleCount {
		m.fileOffset = m.selected - visibleCount + 1
	}
	maxOffset := max(0, len(m.files)-visibleCount)
	if m.fileOffset > maxOffset {
		m.fileOffset = maxOffset
	}
	if m.fileOffset < 0 {
		m.fileOffset = 0
	}
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

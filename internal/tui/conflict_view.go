package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/imadys/diffmate/internal/git"
)

func (m model) renderConflictView(width, height int) string {
	bodyGap := 1
	sidebarWidth := clamp(34, 24, width/3)
	viewerWidth := max(1, width-sidebarWidth-bodyGap)

	sidebar := m.renderConflictSidebar(sidebarWidth, height)
	viewer := m.renderConflictViewer(viewerWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, strings.Repeat(" ", bodyGap), viewer)
}

func (m model) renderConflictSidebar(width, height int) string {
	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	contentWidth := max(1, innerWidth-2)
	title := titleStyle.Render(fmt.Sprintf("Conflicts  %d", len(m.conflicts)))
	lines := []string{title}

	if len(m.conflicts) == 0 {
		if m.mergeInProgress {
			lines = append(lines, addStyle.Render("All files resolved"), mutedStyle.Render("press c to continue"))
		} else {
			lines = append(lines, mutedStyle.Render("No conflicts"))
		}
	} else {
		visibleCount := max(1, innerHeight-1)
		offset := keepIndexVisible(m.conflictSelected, len(m.conflicts), visibleCount)
		end := min(len(m.conflicts), offset+visibleCount)
		for i := offset; i < end; i++ {
			file := m.conflicts[i]
			line := truncate(conflictStatus(file)+" "+file.Path, contentWidth)
			if i == m.conflictSelected {
				line = selectedLineStyle(true, contentWidth).Render(line)
			}
			lines = append(lines, line)
		}
	}

	return lipgloss.NewStyle().
		Width(innerWidth).
		Height(max(1, height-2)).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(0, 1).
		Render(fitLines(lines, innerHeight))
}

func (m model) renderConflictViewer(width, height int) string {
	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	contentWidth := max(1, innerWidth-2)
	title := subtleStyle.Render("Merge view")
	path := m.selectedConflictPath()
	if path != "" {
		title = subtleStyle.Render(truncate(path, contentWidth))
	}

	lines := []string{title}
	content := formatConflictContent(m.conflictContent, path, contentWidth)
	if len(m.conflicts) == 0 && m.mergeInProgress {
		content = []string{
			addStyle.Render("All conflicts are staged."),
			mutedStyle.Render("Press c to continue the merge, or a to abort it."),
		}
	}
	vp := m.conflictViewport
	if vp.Width == 0 || vp.Height == 0 {
		vp = viewport.New(contentWidth, max(1, innerHeight-1))
	}
	vp.Width = contentWidth
	vp.Height = max(1, innerHeight-1)
	vp.SetContent(strings.Join(content, "\n"))
	lines = append(lines, vp.View())

	return lipgloss.NewStyle().
		Width(innerWidth).
		Height(innerHeight).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(0, 1).
		Render(fitLines(strings.Split(strings.Join(lines, "\n"), "\n"), innerHeight))
}

func formatConflictContent(content, path string, width int) []string {
	if strings.TrimSpace(content) == "" {
		return []string{mutedStyle.Render("Select a conflicted file.")}
	}
	lines := strings.Split(strings.ReplaceAll(content, "\t", "    "), "\n")
	formatted := make([]string, 0, len(lines))
	for _, line := range lines {
		line = truncate(line, width)
		switch {
		case strings.HasPrefix(line, "<<<<<<<"):
			formatted = append(formatted, delStyle.Bold(true).Render(line))
		case strings.HasPrefix(line, "======="):
			formatted = append(formatted, hunkStyle.Render(line))
		case strings.HasPrefix(line, ">>>>>>>"):
			formatted = append(formatted, addStyle.Bold(true).Render(line))
		default:
			formatted = append(formatted, highlightCode(line, path))
		}
	}
	return formatted
}

func conflictStatus(file git.FileStatus) string {
	status := strings.TrimSpace(string([]byte{file.Index, file.Worktree}))
	if status == "" {
		return "??"
	}
	return status
}

func conflictFiles(files []git.FileStatus) []git.FileStatus {
	conflicts := make([]git.FileStatus, 0)
	for _, file := range files {
		if file.IsConflict() {
			conflicts = append(conflicts, file)
		}
	}
	return conflicts
}

func (m model) selectedConflictPath() string {
	if len(m.conflicts) == 0 {
		return ""
	}
	return m.conflicts[m.conflictSelected].Path
}

func (m model) conflictHeight() int {
	return max(1, m.conflictViewport.Height)
}

func (m *model) syncConflictViewport() {
	if m.width == 0 || m.height == 0 {
		return
	}
	bodyWidth := max(1, m.width-2)
	headerHeight := 1
	footerHeight := 1
	bodyHeight := max(1, m.height-headerHeight-footerHeight)
	sidebarWidth := clamp(34, 24, bodyWidth/3)
	viewerWidth := max(1, bodyWidth-sidebarWidth-1)
	innerWidth := max(1, viewerWidth-4)
	innerHeight := max(1, bodyHeight-2)
	m.conflictViewport.Width = max(1, innerWidth-2)
	m.conflictViewport.Height = max(1, innerHeight-1)
}

func (m *model) scrollConflict(delta int) {
	m.syncConflictViewport()
	if delta > 0 {
		m.conflictViewport.ScrollDown(delta)
	} else if delta < 0 {
		m.conflictViewport.ScrollUp(-delta)
	}
}

func (m *model) moveConflictSelection(delta int) {
	if len(m.conflicts) == 0 {
		return
	}
	m.conflictSelected = clamp(m.conflictSelected+delta, 0, len(m.conflicts)-1)
	m.conflictViewport.GotoTop()
}

package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/imadys/diffmate/internal/git"
	"strings"
)

func (m model) renderSidebar(width, height int) string {
	tabs := m.visibleTabs()
	if len(tabs) == 0 {
		return panelStyle.Width(width).Height(height).Render(mutedStyle.Render("No sections visible"))
	}

	gapCount := max(0, len(tabs)-1)
	heights := splitHeights(max(1, height-gapCount), len(tabs))
	cards := make([]string, 0, len(tabs))
	for i, tab := range tabs {
		cards = append(cards, m.renderSectionCard(tab, width, heights[i]))
	}
	return strings.Join(cards, "\n")
}
func (m model) renderSectionCard(tab sidebarTab, width, height int) string {
	if height <= 0 {
		return ""
	}

	current := tab == m.tab
	focused := m.focus == sidebarFocus && current
	title := fmt.Sprintf("[%d] %s", int(tab)+1, sectionTitle(tab))
	if current {
		title = titleStyle.Render(title)
	} else {
		title = subtleStyle.Render(title)
	}

	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	lines := []string{title}
	switch tab {
	case changesTab:
		lines = append(lines, m.renderChangeItems(width, innerHeight-1, current, focused)...)
	case branchesTab:
		lines = append(lines, m.renderBranchItems(width, innerHeight-1, current, focused)...)
	case commitsTab:
		lines = append(lines, m.renderCommitItems(width, innerHeight-1, current, focused)...)
	case stashTab:
		lines = append(lines, m.renderStashItems(width, innerHeight-1, current, focused)...)
	}
	border := lipgloss.Color("238")
	if focused {
		border = lipgloss.Color("86")
	} else if current {
		border = lipgloss.Color("60")
	}

	return lipgloss.NewStyle().
		Width(innerWidth).
		Height(innerHeight).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Render(fitLines(lines, innerHeight))
}
func (m model) renderChangeItems(width, height int, current, focused bool) []string {
	if len(m.files) == 0 {
		return []string{mutedStyle.Render("No changes")}
	}

	visibleFiles := m.visibleFiles(height)
	lines := make([]string, 0, len(visibleFiles))
	for visibleIndex, file := range visibleFiles {
		i := m.fileOffset + visibleIndex
		line := m.renderFileLine(file, width-4)
		if current && i == m.selected {
			line = selectedLineStyle(focused, width-4).Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}
func (m model) renderBranchItems(width, height int, current, focused bool) []string {
	if len(m.branches) == 0 {
		return []string{mutedStyle.Render("No branches")}
	}
	lines := make([]string, 0, min(height, len(m.branches)))
	for i, branch := range m.branches[:min(height, len(m.branches))] {
		prefix := "  "
		if branch.Current {
			prefix = "* "
		}
		line := truncate(prefix+branch.Name, width-4)
		if current && i == m.branchSelected {
			line = selectedLineStyle(focused, width-4).Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}
func (m model) renderCommitItems(width, height int, current, focused bool) []string {
	if len(m.commits) == 0 {
		return []string{mutedStyle.Render("No commits")}
	}
	lines := make([]string, 0, min(height, len(m.commits)))
	for i, commit := range m.commits[:min(height, len(m.commits))] {
		line := truncate(commit.Hash+" "+commit.Subject, width-4)
		if current && i == m.commitSelected {
			line = selectedLineStyle(focused, width-4).Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}
func (m model) renderStashItems(width, height int, current, focused bool) []string {
	if len(m.stashes) == 0 {
		return []string{mutedStyle.Render("No stash entries")}
	}
	lines := make([]string, 0, min(height, len(m.stashes)))
	for i, stash := range m.stashes[:min(height, len(m.stashes))] {
		line := truncate(stash.Name+" "+stash.Subject, width-4)
		if current && i == m.stashSelected {
			line = selectedLineStyle(focused, width-4).Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}
func (m model) renderFileLine(file git.FileStatus, width int) string {
	status := strings.TrimSpace(string([]byte{file.Index, file.Worktree}))
	if status == "" {
		status = "??"
	}

	statusStyle := codeMuted
	if strings.Contains(status, "A") || status == "??" {
		statusStyle = addStyle
	}
	if strings.Contains(status, "D") {
		statusStyle = delStyle
	}

	label := statusStyle.Bold(true).Render(fmt.Sprintf("%-2s", status))
	path := truncate(file.Path, max(1, width-lipgloss.Width(label)-1))
	return label + " " + path
}
func selectedLineStyle(focused bool, width int) lipgloss.Style {
	if focused {
		return selectedStyle.Width(width)
	}
	return linkedStyle.Width(width)
}

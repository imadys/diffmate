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

	heights := splitHeights(height, len(tabs))
	cards := make([]string, 0, len(tabs))
	for i, tab := range tabs {
		cards = append(cards, m.renderSectionCard(tab, width, heights[i]))
	}
	return fitLines(strings.Split(strings.Join(cards, "\n"), "\n"), height)
}
func (m model) renderSectionCard(tab sidebarTab, width, height int) string {
	if height <= 0 {
		return ""
	}

	current := tab == m.tab
	focused := m.focus == sidebarFocus && current
	title := fmt.Sprintf("[%d] %s", int(tab)+1, sectionTitle(tab))
	if count, total := m.sectionPosition(tab); total > 0 {
		title += fmt.Sprintf("  %d/%d", count, total)
	}
	if tab == m.tab && m.searchQuery != "" {
		title += "  /" + m.searchQuery
	}
	if current {
		title = titleStyle.Render(title)
	} else {
		title = subtleStyle.Render(title)
	}

	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	contentWidth := max(1, innerWidth-2)
	lines := []string{title}
	switch tab {
	case changesTab:
		lines = append(lines, m.renderChangeItems(contentWidth, innerHeight-1, current, focused)...)
	case branchesTab:
		lines = append(lines, m.renderBranchItems(contentWidth, innerHeight-1, current, focused)...)
	case commitsTab:
		lines = append(lines, m.renderCommitItems(contentWidth, innerHeight-1, current, focused)...)
	case stashTab:
		lines = append(lines, m.renderStashItems(contentWidth, innerHeight-1, current, focused)...)
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

	indices := m.filteredSidebarIndices(changesTab)
	if len(indices) == 0 {
		return []string{mutedStyle.Render("No matches")}
	}
	selectedPosition := indexOfInt(indices, m.selected)
	if selectedPosition < 0 {
		selectedPosition = 0
	}
	offset := keepIndexVisible(selectedPosition, len(indices), height)
	end := min(len(indices), offset+height)
	visibleIndices := indices[offset:end]
	lines := make([]string, 0, len(visibleIndices))
	for _, i := range visibleIndices {
		file := m.files[i]
		line := m.renderFileLine(file, width)
		if current && i == m.selected {
			line = m.renderPlainFileLine(file, width)
			line = selectedLineStyle(focused, width).Render(line)
		}
		lines = append(lines, line)
	}
	return fitLineSlice(lines, height)
}
func (m model) renderBranchItems(width, height int, current, focused bool) []string {
	if len(m.branches) == 0 {
		return []string{mutedStyle.Render("No branches")}
	}
	indices := m.filteredSidebarIndices(branchesTab)
	if len(indices) == 0 {
		return []string{mutedStyle.Render("No matches")}
	}
	selectedPosition := indexOfInt(indices, m.branchSelected)
	if selectedPosition < 0 {
		selectedPosition = 0
	}
	offset := keepIndexVisible(selectedPosition, len(indices), height)
	end := min(len(indices), offset+height)
	lines := make([]string, 0, max(0, end-offset))
	for _, index := range indices[offset:end] {
		branch := m.branches[index]
		prefix := "  "
		if branch.Current {
			prefix = "* "
		}
		line := truncate(prefix+branch.Name, width)
		if current && index == m.branchSelected {
			line = selectedLineStyle(focused, width).Render(line)
		}
		lines = append(lines, line)
	}
	return fitLineSlice(lines, height)
}
func (m model) renderCommitItems(width, height int, current, focused bool) []string {
	if len(m.commits) == 0 {
		return []string{mutedStyle.Render("No commits")}
	}
	indices := m.filteredSidebarIndices(commitsTab)
	if len(indices) == 0 {
		return []string{mutedStyle.Render("No matches")}
	}
	selectedPosition := indexOfInt(indices, m.commitSelected)
	if selectedPosition < 0 {
		selectedPosition = 0
	}
	offset := keepIndexVisible(selectedPosition, len(indices), height)
	end := min(len(indices), offset+height)
	lines := make([]string, 0, max(0, end-offset))
	for _, index := range indices[offset:end] {
		commit := m.commits[index]
		line := truncate(commit.Hash+" "+commit.Subject, width)
		if current && index == m.commitSelected {
			line = selectedLineStyle(focused, width).Render(line)
		}
		lines = append(lines, line)
	}
	return fitLineSlice(lines, height)
}
func (m model) renderStashItems(width, height int, current, focused bool) []string {
	if len(m.stashes) == 0 {
		return []string{mutedStyle.Render("No stash entries")}
	}
	indices := m.filteredSidebarIndices(stashTab)
	if len(indices) == 0 {
		return []string{mutedStyle.Render("No matches")}
	}
	selectedPosition := indexOfInt(indices, m.stashSelected)
	if selectedPosition < 0 {
		selectedPosition = 0
	}
	offset := keepIndexVisible(selectedPosition, len(indices), height)
	end := min(len(indices), offset+height)
	lines := make([]string, 0, max(0, end-offset))
	for _, index := range indices[offset:end] {
		stash := m.stashes[index]
		line := truncate(stash.Name+" "+stash.Subject, width)
		if current && index == m.stashSelected {
			line = selectedLineStyle(focused, width).Render(line)
		}
		lines = append(lines, line)
	}
	return fitLineSlice(lines, height)
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
func (m model) renderPlainFileLine(file git.FileStatus, width int) string {
	status := strings.TrimSpace(string([]byte{file.Index, file.Worktree}))
	if status == "" {
		status = "??"
	}
	return truncate(fmt.Sprintf("%-2s %s", status, file.Path), width)
}
func selectedLineStyle(focused bool, width int) lipgloss.Style {
	if focused {
		return selectedStyle.Width(width)
	}
	return linkedStyle.Width(width)
}

func (m model) sectionPosition(tab sidebarTab) (int, int) {
	switch tab {
	case changesTab:
		indices := m.filteredSidebarIndices(tab)
		if len(indices) == 0 {
			return 0, 0
		}
		position := indexOfInt(indices, m.selected)
		if position < 0 {
			return 0, len(indices)
		}
		return position + 1, len(indices)
	case branchesTab:
		indices := m.filteredSidebarIndices(tab)
		if len(indices) == 0 {
			return 0, 0
		}
		position := indexOfInt(indices, m.branchSelected)
		if position < 0 {
			return 0, len(indices)
		}
		return position + 1, len(indices)
	case commitsTab:
		indices := m.filteredSidebarIndices(tab)
		if len(indices) == 0 {
			return 0, 0
		}
		position := indexOfInt(indices, m.commitSelected)
		if position < 0 {
			return 0, len(indices)
		}
		return position + 1, len(indices)
	case stashTab:
		indices := m.filteredSidebarIndices(tab)
		if len(indices) == 0 {
			return 0, 0
		}
		position := indexOfInt(indices, m.stashSelected)
		if position < 0 {
			return 0, len(indices)
		}
		return position + 1, len(indices)
	default:
		return 0, 0
	}
}

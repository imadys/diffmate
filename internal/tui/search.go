package tui

import (
	"fmt"
	"strings"
)

func (m model) filteredSidebarIndices(tab sidebarTab) []int {
	total := m.sidebarItemCount(tab)
	indices := make([]int, 0, total)
	query := strings.ToLower(strings.TrimSpace(m.searchQuery))
	for i := 0; i < total; i++ {
		if query == "" || tab != m.tab || strings.Contains(strings.ToLower(m.searchLabel(tab, i)), query) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m model) searchLabel(tab sidebarTab, index int) string {
	switch tab {
	case changesTab:
		if index >= 0 && index < len(m.files) {
			file := m.files[index]
			return fmt.Sprintf("%c%c %s", file.Index, file.Worktree, file.Path)
		}
	case branchesTab:
		if index >= 0 && index < len(m.branches) {
			return m.branches[index].Name
		}
	case commitsTab:
		if index >= 0 && index < len(m.commits) {
			commit := m.commits[index]
			return commit.Hash + " " + commit.Subject
		}
	case stashTab:
		if index >= 0 && index < len(m.stashes) {
			stash := m.stashes[index]
			return stash.Name + " " + stash.Subject
		}
	}
	return ""
}

func (m model) sidebarItemCount(tab sidebarTab) int {
	switch tab {
	case changesTab:
		return len(m.files)
	case branchesTab:
		return len(m.branches)
	case commitsTab:
		return len(m.commits)
	case stashTab:
		return len(m.stashes)
	default:
		return 0
	}
}

func (m *model) selectFirstSearchMatch() {
	indices := m.filteredSidebarIndices(m.tab)
	if len(indices) == 0 {
		return
	}
	m.setSidebarSelection(indices[0])
	m.diffOffset = 0
	m.diffViewport.GotoTop()
}

func (m *model) moveFilteredSidebarSelection(delta int) {
	indices := m.filteredSidebarIndices(m.tab)
	if len(indices) == 0 {
		return
	}
	current := m.sidebarSelection()
	position := indexOfInt(indices, current)
	if position < 0 {
		position = 0
	} else {
		position = clamp(position+delta, 0, len(indices)-1)
	}
	m.setSidebarSelection(indices[position])
}

func (m model) sidebarSelection() int {
	switch m.tab {
	case changesTab:
		return m.selected
	case branchesTab:
		return m.branchSelected
	case commitsTab:
		return m.commitSelected
	case stashTab:
		return m.stashSelected
	default:
		return 0
	}
}

func (m *model) setSidebarSelection(index int) {
	switch m.tab {
	case changesTab:
		m.selected = clamp(index, 0, max(0, len(m.files)-1))
	case branchesTab:
		m.branchSelected = clamp(index, 0, max(0, len(m.branches)-1))
	case commitsTab:
		m.commitSelected = clamp(index, 0, max(0, len(m.commits)-1))
	case stashTab:
		m.stashSelected = clamp(index, 0, max(0, len(m.stashes)-1))
	}
}

func indexOfInt(values []int, target int) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}

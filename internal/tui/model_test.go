package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/imadys/diffmate/internal/git"
)

func TestRefreshShowsConflictModeAfterMergePickerConflict(t *testing.T) {
	m := model{mode: mergePickerMode}

	updated, _ := m.Update(refreshMsg{
		files: []git.FileStatus{{
			Path:     "notes.txt",
			Index:    'U',
			Worktree: 'U',
		}},
		mergeInProgress: true,
		conflictContent: "one\n<<<<<<< HEAD\nours\n=======\ntheirs\n>>>>>>> branch\n",
	})
	got := updated.(model)

	if got.mode != conflictMode {
		t.Fatalf("expected conflict mode, got %v", got.mode)
	}
	if len(got.conflicts) != 1 {
		t.Fatalf("expected one conflict, got %d", len(got.conflicts))
	}
}

func TestRefreshKeepsConflictModeWhenAllConflictsStaged(t *testing.T) {
	m := model{mode: conflictMode}

	updated, _ := m.Update(refreshMsg{mergeInProgress: true})
	got := updated.(model)

	if got.mode != conflictMode {
		t.Fatalf("expected conflict mode, got %v", got.mode)
	}
}

func TestSearchSelectsFirstMatchingChange(t *testing.T) {
	m := model{
		tab: sidebarTab(changesTab),
		files: []git.FileStatus{
			{Path: "src/messages/ar.json", Index: ' ', Worktree: 'M'},
			{Path: "src/messages/en.json", Index: ' ', Worktree: 'M'},
			{Path: "README.md", Index: ' ', Worktree: 'M'},
		},
		searchQuery: "en",
	}

	m.selectFirstSearchMatch()

	if m.selected != 1 {
		t.Fatalf("expected en.json to be selected, got %d", m.selected)
	}
}

func TestFilteredMergeIndicesExcludeCurrentAndApplySearch(t *testing.T) {
	m := model{
		branches: []git.Branch{
			{Name: "main", Current: true},
			{Name: "feat/payments"},
			{Name: "fix/navbar"},
		},
		mergeSearch: "pay",
	}

	indices := m.filteredMergeIndices()
	if len(indices) != 1 || indices[0] != 1 {
		t.Fatalf("expected only payments branch, got %#v", indices)
	}
}

func TestMoveMergeSelectionUsesFilteredBranches(t *testing.T) {
	m := model{
		branches: []git.Branch{
			{Name: "main", Current: true},
			{Name: "feat/payments"},
			{Name: "fix/navbar"},
		},
		mergeSelected: 1,
	}

	m.moveMergeSelection(1)

	if m.mergeSelected != 2 {
		t.Fatalf("expected merge selection to move to branch 2, got %d", m.mergeSelected)
	}
}

func TestCommitCopyTextPrefersVisibleError(t *testing.T) {
	m := model{
		commitMessage: "feat: improve modal",
		commitError:   "claude could not suggest a commit message",
	}

	if got := m.commitCopyText(); got != m.commitError {
		t.Fatalf("expected error to be copied, got %q", got)
	}
}

var _ tea.Model = model{}

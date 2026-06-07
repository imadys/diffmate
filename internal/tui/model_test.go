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

var _ tea.Model = model{}

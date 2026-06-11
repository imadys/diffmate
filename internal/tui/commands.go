package tui

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/imadys/diffmate/internal/git"
	"github.com/imadys/diffmate/internal/suggest"
	"strings"
)

func (m model) refresh() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		files, err := m.repo.Status(ctx)
		if err != nil {
			return refreshMsg{err: err}
		}
		branches, err := m.repo.Branches(ctx)
		if err != nil {
			return refreshMsg{files: files, err: err}
		}
		commits, err := m.repo.Commits(ctx)
		if err != nil {
			return refreshMsg{files: files, branches: branches, err: err}
		}
		stashes, err := m.repo.Stashes(ctx)
		if err != nil {
			return refreshMsg{files: files, branches: branches, commits: commits, err: err}
		}

		m.files = files
		m.branches = branches
		m.commits = commits
		m.stashes = stashes
		mergeInProgress := m.repo.MergeInProgress(ctx)
		conflicts := conflictFiles(files)
		m.clampSidebarSelections()

		conflictContent := ""
		if len(conflicts) > 0 {
			selected := clamp(m.conflictSelected, 0, len(conflicts)-1)
			conflictContent, err = m.repo.FileContent(ctx, conflicts[selected])
			if err != nil {
				return refreshMsg{files: files, branches: branches, commits: commits, stashes: stashes, mergeInProgress: mergeInProgress, err: err}
			}
		}

		diff, err := m.preview(ctx)
		if err != nil {
			return refreshMsg{files: files, branches: branches, commits: commits, stashes: stashes, mergeInProgress: mergeInProgress, err: err}
		}

		return refreshMsg{files: files, branches: branches, commits: commits, stashes: stashes, diff: diff, conflictContent: conflictContent, mergeInProgress: mergeInProgress}
	}
}
func (m model) loadDiff() tea.Cmd {
	return func() tea.Msg {
		diff, err := m.preview(context.Background())
		return refreshMsg{
			files:           m.files,
			branches:        m.branches,
			commits:         m.commits,
			stashes:         m.stashes,
			diff:            diff,
			mergeInProgress: m.mergeInProgress,
			err:             err,
		}
	}
}
func (m model) loadConflictContent() tea.Cmd {
	return func() tea.Msg {
		if len(m.conflicts) == 0 {
			return refreshMsg{
				files:           m.files,
				branches:        m.branches,
				commits:         m.commits,
				stashes:         m.stashes,
				diff:            m.diff,
				mergeInProgress: m.mergeInProgress,
			}
		}
		content, err := m.repo.FileContent(context.Background(), m.conflicts[m.conflictSelected])
		return refreshMsg{
			files:           m.files,
			branches:        m.branches,
			commits:         m.commits,
			stashes:         m.stashes,
			diff:            m.diff,
			conflictContent: content,
			mergeInProgress: m.mergeInProgress,
			err:             err,
		}
	}
}
func (m model) preview(ctx context.Context) (string, error) {
	switch m.tab {
	case changesTab:
		if len(m.files) == 0 {
			return "", nil
		}
		return m.repo.Diff(ctx, m.files[m.selected])
	case branchesTab:
		if len(m.branches) == 0 {
			return "", nil
		}
		return m.repo.BranchPreview(ctx, m.branches[m.branchSelected])
	case commitsTab:
		if len(m.commits) == 0 {
			return "", nil
		}
		return m.repo.CommitDiff(ctx, m.commits[m.commitSelected])
	case stashTab:
		if len(m.stashes) == 0 {
			return "", nil
		}
		return m.repo.StashDiff(ctx, m.stashes[m.stashSelected])
	default:
		return "", nil
	}
}
func (m model) stage() tea.Cmd {
	if m.tab != changesTab {
		return nil
	}
	return m.withSelected("staged", func(ctx context.Context, file git.FileStatus) error {
		return m.repo.Stage(ctx, file)
	})
}
func (m model) toggleStage() tea.Cmd {
	if m.tab != changesTab || len(m.files) == 0 {
		return nil
	}
	file := m.files[m.selected]
	if file.Index != ' ' && file.Index != '?' {
		return m.unstage()
	}
	return m.stage()
}
func (m model) unstage() tea.Cmd {
	if m.tab != changesTab {
		return nil
	}
	return m.withSelected("unstaged", func(ctx context.Context, file git.FileStatus) error {
		return m.repo.Unstage(ctx, file)
	})
}
func (m model) stageAll() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.StageAll(context.Background())
		return actionMsg{status: "staged all changes", err: err}
	}
}
func (m model) unstageAll() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.UnstageAll(context.Background())
		return actionMsg{status: "unstaged all changes", err: err}
	}
}
func (m model) stashChanges() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.Stash(context.Background())
		return actionMsg{status: "stashed changes", err: err}
	}
}
func (m model) discardChange() tea.Cmd {
	return m.withSelected("discarded", func(ctx context.Context, file git.FileStatus) error {
		return m.repo.Discard(ctx, file)
	})
}
func (m model) checkoutBranch() tea.Cmd {
	if m.tab != branchesTab || len(m.branches) == 0 {
		return nil
	}
	branch := m.branches[m.branchSelected]
	return func() tea.Msg {
		err := m.repo.CheckoutBranch(context.Background(), branch)
		return actionMsg{status: "checked out " + branch.Name, err: err}
	}
}
func (m model) createBranch() tea.Cmd {
	name := normalizeBranchName(m.branchNameInput)
	return func() tea.Msg {
		err := m.repo.CreateBranch(context.Background(), name)
		return actionMsg{status: "created branch " + name, err: err}
	}
}
func (m model) deleteBranch() tea.Cmd {
	if m.tab != branchesTab || len(m.branches) == 0 {
		return nil
	}
	branch := m.branches[m.branchSelected]
	return func() tea.Msg {
		err := m.repo.DeleteBranch(context.Background(), branch)
		return actionMsg{status: "deleted branch " + branch.Name, err: err}
	}
}
func (m model) deleteRemoteBranch() tea.Cmd {
	if m.tab != branchesTab || len(m.branches) == 0 {
		return nil
	}
	branch := m.branches[m.branchSelected]
	return func() tea.Msg {
		err := m.repo.DeleteRemoteBranch(context.Background(), branch)
		return actionMsg{status: "deleted origin/" + branch.Name, err: err}
	}
}
func (m model) deleteBranchBoth() tea.Cmd {
	if m.tab != branchesTab || len(m.branches) == 0 {
		return nil
	}
	branch := m.branches[m.branchSelected]
	return func() tea.Msg {
		if err := m.repo.DeleteRemoteBranch(context.Background(), branch); err != nil {
			return actionMsg{status: "delete failed", err: err}
		}
		err := m.repo.DeleteBranch(context.Background(), branch)
		return actionMsg{status: "deleted " + branch.Name + " locally and remotely", err: err}
	}
}
func (m model) mergeSelectedBranch() tea.Cmd {
	indices := m.filteredMergeIndices()
	if len(indices) == 0 {
		return nil
	}
	selected := m.mergeSelected
	if indexOfInt(indices, selected) < 0 {
		selected = indices[0]
	}
	branch := m.branches[selected]
	target := m.currentBranchName()
	return func() tea.Msg {
		output, err := m.repo.MergeBranch(context.Background(), branch)
		if err != nil {
			return actionMsg{status: "merge failed", err: err}
		}
		status := "merged " + branch.Name + " into " + target
		if strings.Contains(strings.ToLower(output), "already up to date") {
			status = branch.Name + " already up to date"
		}
		return actionMsg{status: status, err: nil}
	}
}
func (m model) updateFromUpstream() tea.Cmd {
	current := m.currentBranchName()
	return func() tea.Msg {
		err := m.repo.PullUpstream(context.Background())
		return actionMsg{status: "updated " + current + " from upstream", err: err}
	}
}
func (m model) commit() tea.Cmd {
	message := strings.TrimSpace(m.commitMessage)
	return func() tea.Msg {
		err := m.repo.Commit(context.Background(), message)
		return actionMsg{status: "commit created", err: err}
	}
}
func (m model) push() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.Push(context.Background())
		return actionMsg{status: "pushed current branch", err: err}
	}
}
func (m model) openPreferredEditor() tea.Cmd {
	command := m.settings.Editor
	return tea.ExecProcess(projectCommand(m.repo.Root, command), func(err error) tea.Msg {
		return actionMsg{status: "opened project in " + command, err: err}
	})
}
func (m model) openPreferredAgent() tea.Cmd {
	command := m.settings.Agent
	return tea.ExecProcess(agentCommand(m.repo.Root, command), func(err error) tea.Msg {
		return actionMsg{status: "opened " + command, err: err}
	})
}
func (m model) suggestCommitMessage() tea.Cmd {
	return func() tea.Msg {
		diff, err := m.repo.CommitMessageDiff(context.Background())
		if err != nil {
			return suggestMsg{err: err}
		}
		message, err := suggest.CommitMessage(context.Background(), m.repo.Root, m.settings.Agent, diff)
		return suggestMsg{message: message, err: err}
	}
}
func (m model) openEditor() tea.Cmd {
	if m.tab != changesTab || len(m.files) == 0 {
		return nil
	}
	file := m.files[m.selected]
	editor := m.settings.Editor
	return tea.ExecProcess(preferredEditorFileCommand(m.repo.Root, editor, file.Path), func(err error) tea.Msg {
		return actionMsg{status: "opened " + file.Path + " in " + editor, err: err}
	})
}
func (m model) acceptOurs() tea.Cmd {
	return m.withConflict("accepted ours", func(ctx context.Context, file git.FileStatus) error {
		return m.repo.CheckoutOurs(ctx, file)
	})
}
func (m model) acceptTheirs() tea.Cmd {
	return m.withConflict("accepted theirs", func(ctx context.Context, file git.FileStatus) error {
		return m.repo.CheckoutTheirs(ctx, file)
	})
}
func (m model) stageConflict() tea.Cmd {
	return m.withConflict("marked resolved", func(ctx context.Context, file git.FileStatus) error {
		return m.repo.Stage(ctx, file)
	})
}
func (m model) abortMerge() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.AbortMerge(context.Background())
		return actionMsg{status: "merge aborted", err: err}
	}
}
func (m model) continueMerge() tea.Cmd {
	return func() tea.Msg {
		if len(m.conflicts) > 0 {
			return actionMsg{status: "merge blocked", err: fmt.Errorf("%d unresolved conflict(s)", len(m.conflicts))}
		}
		err := m.repo.ContinueMerge(context.Background())
		return actionMsg{status: "merge continued", err: err}
	}
}
func (m model) openConflictEditor() tea.Cmd {
	if len(m.conflicts) == 0 {
		return nil
	}
	file := m.conflicts[m.conflictSelected]
	editor := m.settings.Editor
	return tea.ExecProcess(preferredEditorFileCommand(m.repo.Root, editor, file.Path), func(err error) tea.Msg {
		return actionMsg{status: "opened " + file.Path + " in " + editor, err: err}
	})
}
func (m model) withSelected(status string, fn func(context.Context, git.FileStatus) error) tea.Cmd {
	if len(m.files) == 0 {
		return nil
	}
	file := m.files[m.selected]
	return func() tea.Msg {
		err := fn(context.Background(), file)
		return actionMsg{status: fmt.Sprintf("%s %s", status, file.Path), err: err}
	}
}
func (m model) withConflict(status string, fn func(context.Context, git.FileStatus) error) tea.Cmd {
	if len(m.conflicts) == 0 {
		return nil
	}
	file := m.conflicts[m.conflictSelected]
	return func() tea.Msg {
		err := fn(context.Background(), file)
		return actionMsg{status: fmt.Sprintf("%s %s", status, file.Path), err: err}
	}
}

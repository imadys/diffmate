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
		m.clampSidebarSelections()

		diff, err := m.preview(ctx)
		if err != nil {
			return refreshMsg{files: files, branches: branches, commits: commits, stashes: stashes, err: err}
		}

		return refreshMsg{files: files, branches: branches, commits: commits, stashes: stashes, diff: diff}
	}
}
func (m model) loadDiff() tea.Cmd {
	return func() tea.Msg {
		diff, err := m.preview(context.Background())
		return refreshMsg{
			files:    m.files,
			branches: m.branches,
			commits:  m.commits,
			stashes:  m.stashes,
			diff:     diff,
			err:      err,
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
	return tea.ExecProcess(editorCommand(m.repo.Root, file.Path), func(err error) tea.Msg {
		return actionMsg{status: "editor closed", err: err}
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

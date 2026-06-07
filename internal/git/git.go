package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrNotRepository = errors.New("not a Git repository")

type FileStatus struct {
	Path     string
	OldPath  string
	Index    byte
	Worktree byte
}

func (f FileStatus) Label() string {
	status := strings.TrimSpace(string([]byte{f.Index, f.Worktree}))
	if status == "" {
		status = "??"
	}
	return fmt.Sprintf("%-2s %s", status, f.Path)
}

func (f FileStatus) IsUntracked() bool {
	return f.Index == '?' && f.Worktree == '?'
}

type Repo struct {
	Root string
}

type Branch struct {
	Name    string
	Current bool
}

type Commit struct {
	Hash    string
	Subject string
}

type Stash struct {
	Name    string
	Subject string
}

func Open(ctx context.Context) (Repo, error) {
	return OpenInDir(ctx, "")
}

func OpenInDir(ctx context.Context, dir string) (Repo, error) {
	out, err := run(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return Repo{}, ErrNotRepository
	}
	return Repo{Root: strings.TrimSpace(out)}, nil
}

func Init(ctx context.Context, dir, name string) (Repo, error) {
	if _, err := run(ctx, dir, "init"); err != nil {
		return Repo{}, err
	}

	name = strings.TrimSpace(name)
	if name != "" {
		description := filepath.Join(dir, ".git", "description")
		if err := os.WriteFile(description, []byte(name+"\n"), 0o644); err != nil {
			return Repo{}, err
		}
	}

	return Repo{Root: dir}, nil
}

func (r Repo) Status(ctx context.Context) ([]FileStatus, error) {
	out, err := run(ctx, r.Root, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	return parseStatus(out), nil
}

func (r Repo) Branches(ctx context.Context) ([]Branch, error) {
	out, err := run(ctx, r.Root, "branch", "--format=%(HEAD) %(refname:short)")
	if err != nil {
		return nil, err
	}

	var branches []Branch
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		branches = append(branches, Branch{
			Current: strings.HasPrefix(line, "*"),
			Name:    strings.TrimSpace(strings.TrimPrefix(line, "*")),
		})
	}
	return branches, nil
}

func (r Repo) Commits(ctx context.Context) ([]Commit, error) {
	out, err := run(ctx, r.Root, "log", "--oneline", "--decorate", "-n", "50")
	if err != nil {
		if strings.Contains(err.Error(), "does not have any commits") {
			return nil, nil
		}
		return nil, err
	}

	var commits []Commit
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		hash, subject, _ := strings.Cut(line, " ")
		commits = append(commits, Commit{Hash: hash, Subject: subject})
	}
	return commits, nil
}

func (r Repo) Stashes(ctx context.Context) ([]Stash, error) {
	out, err := run(ctx, r.Root, "stash", "list")
	if err != nil {
		return nil, err
	}

	var stashes []Stash
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		name, subject, _ := strings.Cut(line, ": ")
		stashes = append(stashes, Stash{Name: name, Subject: subject})
	}
	return stashes, nil
}

func (r Repo) Diff(ctx context.Context, file FileStatus) (string, error) {
	if file.IsUntracked() {
		return runAllowExitOne(ctx, r.Root, "diff", "--no-index", "--", os.DevNull, file.Path)
	}

	args := []string{"diff", "--", file.Path}
	if file.Index != ' ' && file.Worktree == ' ' {
		args = []string{"diff", "--cached", "--", file.Path}
	}

	out, err := run(ctx, r.Root, args...)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(out) == "" && file.Index != ' ' {
		out, err = run(ctx, r.Root, "diff", "--cached", "--", file.Path)
		if err != nil {
			return "", err
		}
	}
	return out, nil
}

func (r Repo) BranchPreview(ctx context.Context, branch Branch) (string, error) {
	if branch.Name == "" {
		return "", nil
	}
	return run(ctx, r.Root, "log", "--oneline", "--decorate", "-n", "30", branch.Name)
}

func (r Repo) CommitDiff(ctx context.Context, commit Commit) (string, error) {
	if commit.Hash == "" {
		return "", nil
	}
	return run(ctx, r.Root, "show", "--stat", "--patch", "--color=never", commit.Hash)
}

func (r Repo) StashDiff(ctx context.Context, stash Stash) (string, error) {
	if stash.Name == "" {
		return "", nil
	}
	return run(ctx, r.Root, "stash", "show", "--patch", stash.Name)
}

func (r Repo) Stage(ctx context.Context, file FileStatus) error {
	_, err := run(ctx, r.Root, "add", "--", file.Path)
	return err
}

func (r Repo) StageAll(ctx context.Context) error {
	_, err := run(ctx, r.Root, "add", "--all")
	return err
}

func (r Repo) Unstage(ctx context.Context, file FileStatus) error {
	if file.Index == 'A' || file.Index == '?' {
		_, err := run(ctx, r.Root, "rm", "--cached", "-r", "--", file.Path)
		return err
	}

	_, err := run(ctx, r.Root, "restore", "--staged", "--", file.Path)
	return err
}

func (r Repo) UnstageAll(ctx context.Context) error {
	if r.HasHead(ctx) {
		_, err := run(ctx, r.Root, "restore", "--staged", "--", ".")
		return err
	}

	_, err := run(ctx, r.Root, "rm", "--cached", "-r", "--", ".")
	return err
}

func (r Repo) Commit(ctx context.Context, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return errors.New("commit message cannot be empty")
	}

	hasStagedChanges, err := r.HasStagedChanges(ctx)
	if err != nil {
		return err
	}
	if !hasStagedChanges {
		return errors.New("nothing staged to commit; stage files first with s or S")
	}

	_, err = run(ctx, r.Root, "commit", "-m", message)
	return err
}

func (r Repo) Push(ctx context.Context) error {
	_, err := run(ctx, r.Root, "push")
	return err
}

func (r Repo) Stash(ctx context.Context) error {
	_, err := run(ctx, r.Root, "stash", "push", "-u")
	return err
}

func (r Repo) Discard(ctx context.Context, file FileStatus) error {
	if file.IsUntracked() {
		_, err := run(ctx, r.Root, "clean", "-fd", "--", file.Path)
		return err
	}

	args := []string{"restore", "--staged", "--worktree", "--", file.Path}
	if r.HasHead(ctx) {
		args = []string{"restore", "--source=HEAD", "--staged", "--worktree", "--", file.Path}
	}
	_, err := run(ctx, r.Root, args...)
	return err
}

func (r Repo) CheckoutBranch(ctx context.Context, branch Branch) error {
	if branch.Name == "" {
		return errors.New("branch name cannot be empty")
	}
	if branch.Current {
		return nil
	}
	_, err := run(ctx, r.Root, "checkout", branch.Name)
	return err
}

func (r Repo) CreateBranch(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("branch name cannot be empty")
	}
	_, err := run(ctx, r.Root, "checkout", "-b", name)
	return err
}

func (r Repo) DeleteBranch(ctx context.Context, branch Branch) error {
	if branch.Name == "" {
		return errors.New("branch name cannot be empty")
	}
	if branch.Current {
		return errors.New("cannot delete the current branch")
	}
	_, err := run(ctx, r.Root, "branch", "-d", branch.Name)
	return err
}

func (r Repo) HasStagedChanges(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--quiet", "--exit-code")
	cmd.Dir = r.Root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return false, nil
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return true, nil
	}
	if message := strings.TrimSpace(stderr.String()); message != "" {
		return false, errors.New(message)
	}
	return false, err
}

func (r Repo) CommitMessageDiff(ctx context.Context) (string, error) {
	out, err := run(ctx, r.Root, "diff", "--cached", "--stat", "--patch")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(out) != "" {
		return out, nil
	}

	return run(ctx, r.Root, "diff", "--stat", "--patch")
}

func (r Repo) HasHead(ctx context.Context) bool {
	_, err := run(ctx, r.Root, "rev-parse", "--verify", "HEAD")
	return err == nil
}

func parseStatus(out string) []FileStatus {
	if out == "" {
		return nil
	}

	parts := strings.Split(out, "\x00")
	files := make([]FileStatus, 0, len(parts))
	for i := 0; i < len(parts); i++ {
		entry := parts[i]
		if len(entry) < 4 {
			continue
		}

		file := FileStatus{
			Index:    entry[0],
			Worktree: entry[1],
			Path:     entry[3:],
		}

		if file.Index == 'R' || file.Index == 'C' {
			if i+1 < len(parts) {
				file.OldPath = parts[i+1]
				i++
			}
		}

		files = append(files, file)
	}

	return files
}

func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return stdout.String(), errors.New(message)
	}

	return stdout.String(), nil
}

func runAllowExitOne(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return stdout.String(), nil
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return stdout.String(), nil
	}

	message := strings.TrimSpace(stderr.String())
	if message == "" {
		message = err.Error()
	}
	return stdout.String(), errors.New(message)
}

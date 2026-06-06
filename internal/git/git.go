package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

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

func Open(ctx context.Context) (Repo, error) {
	out, err := run(ctx, "", "rev-parse", "--show-toplevel")
	if err != nil {
		return Repo{}, errors.New("diffmate must be run inside a Git repository")
	}
	return Repo{Root: strings.TrimSpace(out)}, nil
}

func (r Repo) Status(ctx context.Context) ([]FileStatus, error) {
	out, err := run(ctx, r.Root, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	return parseStatus(out), nil
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

	_, err := run(ctx, r.Root, "commit", "-m", message)
	return err
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

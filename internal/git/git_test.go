package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStatus(t *testing.T) {
	out := " M README.md\x00A  main.go\x00?? scratch.txt\x00R  new.go\x00old.go\x00"

	files := parseStatus(out)
	if len(files) != 4 {
		t.Fatalf("expected 4 files, got %d", len(files))
	}

	if files[0].Path != "README.md" || files[0].Index != ' ' || files[0].Worktree != 'M' {
		t.Fatalf("unexpected modified file: %#v", files[0])
	}

	if !files[2].IsUntracked() {
		t.Fatalf("expected scratch.txt to be untracked: %#v", files[2])
	}

	if files[3].Path != "new.go" || files[3].OldPath != "old.go" {
		t.Fatalf("unexpected renamed file: %#v", files[3])
	}
}

func TestUntrackedDiff(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")

	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := Repo{Root: dir}
	diff, err := repo.Diff(context.Background(), FileStatus{
		Path:     "notes.txt",
		Index:    '?',
		Worktree: '?',
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "+hello") {
		t.Fatalf("expected untracked file content in diff, got:\n%s", diff)
	}
}

func TestStageAllUnstageAllAndCommit(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "diffmate@example.com")
	runGit(t, dir, "config", "user.name", "Diffmate Test")

	if err := os.WriteFile(filepath.Join(dir, "one.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "two.txt"), []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := Repo{Root: dir}
	if err := repo.StageAll(context.Background()); err != nil {
		t.Fatal(err)
	}

	status := gitOutput(t, dir, "status", "--porcelain")
	if !strings.Contains(status, "A  one.txt") || !strings.Contains(status, "A  two.txt") {
		t.Fatalf("expected files to be staged, got:\n%s", status)
	}

	if err := repo.UnstageAll(context.Background()); err != nil {
		t.Fatal(err)
	}

	status = gitOutput(t, dir, "status", "--porcelain")
	if !strings.Contains(status, "?? one.txt") || !strings.Contains(status, "?? two.txt") {
		t.Fatalf("expected files to be unstaged, got:\n%s", status)
	}

	if err := repo.StageAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := repo.Commit(context.Background(), "initial commit"); err != nil {
		t.Fatal(err)
	}

	status = gitOutput(t, dir, "status", "--porcelain")
	if strings.TrimSpace(status) != "" {
		t.Fatalf("expected clean working tree after commit, got:\n%s", status)
	}
}

func TestInitCreatesRepositoryWithDescription(t *testing.T) {
	dir := t.TempDir()

	repo, err := Init(context.Background(), dir, "menu app")
	if err != nil {
		t.Fatal(err)
	}
	if repo.Root != dir {
		t.Fatalf("expected repo root %q, got %q", dir, repo.Root)
	}

	description, err := os.ReadFile(filepath.Join(dir, ".git", "description"))
	if err != nil {
		t.Fatal(err)
	}
	if string(description) != "menu app\n" {
		t.Fatalf("unexpected description: %q", description)
	}

	if _, err := OpenInDir(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

package suggest

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

const InstallCodexPlaceholder = "install codex cli to suggest a commit message for you"

func CommitMessage(ctx context.Context, repoRoot string) (string, error) {
	if _, err := exec.LookPath("codex"); err != nil {
		return InstallCodexPlaceholder, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	output, err := os.CreateTemp("", "diffmate-codex-*.txt")
	if err != nil {
		return "", err
	}
	outputPath := output.Name()
	output.Close()
	defer os.Remove(outputPath)

	prompt := `Inspect the Git changes in this repository and suggest exactly one commit message.

Rules:
- Use Conventional Commits.
- Prefer "type(scope): summary" when a clear scope exists.
- Use "type: summary" when no clear scope exists.
- Keep it under 72 characters.
- Output only the commit message, with no markdown and no explanation.
- Do not edit files.`

	cmd := exec.CommandContext(
		ctx,
		"codex",
		"exec",
		"--cd", repoRoot,
		"--sandbox", "read-only",
		"--color", "never",
		"--output-last-message", outputPath,
		"-",
	)
	cmd.Dir = repoRoot
	cmd.Stdin = strings.NewReader(prompt)

	if out, err := cmd.CombinedOutput(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "codex suggestion timed out; write commit message manually", nil
		}
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		return "codex could not suggest a commit message: " + firstLine(message), nil
	}

	message, err := os.ReadFile(outputPath)
	if err != nil {
		return "", err
	}

	suggestion := cleanCommitMessage(string(message))
	if suggestion == "" {
		return "codex returned an empty suggestion; write commit message manually", nil
	}

	return suggestion, nil
}

func cleanCommitMessage(message string) string {
	message = strings.TrimSpace(message)
	message = strings.Trim(message, "`")
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "```") && line != "txt" {
			return line
		}
	}
	return ""
}

func firstLine(message string) string {
	for _, line := range strings.Split(message, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "WARNING:") {
			return line
		}
	}
	return "unknown error"
}

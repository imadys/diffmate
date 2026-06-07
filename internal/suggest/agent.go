package suggest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const InstallAgentPlaceholder = "install the selected coding agent cli to suggest a commit message for you"

type agentRunner struct {
	command       string
	args          []string
	stdout        bool
	promptInStdin bool
}

func CommitMessage(ctx context.Context, repoRoot, agent, diff string) (string, error) {
	agent = strings.TrimSpace(strings.ToLower(agent))
	if agent == "" {
		agent = "codex"
	}

	runner, err := runnerForAgent(agent, repoRoot)
	if err != nil {
		return "", err
	}
	if _, err := exec.LookPath(runner.command); err != nil {
		return installPlaceholder(agent), nil
	}

	diff = strings.TrimSpace(diff)
	if diff == "" {
		return "stage files first, or edit the commit message manually", nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	prompt := commitPrompt(diff)
	if !runner.promptInStdin {
		prompt = "Diff:\n" + diff
	}
	cmd := exec.CommandContext(ctx, runner.command, runner.args...)
	cmd.Dir = repoRoot
	cmd.Stdin = strings.NewReader(prompt)

	out, err := runAgent(cmd, runner.stdout)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Sprintf("%s suggestion timed out; write commit message manually", agent), nil
		}
		message := strings.TrimSpace(out)
		if message == "" {
			message = err.Error()
		}
		return fmt.Sprintf("%s could not suggest a commit message: %s", agent, agentErrorLine(message)), nil
	}

	suggestion := cleanCommitMessage(out)
	if suggestion == "" {
		return fmt.Sprintf("%s returned an empty suggestion; write commit message manually", agent), nil
	}

	return suggestion, nil
}

func runnerForAgent(agent, repoRoot string) (agentRunner, error) {
	switch agent {
	case "codex":
		output, err := os.CreateTemp("", "diffmate-codex-*.txt")
		if err != nil {
			return agentRunner{}, err
		}
		outputPath := output.Name()
		_ = output.Close()
		return agentRunner{
			command: "codex",
			args: []string{
				"exec",
				"--cd", repoRoot,
				"--sandbox", "read-only",
				"--color", "never",
				"--model", "codex-mini-latest",
				"--output-last-message", outputPath,
				"-",
			},
			promptInStdin: true,
		}, nil
	case "claude":
		return agentRunner{
			command: "claude",
			args:    []string{"--bare", "--model", "haiku", "-p", commitInstructions()},
			stdout:  true,
		}, nil
	case "gemini":
		return agentRunner{
			command: "gemini",
			args:    []string{"-p", commitInstructions(), "--model", "gemini-2.5-flash", "--output-format", "json"},
			stdout:  true,
		}, nil
	case "antigravity":
		return agentRunner{}, fmt.Errorf("antigravity does not have a verified headless prompt mode yet")
	default:
		return agentRunner{}, fmt.Errorf("%s does not support commit suggestions yet", agent)
	}
}

func runAgent(cmd *exec.Cmd, stdout bool) (string, error) {
	if stdout {
		out, err := cmd.CombinedOutput()
		if cmdName(cmd) == "gemini" {
			return geminiResponse(string(out)), err
		}
		return string(out), err
	}

	outputPath := outputPathArg(cmd.Args)
	defer os.Remove(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}

	message, err := os.ReadFile(outputPath)
	if err != nil {
		return "", err
	}
	return string(message), nil
}

func commitPrompt(diff string) string {
	return commitInstructions() + `

Diff:
` + diff
}

func commitInstructions() string {
	return `Suggest exactly one commit message for this Git diff.

Rules:
- Use Conventional Commits.
- Prefer "type(scope): summary" when a clear scope exists.
- Use "type: summary" when no clear scope exists.
- Keep it under 72 characters.
- Output only the commit message, with no markdown and no explanation.`
}

func installPlaceholder(agent string) string {
	if agent == "" {
		return InstallAgentPlaceholder
	}
	return fmt.Sprintf("install %s cli to suggest a commit message for you", agent)
}

func outputPathArg(args []string) string {
	for i, arg := range args {
		if arg == "--output-last-message" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func cmdName(cmd *exec.Cmd) string {
	if len(cmd.Args) == 0 {
		return ""
	}
	return cmd.Args[0]
}

func geminiResponse(output string) string {
	var response struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal([]byte(output), &response); err == nil && strings.TrimSpace(response.Response) != "" {
		return response.Response
	}
	return output
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

func agentErrorLine(message string) string {
	for _, line := range strings.Split(message, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "WARNING:") {
			continue
		}
		if strings.HasPrefix(line, "OpenAI Codex ") || strings.HasPrefix(line, "Codex ") {
			continue
		}
		return line
	}
	return "unknown error"
}

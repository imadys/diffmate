package tui

import (
	"os"
	"os/exec"
)

type toolOption struct {
	Label   string
	Command string
}

var editorOptions = []toolOption{
	{Label: "VS Code", Command: "code"},
	{Label: "Zed", Command: "zed"},
	{Label: "Cursor", Command: "cursor"},
	{Label: "Neovim", Command: "nvim"},
}

var agentOptions = []toolOption{
	{Label: "Codex", Command: "codex"},
	{Label: "Claude", Command: "claude"},
	{Label: "Antigravity", Command: "antigravity"},
	{Label: "Gemini", Command: "gemini"},
}

func editorCommand(dir, path string) *exec.Cmd {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, path)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func projectCommand(dir, command string) *exec.Cmd {
	cmd := exec.Command(command, ".")
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func agentCommand(dir, command string) *exec.Cmd {
	cmd := exec.Command(command)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

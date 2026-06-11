package tui

import (
	"bytes"
	"errors"
	"os/exec"
	"runtime"
)

func copyToClipboard(value string) error {
	if value == "" {
		return errors.New("nothing to copy")
	}
	for _, command := range clipboardCommands() {
		cmd := exec.Command(command.name, command.args...)
		cmd.Stdin = bytes.NewBufferString(value)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return errors.New("no clipboard command found")
}

type clipboardCommand struct {
	name string
	args []string
}

func clipboardCommands() []clipboardCommand {
	switch runtime.GOOS {
	case "darwin":
		return []clipboardCommand{{name: "pbcopy"}}
	case "windows":
		return []clipboardCommand{{name: "clip"}}
	default:
		return []clipboardCommand{
			{name: "wl-copy"},
			{name: "xclip", args: []string{"-selection", "clipboard"}},
			{name: "xsel", args: []string{"--clipboard", "--input"}},
		}
	}
}

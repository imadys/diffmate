package app

import (
	"context"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/imadys/diffmate/internal/git"
	"github.com/imadys/diffmate/internal/tui"
)

func Run(args []string) error {
	if len(args) == 0 {
		return runReview()
	}

	switch args[0] {
	case "review":
		return runReview()
	case "help", "-h", "--help":
		fmt.Println(helpText)
		return nil
	case "version", "-v", "--version":
		fmt.Println("diffmate dev")
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], helpText)
	}
}

func runReview() error {
	ctx := context.Background()
	repo, err := git.Open(ctx)
	if err != nil {
		return err
	}

	program := tea.NewProgram(tui.New(repo), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		if errors.Is(err, tea.ErrProgramKilled) {
			return nil
		}
		return err
	}

	return nil
}

const helpText = `diffmate - review your working tree before committing

Usage:
  diffmate review
  diffmate help

Keybindings:
  up/down, k/j   move through files
  pgup/pgdn      scroll diff
  s              stage selected file
  u              unstage selected file
  S              stage all files
  U              unstage all files
  c              write commit message
  ctrl+s         create commit from message box
  e              open selected file in $EDITOR
  r              refresh
  q              quit`

package app

import (
	"context"
	"errors"
	"fmt"
	"os"

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
		if !errors.Is(err, git.ErrNotRepository) {
			return err
		}
		initialized, err := runSetup()
		if err != nil || !initialized {
			return err
		}
		repo, err = git.Open(ctx)
		if err != nil {
			return err
		}
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

func runSetup() (bool, error) {
	dir, err := os.Getwd()
	if err != nil {
		return false, err
	}

	program := tea.NewProgram(tui.NewSetup(dir), tea.WithAltScreen())
	final, err := program.Run()
	if err != nil {
		if errors.Is(err, tea.ErrProgramKilled) {
			return false, nil
		}
		return false, err
	}

	result, ok := final.(interface{ Initialized() bool })
	return ok && result.Initialized(), nil
}

const helpText = `diffmate - review your working tree before committing

Usage:
  diffmate review
  diffmate help

Keybindings:
  up/down, k/j   move through files
  1-4            focus sidebar cards
  5              focus diff
  tab            cycle cards and diff
  ,              open config
  t              open config sections
  [, ]           scroll diff by line
  pgup/pgdn      scroll diff by page
  space          scroll diff down by page
  g, G           jump diff top/bottom
  s              stage selected file
  u              unstage selected file
  S              stage all files
  U              unstage all files
  c              write commit message
  ctrl+g         suggest commit message with preferred agent
  ctrl+s         create commit from message box
  ctrl+d         clear commit message and modal errors
  p              push current branch
  o              open preferred editor
  a              open preferred agent
  e              open selected file in $EDITOR
  r              refresh
  ?              show full keymap
  q              quit`

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommandKeyMapsArabicKeyboardAliases(t *testing.T) {
	tests := map[string]string{
		"ؤ": "c",
		"ح": "p",
		"ر": "v",
		"ش": "a",
		"س": "s",
		"ظ": "/",
		"؟": "?",
	}

	for input, want := range tests {
		got := commandKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(input)})
		if got != want {
			t.Fatalf("expected %q to map to %q, got %q", input, want, got)
		}
	}
}

func TestCommandKeyKeepsTextWithoutAlias(t *testing.T) {
	got := commandKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if got != "x" {
		t.Fatalf("expected x to stay x, got %q", got)
	}
}

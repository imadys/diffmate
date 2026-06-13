package tui

import "testing"

func TestVisualTextReordersRTLRunForTerminalDisplay(t *testing.T) {
	got := visualText("قريب")
	if got != "بيرق" {
		t.Fatalf("expected Arabic run to be reversed for terminal display, got %q", got)
	}
}

func TestVisualTextLeavesLTRTextAlone(t *testing.T) {
	got := visualText("Review your working tree")
	if got != "Review your working tree" {
		t.Fatalf("expected LTR text to stay unchanged, got %q", got)
	}
}

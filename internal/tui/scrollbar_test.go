package tui

import (
	"strings"
	"testing"
)

func TestRenderScrollbarOnlyShowsTrackWhenScrollable(t *testing.T) {
	scrollbar := renderScrollbar(20, 5, 10)
	lines := strings.Split(scrollbar, "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}
	if strings.TrimSpace(scrollbar) == "" {
		t.Fatal("expected visible scrollbar")
	}
}

func TestRenderScrollbarBlankWhenContentFits(t *testing.T) {
	scrollbar := renderScrollbar(3, 5, 0)
	if strings.TrimSpace(scrollbar) != "" {
		t.Fatalf("expected blank scrollbar, got %q", scrollbar)
	}
}

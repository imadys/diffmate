package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"path/filepath"
	"strings"
	"time"
)

func (m model) renderHeader(width int) string {
	count := fmt.Sprintf("%d files", len(m.files))
	if len(m.files) == 1 {
		count = "1 file"
	}

	left := subtleStyle.Render("review before commit")
	right := subtleStyle.Render(m.repoName() + "  " + count)
	if lipgloss.Width(left)+lipgloss.Width(right)+1 > width {
		return headerStyle.Width(width).Render(truncate(left, width))
	}
	spacer := strings.Repeat(" ", max(1, width-lipgloss.Width(left)-lipgloss.Width(right)))
	return headerStyle.Width(width).Render(left + spacer + right)
}
func (m model) repoName() string {
	name := filepath.Base(m.repo.Root)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "repository"
	}
	return name
}
func (m model) keySegments() []keySegment {
	if m.mode == commitMode {
		return []keySegment{
			{"type", "message"},
			{"enter", "newline"},
			{"ctrl+g", m.settings.Agent + " suggest"},
			{"ctrl+s", "commit"},
			{"esc", "cancel"},
		}
	}

	if m.showHelp {
		return []keySegment{
			{"?", "hide help"},
			{"esc", "quit"},
		}
	}

	return []keySegment{
		{"1-4", "cards"},
		{"5", "diff"},
		{"tab", "next"},
		{",", "config"},
		{"j/k", "files"},
		{"[/]", "diff line"},
		{"space", "diff page"},
		{"g/G", "top/bottom"},
		{"s/u", "file stage"},
		{"S/U", "all stage"},
		{"c", "commit"},
		{"p", "push"},
		{"o", "editor"},
		{"a", "agent"},
		{"?", "all keys"},
	}
}

type keySegment struct {
	key   string
	label string
}

func (m model) renderKeySegments(width int) string {
	logo := miniLogo()
	status := m.footerStatus()
	content := logo + " " + status + "  " + keyStyle.Render("c") + " commit  " + keyStyle.Render("p") + " push  " + keyStyle.Render("S") + " stage all  " + keyStyle.Render("U") + " unstage all  " + keyStyle.Render("o") + " editor  " + keyStyle.Render("a") + " agent  " + keyStyle.Render(",") + " config  " + keyStyle.Render("?") + " keymap  " + keyStyle.Render("q") + " quit"
	return keyBarStyle.Width(width).Render(truncate(content, max(1, width-2)))
}
func suggestTick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return suggestTickMsg{}
	})
}
func firstLine(message string) string {
	for _, line := range strings.Split(message, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "unknown error"
}
func (m model) renderFooter(width int) string {
	return m.renderKeySegments(width)
}

func (m model) footerStatus() string {
	status := m.status
	if m.loading {
		status = "loading"
	}
	if m.err != nil {
		status = errorStyle.Render(m.err.Error())
	}
	if status == "" {
		status = "ready"
	}
	if m.mode == commitMode {
		status = "commit mode"
	} else if m.showHelp {
		status = "help"
	}
	return status
}
func miniLogo() string {
	return titleStyle.Render("diffmate")
}

package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"path/filepath"
	"strings"
	"time"
)

func (m model) renderHeader() string {
	count := fmt.Sprintf("%d files", len(m.files))
	if len(m.files) == 1 {
		count = "1 file"
	}

	left := subtleStyle.Render("review before commit")
	right := subtleStyle.Render(m.repoName() + "  " + count)
	if lipgloss.Width(left)+lipgloss.Width(right)+1 > m.width {
		return headerStyle.Width(m.width).Render(truncate(left, m.width))
	}
	spacer := strings.Repeat(" ", max(1, m.width-lipgloss.Width(left)-lipgloss.Width(right)))
	return headerStyle.Width(m.width).Render(left + spacer + right)
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

func (m model) renderKeySegments(segments []keySegment) string {
	logo := miniLogo()
	content := keyStyle.Render("c") + " commit  " + keyStyle.Render("p") + " push  " + keyStyle.Render("S") + " stage all  " + keyStyle.Render("U") + " unstage all  " + keyStyle.Render("o") + " editor  " + keyStyle.Render("a") + " agent  " + keyStyle.Render(",") + " config  " + keyStyle.Render("?") + " keymap"
	available := max(1, m.width-lipgloss.Width(logo)-2)
	return keyBarStyle.Width(m.width).Render(logo + " " + truncate(content, available))
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
func (m model) renderFooter() string {
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

	right := "q quit"
	if m.mode == commitMode {
		right = "commit mode"
	} else if m.showHelp {
		right = "help"
	}

	leftWidth := max(0, m.width-len(right)-4)
	left := truncate(status, leftWidth)
	spacer := strings.Repeat(" ", max(1, m.width-lipgloss.Width(left)-len(right)-2))
	statusLine := statusStyle.Width(m.width).Render(left + spacer + right)
	return statusLine + "\n" + m.renderKeySegments(m.keySegments())
}
func miniLogo() string {
	return titleStyle.Render("diffmate")
}

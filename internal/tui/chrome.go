package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"path/filepath"
	"strings"
	"time"
)

func (m model) renderHeader(width int) string {
	count := fmt.Sprintf("%d files", len(m.files))
	if len(m.files) == 1 {
		count = "1 file"
	}

	content := titleStyle.Render(m.repoName()) + subtleStyle.Render("  "+count)
	return headerStyle.Render(truncate(content, width))
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
			{"ctrl+d", "clear"},
			{"esc", "cancel"},
		}
	}

	if m.mode == confirmMode {
		return []keySegment{
			{"y/enter", "confirm"},
			{"esc", "cancel"},
		}
	}

	if m.mode == branchInputMode {
		return []keySegment{
			{"type", "branch name"},
			{"enter", "create"},
			{"esc", "cancel"},
		}
	}

	if m.mode == mergePickerMode {
		return []keySegment{
			{"j/k", "branch"},
			{"enter", "merge"},
			{"esc", "cancel"},
		}
	}

	if m.mode == conflictMode {
		if len(m.conflicts) == 0 {
			return []keySegment{
				{"c", "continue"},
				{"a", "abort"},
				{"r", "refresh"},
				{"?", "keymap"},
			}
		}
		return []keySegment{
			{"j/k", "file"},
			{"[/]", "scroll"},
			{"o", "ours"},
			{"t", "theirs"},
			{"s", "stage"},
			{"e", "fix in editor"},
			{"a", "abort"},
			{"r", "refresh"},
			{"?", "keymap"},
		}
	}

	if m.showHelp {
		return []keySegment{
			{"?", "hide help"},
			{"esc", "quit"},
		}
	}

	if m.focus == diffFocus {
		return []keySegment{
			{"[/]", "line"},
			{"space", "page"},
			{"g/G", "top/bottom"},
			{"left", "cards"},
			{"?", "keymap"},
		}
	}

	switch m.tab {
	case changesTab:
		return []keySegment{
			{"space", "stage/unstage"},
			{"S", "stage all"},
			{"U", "unstage all"},
			{"s", "stash"},
			{"D", "reset"},
			{"c", "commit"},
			{"?", "keymap"},
		}
	case branchesTab:
		return []keySegment{
			{"space", "checkout"},
			{"m", "merge"},
			{"u", "upstream"},
			{"p", "push"},
			{"n", "new branch"},
			{"d", "delete"},
			{"D", "delete remote"},
			{"ctrl+d", "delete both"},
			{"?", "keymap"},
		}
	case commitsTab:
		return []keySegment{
			{"space", "view diff"},
			{"?", "keymap"},
		}
	case stashTab:
		return []keySegment{
			{"space", "view stash"},
			{"?", "keymap"},
		}
	default:
		return []keySegment{
			{"?", "keymap"},
		}
	}
}

type keySegment struct {
	key   string
	label string
}

func (m model) renderKeySegments(width int) string {
	logo := miniLogo()
	status := m.footerStatus()
	parts := []string{logo + " " + status}
	for _, segment := range m.keySegments() {
		parts = append(parts, keyStyle.Render(segment.key)+" "+segment.label)
	}
	if m.mode != conflictMode {
		parts = append(parts, keyStyle.Render("~")+" console")
		if m.consoleVisible {
			parts = append(parts, keyStyle.Render("ctrl+l")+" clear")
		}
	}
	parts = append(parts, keyStyle.Render("q")+" quit")
	content := strings.Join(parts, " | ")
	return keyBarStyle.Render(truncate(content, width))
}
func suggestTick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return suggestTickMsg{}
	})
}
func cursorTick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return cursorTickMsg{}
	})
}
func autoRefreshTick() tea.Cmd {
	return tea.Tick(3*time.Minute, func(time.Time) tea.Msg {
		return autoRefreshMsg{}
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
	if status == "" {
		status = "ready"
	}
	if m.mode == commitMode {
		status = "commit mode"
	} else if m.mode == confirmMode {
		status = "confirm"
	} else if m.mode == branchInputMode {
		status = "new branch"
	} else if m.mode == mergePickerMode {
		status = "merge"
	} else if m.mode == conflictMode {
		status = fmt.Sprintf("conflict %d", len(m.conflicts))
	} else if m.showHelp {
		status = "help"
	}
	return status
}
func miniLogo() string {
	return titleStyle.Render("diffmate")
}

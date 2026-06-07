package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

func (m model) renderHelpBox() string {
	lines := []string{
		titleStyle.Render("Keymap"),
		"",
		"1-4                focus sidebar cards",
		"5                  focus diff",
		"tab                cycle cards and diff",
		",                  open config",
		"t                  open config sections",
		"right              focus diff for selected row",
		"left               return to sidebar from diff",
		"h/l                switch cards or scroll diff",
		"k/j, up/down       move between files",
		"[, ], left/right   scroll diff by line",
		"space, f/b         scroll diff by page",
		"ctrl+d/ctrl+u      scroll diff by page",
		"g/G                jump diff top/bottom",
		"s/u                stage/unstage selected file",
		"S/U                stage/unstage all changes",
		"c                  open commit message box",
		"p                  push current branch",
		"o                  open preferred editor",
		"a                  open preferred agent",
		"ctrl+g             suggest commit message with codex",
		"e, enter           open selected file in editor",
		"r                  refresh",
		"q, esc             quit",
		"?                  show/hide this help",
	}

	width := clamp(58, 38, max(38, m.width-8))
	height := min(len(lines)+2, max(8, m.height-4))
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(1, 2).
		Render(fitLines(lines, height-2))
}
func (m model) renderWelcomeBox() string {
	lines := []string{
		miniLogo(),
		"",
		titleStyle.Render("Review your working tree before committing."),
		mutedStyle.Render("Press any key to start. Press ? anytime for this keymap."),
		"",
		"1-4                focus sidebar cards",
		"5                  focus diff",
		"tab                cycle cards and diff",
		",                  open config",
		"j/k, up/down       move in focused card or diff",
		"right              focus diff for selected row",
		"left               return to sidebar from diff",
		"h/l                switch cards or scroll diff",
		"s/u                stage/unstage selected file",
		"S/U                stage/unstage all changes",
		"c                  commit message box",
		"ctrl+g             suggest commit with Codex",
		"p                  push current branch",
		"o / a              open preferred editor / agent",
		"q                  quit",
	}

	width := clamp(66, 44, max(44, m.width-8))
	height := min(len(lines)+2, max(10, m.height-4))
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(1, 2).
		Render(fitLines(lines, height-2))
}
func (m model) renderTabsBox() string {
	lines := []string{
		titleStyle.Render("Config"),
		"",
		m.renderConfigSections(),
		"",
		mutedStyle.Render("left/right section  space select  esc close"),
		"",
	}

	lines = append(lines, m.renderConfigItems()...)

	return lipgloss.NewStyle().
		Width(44).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))
}
func (m model) renderConfigSections() string {
	sections := []string{"Sections", "Editor", "Agent"}
	for i, section := range sections {
		if i == m.configSection {
			sections[i] = selectedStyle.Render(section)
		} else {
			sections[i] = subtleStyle.Render(section)
		}
	}
	return strings.Join(sections, "  ")
}
func (m model) renderConfigItems() []string {
	switch m.configSection {
	case 0:
		lines := make([]string, 0, 4)
		for i, tab := range []sidebarTab{changesTab, branchesTab, commitsTab, stashTab} {
			check := "[ ]"
			if m.tabsEnabled[tab] {
				check = "[x]"
			}
			line := check + " " + sectionTitle(tab)
			if i == m.tabMenuSelected {
				line = selectedStyle.Width(34).Render(line)
			}
			lines = append(lines, line)
		}
		return lines
	case 1:
		return renderToolOptions(editorOptions, m.tabMenuSelected, m.settings.Editor)
	case 2:
		return renderToolOptions(agentOptions, m.tabMenuSelected, m.settings.Agent)
	default:
		return nil
	}
}
func renderToolOptions(options []toolOption, selected int, active string) []string {
	lines := make([]string, 0, len(options))
	for i, option := range options {
		check := "  "
		if option.Command == active {
			check = "* "
		}
		line := check + option.Label + " (" + option.Command + ")"
		if i == selected {
			line = selectedStyle.Width(34).Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}
func (m model) renderCommitBox() string {
	width := clamp(72, 40, max(40, m.width-8))
	height := clamp(12, 8, max(8, m.height-8))
	contentHeight := height - 4
	if m.commitError != "" {
		contentHeight--
	}

	messageLines := strings.Split(m.commitMessage, "\n")
	if m.commitMessage == "" {
		messageLines = []string{mutedStyle.Render("Commit message")}
	}
	messageBody := fitLines(messageLines, contentHeight)

	help := mutedStyle.Render("ctrl+s commit  esc cancel")
	if m.suggesting {
		help = mutedStyle.Render(fmt.Sprintf("asking %s for a commit message... %ds", m.settings.Agent, m.suggestElapsed))
	} else {
		help = mutedStyle.Render("ctrl+g suggest  ctrl+s commit  esc cancel")
	}
	if m.commitError != "" {
		help = errorStyle.Render(m.commitError) + "\n" + help
	}
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(1, 2).
		Render(titleStyle.Render("Commit") + "\n" + messageBody + "\n" + help)
}
func sectionTitle(tab sidebarTab) string {
	switch tab {
	case changesTab:
		return "Changes"
	case branchesTab:
		return "Local branches"
	case commitsTab:
		return "Commits"
	case stashTab:
		return "Stash"
	default:
		return "Tab"
	}
}

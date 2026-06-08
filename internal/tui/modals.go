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
		"/                  search focused sidebar card",
		"[, ], left/right   scroll diff by line",
		"space, f/b         scroll diff by page",
		"ctrl+d/ctrl+u      scroll diff by page",
		"g/G                jump diff top/bottom",
		"space              primary action in focused card",
		"S/U                stage/unstage all changes in changes",
		"s                  stash changes in changes",
		"D                  reset selected change with confirm",
		"n                  new branch in branches",
		"d                  delete branch with confirm",
		"D                  delete remote branch with confirm",
		"ctrl+d             delete local and remote branch",
		"m                  merge selected branch into current branch",
		"u                  update current branch from upstream",
		"p                  push current branch",
		"",
		"Conflict mode",
		"j/k                move conflicted file",
		"o/t                accept ours/theirs",
		"s                  stage selected file as resolved",
		"e                  open selected conflict in editor to fix manually",
		"a                  abort merge with confirm",
		"c                  continue merge after all conflicts are staged",
		"",
		"Common actions",
		"c                  open commit message box",
		"p                  push current branch",
		"o                  open preferred editor",
		"a                  open preferred agent",
		"ctrl+g             suggest commit message with codex",
		"e, enter           open selected file in editor",
		"/                  search focused sidebar card",
		"r                  refresh",
		"~                  show/hide console",
		"ctrl+l             clear console log",
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
		"space              primary action in focused card",
		"S/U                stage/unstage all changes",
		"s                  stash changes",
		"D                  reset selected change",
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
		messageLines = []string{inputCursor(m.cursorVisible) + mutedStyle.Render("Commit message")}
	} else {
		messageLines[len(messageLines)-1] += inputCursor(m.cursorVisible)
	}
	messageBody := fitLines(messageLines, contentHeight)

	help := mutedStyle.Render("ctrl+s commit  esc cancel")
	if m.suggesting {
		help = mutedStyle.Render(fmt.Sprintf("asking %s for a commit message... %ds", m.settings.Agent, m.suggestElapsed))
	} else {
		help = mutedStyle.Render("ctrl+g suggest  ctrl+s commit  ctrl+d clear  esc cancel")
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

func (m model) renderConfirmBox() string {
	lines := []string{
		titleStyle.Render(m.confirmTitle),
		"",
		m.confirmMessage,
		"",
		mutedStyle.Render("y/enter confirm  esc cancel"),
	}
	return lipgloss.NewStyle().
		Width(clamp(72, 42, max(42, m.width-8))).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("203")).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))
}

func (m model) renderBranchInputBox() string {
	width := clamp(64, 38, max(38, m.width-8))
	input := m.branchNameInput
	if input == "" {
		input = inputCursor(m.cursorVisible) + mutedStyle.Render("branch name")
	} else {
		input += inputCursor(m.cursorVisible)
	}
	help := mutedStyle.Render("enter create  esc cancel")
	if m.inputError != "" {
		help = errorStyle.Render(m.inputError) + "\n" + help
	}
	lines := []string{
		titleStyle.Render("New branch"),
		"",
		input,
		"",
		help,
	}
	return lipgloss.NewStyle().
		Width(width).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))
}

func (m model) renderMergePickerBox() string {
	width := clamp(64, 42, max(42, m.width-8))
	height := clamp(16, 8, max(8, m.height-8))
	contentHeight := max(1, height-5)
	offset := keepIndexVisible(m.mergeSelected, len(m.branches), contentHeight)
	end := min(len(m.branches), offset+contentHeight)
	lines := []string{
		titleStyle.Render("Merge branch"),
		mutedStyle.Render("into " + m.currentBranchName()),
		"",
	}

	for i := offset; i < end; i++ {
		branch := m.branches[i]
		label := branch.Name
		if branch.Current {
			label = label + " " + mutedStyle.Render("(current)")
		}
		if i == m.mergeSelected {
			label = selectedLineStyle(true, width-4).Render(label)
		}
		lines = append(lines, label)
	}

	lines = append(lines, "", mutedStyle.Render("enter merge  esc cancel"))
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(1, 2).
		Render(fitLines(lines, height-2))
}

func inputCursor(visible bool) string {
	if !visible {
		return " "
	}
	return selectedStyle.Render(" ")
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

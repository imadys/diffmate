package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading diffmate..."
	}

	sideInset := 1
	topGap := 0
	bodyGap := 1
	bodyWidth := max(1, m.width-sideInset*2)

	header := indentBlock(m.renderHeader(bodyWidth), sideInset)
	footer := indentBlock(m.renderFooter(bodyWidth), sideInset)
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	bodyHeight := max(1, m.height-headerHeight-footerHeight-topGap)
	sidebarWidth := clamp(34, 26, bodyWidth/2)
	diffWidth := max(1, bodyWidth-sidebarWidth-bodyGap)
	diffHeight, consoleHeight := m.rightPaneHeights(bodyHeight)

	files := m.renderSidebar(sidebarWidth, bodyHeight)
	rightPane := m.renderDiff(diffWidth, diffHeight)
	if consoleHeight > 0 {
		rightPane += "\n" + m.renderConsole(diffWidth, consoleHeight)
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, files, strings.Repeat(" ", bodyGap), rightPane)

	if m.mode == commitMode {
		body = overlayCommitBox(body, m.renderCommitBox())
	}
	if m.mode == confirmMode {
		body = overlayCommitBox(body, m.renderConfirmBox())
	}
	if m.mode == branchInputMode {
		body = overlayCommitBox(body, m.renderBranchInputBox())
	}
	if m.mode == mergePickerMode {
		body = overlayCommitBox(body, m.renderMergePickerBox())
	}
	if m.showHelp {
		body = overlayCommitBox(body, m.renderHelpBox())
	}
	if m.showTabs {
		body = overlayCommitBox(body, m.renderTabsBox())
	}
	if m.showWelcome {
		body = overlayCommitBox(body, m.renderWelcomeBox())
	}

	body = indentBlock(body, sideInset)
	gap := ""
	if topGap > 0 {
		gap = strings.Repeat("\n", topGap)
	}
	return appStyle.Render(header + "\n" + gap + body + "\n" + footer)
}

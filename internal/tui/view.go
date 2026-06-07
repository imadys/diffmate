package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading diffmate..."
	}

	header := m.renderHeader()
	headerHeight := lipgloss.Height(header)
	footer := m.renderFooter()
	footerHeight := lipgloss.Height(footer)
	bodyMarginLeft := 1
	bodyMarginTop := 1
	bodyGap := 1
	bodyHeight := max(1, m.height-headerHeight-footerHeight-bodyMarginTop-2)
	bodyWidth := max(1, m.width-bodyMarginLeft)
	sidebarWidth := clamp(34, 26, bodyWidth/2)
	diffWidth := max(1, bodyWidth-sidebarWidth-bodyGap)

	files := m.renderSidebar(sidebarWidth, bodyHeight)
	diff := m.renderDiff(diffWidth, bodyHeight)
	body := lipgloss.JoinHorizontal(lipgloss.Top, files, strings.Repeat(" ", bodyGap), diff)

	if m.mode == commitMode {
		body = overlayCommitBox(body, m.renderCommitBox())
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

	body = strings.Repeat("\n", bodyMarginTop) + indentBlock(body, bodyMarginLeft)
	return appStyle.Render(header + "\n" + body + "\n" + footer)
}

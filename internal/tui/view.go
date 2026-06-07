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
	topGap := 1
	bodyGap := 1
	bodyWidth := max(1, m.width-sideInset*2)

	header := indentBlock(m.renderHeader(bodyWidth), sideInset)
	footer := indentBlock(m.renderFooter(bodyWidth), sideInset)
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	bodyHeight := max(1, m.height-headerHeight-footerHeight-topGap)
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

	body = indentBlock(body, sideInset)
	return appStyle.Render(header + "\n" + strings.Repeat("\n", topGap) + body + "\n" + footer)
}

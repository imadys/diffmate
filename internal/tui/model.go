package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/imadys/diffmate/internal/git"
)

type screenMode int

const (
	reviewMode screenMode = iota
	commitMode
)

type model struct {
	repo          git.Repo
	files         []git.FileStatus
	selected      int
	diff          string
	diffOffset    int
	fileOffset    int
	width         int
	height        int
	err           error
	status        string
	loading       bool
	mode          screenMode
	showHelp      bool
	commitMessage string
}

type refreshMsg struct {
	files []git.FileStatus
	diff  string
	err   error
}

type actionMsg struct {
	status string
	err    error
}

var (
	appStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	subtleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("60")).Bold(true)
	panelStyle    = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238"))
	headerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("235")).Bold(true)
	addStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	delStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	hunkStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)
	codeMuted     = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
)

func New(repo git.Repo) model {
	return model{repo: repo, loading: true}
}

func (m model) Init() tea.Cmd {
	return m.refresh()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case refreshMsg:
		m.loading = false
		m.err = msg.err
		if msg.err != nil {
			return m, nil
		}
		m.files = msg.files
		if m.selected >= len(m.files) {
			m.selected = len(m.files) - 1
		}
		if m.selected < 0 {
			m.selected = 0
		}
		m.diff = msg.diff
		m.diffOffset = 0
		if len(m.files) == 0 {
			m.status = "working tree clean"
		}
		return m, nil
	case actionMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.status = msg.status
			if m.mode == commitMode {
				m.mode = reviewMode
				m.commitMessage = ""
			}
		}
		return m, m.refresh()
	case tea.KeyMsg:
		if m.showHelp && msg.String() != "?" {
			m.showHelp = false
			return m, nil
		}
		if m.mode == commitMode {
			return m.updateCommitMode(msg)
		}
		return m.updateReviewMode(msg)
	}

	return m, nil
}

func (m model) updateReviewMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "r":
		m.loading = true
		m.status = "refreshing"
		return m, m.refresh()
	case "up", "k":
		if m.selected > 0 {
			m.selected--
			m.keepSelectedFileVisible()
			m.diffOffset = 0
			return m, m.loadDiff()
		}
	case "down", "j":
		if m.selected < len(m.files)-1 {
			m.selected++
			m.keepSelectedFileVisible()
			m.diffOffset = 0
			return m, m.loadDiff()
		}
	case "[", "left":
		m.scrollDiff(-1)
	case "]", "right":
		m.scrollDiff(1)
	case "pgup", "b", "ctrl+u":
		m.scrollDiff(-m.diffHeight())
	case "pgdown", "f", " ", "ctrl+d":
		m.scrollDiff(m.diffHeight())
	case "g", "home":
		m.diffOffset = 0
	case "G", "end":
		m.diffOffset = max(0, len(m.diffLines())-m.diffHeight())
	case "s":
		return m, m.stage()
	case "u":
		return m, m.unstage()
	case "S":
		return m, m.stageAll()
	case "U":
		return m, m.unstageAll()
	case "c":
		m.mode = commitMode
		m.err = nil
		m.status = "write commit message"
		return m, nil
	case "e", "enter":
		return m, m.openEditor()
	}

	return m, nil
}

func (m model) updateCommitMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = reviewMode
		m.err = nil
		m.status = "commit cancelled"
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+s":
		return m, m.commit()
	case "enter":
		m.commitMessage += "\n"
	case "backspace":
		if len(m.commitMessage) > 0 {
			runes := []rune(m.commitMessage)
			m.commitMessage = string(runes[:len(runes)-1])
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.commitMessage += msg.String()
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading diffmate..."
	}

	bodyHeight := max(5, m.height-3)
	sidebarWidth := clamp(34, 26, m.width/2)
	diffWidth := max(24, m.width-sidebarWidth)

	header := m.renderHeader()
	files := m.renderFiles(sidebarWidth, bodyHeight)
	diff := m.renderDiff(diffWidth, bodyHeight)
	footer := m.renderFooter()
	body := lipgloss.JoinHorizontal(lipgloss.Top, files, diff)

	if m.mode == commitMode {
		body = overlayCommitBox(body, m.renderCommitBox())
	}
	if m.showHelp {
		body = overlayCommitBox(body, m.renderHelpBox())
	}

	return appStyle.Render(header + "\n" + body + "\n" + footer)
}

func (m model) refresh() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		files, err := m.repo.Status(ctx)
		if err != nil {
			return refreshMsg{err: err}
		}

		diff := ""
		if len(files) > 0 {
			diff, err = m.repo.Diff(ctx, files[min(m.selected, len(files)-1)])
			if err != nil {
				return refreshMsg{files: files, err: err}
			}
		}

		return refreshMsg{files: files, diff: diff}
	}
}

func (m model) loadDiff() tea.Cmd {
	return func() tea.Msg {
		if len(m.files) == 0 {
			return refreshMsg{files: m.files}
		}
		diff, err := m.repo.Diff(context.Background(), m.files[m.selected])
		return refreshMsg{files: m.files, diff: diff, err: err}
	}
}

func (m model) stage() tea.Cmd {
	return m.withSelected("staged", func(ctx context.Context, file git.FileStatus) error {
		return m.repo.Stage(ctx, file)
	})
}

func (m model) unstage() tea.Cmd {
	return m.withSelected("unstaged", func(ctx context.Context, file git.FileStatus) error {
		return m.repo.Unstage(ctx, file)
	})
}

func (m model) stageAll() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.StageAll(context.Background())
		return actionMsg{status: "staged all changes", err: err}
	}
}

func (m model) unstageAll() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.UnstageAll(context.Background())
		return actionMsg{status: "unstaged all changes", err: err}
	}
}

func (m model) commit() tea.Cmd {
	message := strings.TrimSpace(m.commitMessage)
	return func() tea.Msg {
		err := m.repo.Commit(context.Background(), message)
		return actionMsg{status: "commit created", err: err}
	}
}

func (m model) openEditor() tea.Cmd {
	if len(m.files) == 0 {
		return nil
	}
	file := m.files[m.selected]
	return tea.ExecProcess(editorCommand(m.repo.Root, file.Path), func(err error) tea.Msg {
		return actionMsg{status: "editor closed", err: err}
	})
}

func (m model) withSelected(status string, fn func(context.Context, git.FileStatus) error) tea.Cmd {
	if len(m.files) == 0 {
		return nil
	}
	file := m.files[m.selected]
	return func() tea.Msg {
		err := fn(context.Background(), file)
		return actionMsg{status: fmt.Sprintf("%s %s", status, file.Path), err: err}
	}
}

func (m model) renderHeader() string {
	count := fmt.Sprintf("%d files", len(m.files))
	if len(m.files) == 1 {
		count = "1 file"
	}

	left := titleStyle.Render("diffmate") + subtleStyle.Render("  review before commit")
	right := subtleStyle.Render(count)
	spacer := strings.Repeat(" ", max(1, m.width-lipgloss.Width(left)-lipgloss.Width(right)))
	return headerStyle.Width(m.width).Padding(0, 1).Render(left + spacer + right)
}

func (m model) renderFiles(width, height int) string {
	contentHeight := max(1, height-2)
	lines := []string{subtleStyle.Render("Changes")}
	if len(m.files) == 0 {
		lines = append(lines, mutedStyle.Render("No changes"))
	} else {
		visibleFiles := m.visibleFiles(contentHeight - 1)
		for visibleIndex, file := range visibleFiles {
			i := m.fileOffset + visibleIndex
			line := m.renderFileLine(file, width-4)
			if i == m.selected {
				line = selectedStyle.Width(width - 4).Render(line)
			}
			lines = append(lines, line)
		}
	}

	return panelStyle.
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(fitLines(lines, contentHeight))
}

func (m model) renderFileLine(file git.FileStatus, width int) string {
	status := strings.TrimSpace(string([]byte{file.Index, file.Worktree}))
	if status == "" {
		status = "??"
	}

	statusStyle := codeMuted
	if strings.Contains(status, "A") || status == "??" {
		statusStyle = addStyle
	}
	if strings.Contains(status, "D") {
		statusStyle = delStyle
	}

	label := statusStyle.Bold(true).Render(fmt.Sprintf("%-2s", status))
	path := truncate(file.Path, max(1, width-lipgloss.Width(label)-1))
	return label + " " + path
}

func (m model) renderDiff(width, height int) string {
	contentHeight := max(1, height-2)
	lines := m.diffLines()
	if strings.TrimSpace(m.diff) == "" {
		lines = []string{mutedStyle.Render("Select a changed file to view its diff.")}
	}

	m.clampDiffOffset()
	end := min(len(lines), m.diffOffset+contentHeight-1)
	if m.diffOffset > len(lines) {
		m.diffOffset = 0
	}
	visible := lines[m.diffOffset:end]

	header := subtleStyle.Render("Diff")
	if len(m.files) > 0 {
		scroll := fmt.Sprintf("  %d/%d", min(m.diffOffset+1, len(lines)), max(1, len(lines)))
		header = subtleStyle.Render(truncate(m.files[m.selected].Path, max(1, width-4-len(scroll)))) + mutedStyle.Render(scroll)
	}

	out := append([]string{header}, visible...)
	for i := 1; i < len(out); i++ {
		out[i] = colorDiffLine(truncate(out[i], width-4))
	}

	return panelStyle.
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(fitLines(out, contentHeight))
}

func (m model) renderHelpBox() string {
	lines := []string{
		titleStyle.Render("Keymap"),
		"",
		"k/j, up/down       move between files",
		"[, ], left/right   scroll diff by line",
		"space, f/b         scroll diff by page",
		"ctrl+d/ctrl+u      scroll diff by page",
		"g/G                jump diff top/bottom",
		"s/u                stage/unstage selected file",
		"S/U                stage/unstage all changes",
		"c                  open commit message box",
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

func (m model) renderCommitBox() string {
	width := clamp(72, 40, max(40, m.width-8))
	height := clamp(12, 8, max(8, m.height-8))
	contentHeight := height - 4

	messageLines := strings.Split(m.commitMessage, "\n")
	if m.commitMessage == "" {
		messageLines = []string{mutedStyle.Render("Commit message")}
	}
	messageBody := fitLines(messageLines, contentHeight)

	help := mutedStyle.Render("ctrl+s commit  esc cancel")
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(1, 2).
		Render(titleStyle.Render("Commit") + "\n" + messageBody + "\n" + help)
}

func (m model) renderFooter() string {
	status := m.status
	if m.loading {
		status = "loading"
	}
	if m.err != nil {
		status = errorStyle.Render(m.err.Error())
	}

	keys := "k/j files  [/]/space diff  s/u file  S/U all  c commit  ? keys  q quit"
	if m.mode == commitMode {
		keys = "type message  ctrl+s commit  esc cancel"
	} else if m.showHelp {
		keys = "? hide help"
	}
	keysRendered := mutedStyle.Render(keys)

	leftWidth := max(0, m.width-lipgloss.Width(keysRendered)-3)
	left := truncate(status, leftWidth)
	return lipgloss.NewStyle().Width(m.width).Padding(0, 1).Render(left + "   " + keysRendered)
}

func (m model) diffHeight() int {
	return max(1, m.height-5)
}

func (m *model) clampDiffOffset() {
	maxOffset := max(0, len(m.diffLines())-m.diffHeight())
	if m.diffOffset > maxOffset {
		m.diffOffset = maxOffset
	}
	if m.diffOffset < 0 {
		m.diffOffset = 0
	}
}

func (m *model) scrollDiff(delta int) {
	m.diffOffset += delta
	m.clampDiffOffset()
}

func (m model) diffLines() []string {
	if m.diff == "" {
		return []string{""}
	}
	return strings.Split(m.diff, "\n")
}

func (m model) visibleFiles(count int) []git.FileStatus {
	if count <= 0 || len(m.files) == 0 {
		return nil
	}
	m.keepSelectedFileVisible()
	end := min(len(m.files), m.fileOffset+count)
	return m.files[m.fileOffset:end]
}

func (m *model) keepSelectedFileVisible() {
	if m.selected < m.fileOffset {
		m.fileOffset = m.selected
	}
	visibleCount := max(1, m.height-6)
	if m.selected >= m.fileOffset+visibleCount {
		m.fileOffset = m.selected - visibleCount + 1
	}
	maxOffset := max(0, len(m.files)-visibleCount)
	if m.fileOffset > maxOffset {
		m.fileOffset = maxOffset
	}
	if m.fileOffset < 0 {
		m.fileOffset = 0
	}
}

func colorDiffLine(line string) string {
	switch {
	case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
		return codeMuted.Render(line)
	case strings.HasPrefix(line, "+"):
		return addStyle.Render(line)
	case strings.HasPrefix(line, "-"):
		return delStyle.Render(line)
	case strings.HasPrefix(line, "@@"):
		return hunkStyle.Render(line)
	default:
		return line
	}
}

func overlayCommitBox(body, box string) string {
	bodyLines := strings.Split(body, "\n")
	boxLines := strings.Split(box, "\n")
	if len(bodyLines) == 0 || len(boxLines) == 0 {
		return body
	}

	startRow := max(0, (len(bodyLines)-len(boxLines))/2)
	for i, boxLine := range boxLines {
		row := startRow + i
		if row >= len(bodyLines) {
			break
		}
		lineWidth := lipgloss.Width(bodyLines[row])
		boxWidth := lipgloss.Width(boxLine)
		startCol := max(0, (lineWidth-boxWidth)/2)
		bodyLines[row] = replaceVisualSegment(bodyLines[row], boxLine, startCol)
	}

	return strings.Join(bodyLines, "\n")
}

func replaceVisualSegment(base, insert string, start int) string {
	if start <= 0 {
		return insert + truncate(base, max(0, lipgloss.Width(base)-lipgloss.Width(insert)))
	}

	prefix := truncate(base, start)
	suffixWidth := max(0, lipgloss.Width(base)-lipgloss.Width(prefix)-lipgloss.Width(insert))
	suffix := ""
	if suffixWidth > 0 {
		suffix = strings.Repeat(" ", suffixWidth)
	}
	return prefix + insert + suffix
}

func fitLines(lines []string, height int) string {
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > width-1 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func clamp(value, low, high int) int {
	return min(max(value, low), high)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

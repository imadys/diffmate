package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/imadys/diffmate/internal/git"
	"github.com/imadys/diffmate/internal/suggest"
)

type screenMode int

const (
	reviewMode screenMode = iota
	commitMode
)

type focusArea int

const (
	sidebarFocus focusArea = iota
	diffFocus
)

type sidebarTab int

const (
	changesTab sidebarTab = iota
	branchesTab
	commitsTab
	stashTab
)

type model struct {
	repo            git.Repo
	files           []git.FileStatus
	branches        []git.Branch
	commits         []git.Commit
	stashes         []git.Stash
	selected        int
	branchSelected  int
	commitSelected  int
	stashSelected   int
	diff            string
	diffOffset      int
	fileOffset      int
	width           int
	height          int
	err             error
	status          string
	loading         bool
	suggesting      bool
	suggestStarted  time.Time
	suggestElapsed  int
	mode            screenMode
	focus           focusArea
	tab             sidebarTab
	showHelp        bool
	showTabs        bool
	tabsEnabled     [4]bool
	tabMenuSelected int
	commitMessage   string
}

type refreshMsg struct {
	files    []git.FileStatus
	branches []git.Branch
	commits  []git.Commit
	stashes  []git.Stash
	diff     string
	err      error
}

type actionMsg struct {
	status string
	err    error
}

type suggestMsg struct {
	message string
	err     error
}

type suggestTickMsg struct{}

var (
	appStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	subtleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("60")).Bold(true)
	panelStyle    = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238"))
	headerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("235")).Bold(true)
	keyStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("60")).Bold(true).Padding(0, 1)
	keyBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("248")).Background(lipgloss.Color("236")).Padding(0, 1)
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("248")).Background(lipgloss.Color("235")).Padding(0, 1)
	addStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	delStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	hunkStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)
	codeMuted     = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	keywordStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	stringStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("150"))
	commentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
	tagStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))

	commentPattern = regexp.MustCompile(`(//.*|#.*)$`)
	stringPattern  = regexp.MustCompile(`"[^"]*"|'[^']*'|` + "`[^`]*`")
	tagPattern     = regexp.MustCompile(`</?[A-Za-z][A-Za-z0-9_.:-]*`)
	keywordPattern = regexp.MustCompile(`\b(import|from|export|function|const|let|var|type|interface|return|if|else|for|while|switch|case|default|class|extends|async|await|try|catch|new|nil|null|true|false|package|func|struct|map|range|go|defer|select)\b`)
)

func New(repo git.Repo) model {
	return model{
		repo:        repo,
		loading:     true,
		focus:       sidebarFocus,
		tab:         changesTab,
		tabsEnabled: [4]bool{true, true, true, true},
	}
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
		m.branches = msg.branches
		m.commits = msg.commits
		m.stashes = msg.stashes
		if m.selected >= len(m.files) {
			m.selected = len(m.files) - 1
		}
		if m.selected < 0 {
			m.selected = 0
		}
		m.clampSidebarSelections()
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
	case suggestMsg:
		m.suggesting = false
		m.suggestElapsed = 0
		m.err = msg.err
		if msg.err != nil {
			m.status = "commit suggestion failed"
			return m, nil
		}
		m.commitMessage = msg.message
		m.status = "commit message suggested"
		return m, nil
	case suggestTickMsg:
		if !m.suggesting {
			return m, nil
		}
		m.suggestElapsed = int(time.Since(m.suggestStarted).Round(time.Second).Seconds())
		return m, suggestTick()
	case tea.KeyMsg:
		if m.showTabs {
			return m.updateTabsMode(msg)
		}
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
	case "t":
		m.showTabs = true
		m.tabMenuSelected = int(m.tab)
		return m, nil
	case "tab":
		m.toggleFocus()
		return m, nil
	case "1":
		m.focus = sidebarFocus
		return m, nil
	case "2":
		m.focus = diffFocus
		return m, nil
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "r":
		m.loading = true
		m.status = "refreshing"
		return m, m.refresh()
	case "up", "k":
		if m.focus == sidebarFocus {
			m.moveSidebarSelection(-1)
			m.diffOffset = 0
			return m, m.loadDiff()
		}
		m.scrollDiff(-1)
	case "down", "j":
		if m.focus == sidebarFocus {
			m.moveSidebarSelection(1)
			m.diffOffset = 0
			return m, m.loadDiff()
		}
		m.scrollDiff(1)
	case "left", "h":
		if m.focus == sidebarFocus {
			m.moveTab(-1)
			return m, m.loadDiff()
		}
		m.scrollDiff(-1)
	case "right", "l":
		if m.focus == sidebarFocus {
			m.moveTab(1)
			return m, m.loadDiff()
		}
		m.scrollDiff(1)
	case "[":
		m.scrollDiff(-1)
	case "]":
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
	case "p":
		m.loading = true
		m.status = "pushing"
		return m, m.push()
	case "e", "enter":
		return m, m.openEditor()
	}

	return m, nil
}

func (m model) updateCommitMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.suggesting {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}

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
	case "ctrl+g":
		m.suggesting = true
		m.suggestStarted = time.Now()
		m.suggestElapsed = 0
		m.err = nil
		m.status = "asking codex for commit message"
		return m, tea.Batch(m.suggestCommitMessage(), suggestTick())
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

func (m model) updateTabsMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "t":
		m.showTabs = false
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.tabMenuSelected = max(0, m.tabMenuSelected-1)
	case "down", "j":
		m.tabMenuSelected = min(3, m.tabMenuSelected+1)
	case " ":
		tab := sidebarTab(m.tabMenuSelected)
		m.tabsEnabled[tab] = !m.tabsEnabled[tab]
		if !m.anyTabsEnabled() {
			m.tabsEnabled[tab] = true
		}
		if !m.tabsEnabled[m.tab] {
			m.tab = m.nextVisibleTab(1)
			m.diffOffset = 0
			return m, m.loadDiff()
		}
	case "enter":
		tab := sidebarTab(m.tabMenuSelected)
		if m.tabsEnabled[tab] {
			m.tab = tab
			m.showTabs = false
			m.diffOffset = 0
			return m, m.loadDiff()
		}
	}

	return m, nil
}

func (m *model) toggleFocus() {
	if m.focus == sidebarFocus {
		m.focus = diffFocus
		return
	}
	m.focus = sidebarFocus
}

func (m *model) moveTab(delta int) {
	if delta == 0 {
		return
	}
	m.tab = m.nextVisibleTab(delta)
	m.diffOffset = 0
}

func (m model) nextVisibleTab(delta int) sidebarTab {
	next := int(m.tab)
	for range 4 {
		next = (next + delta + 4) % 4
		if m.tabsEnabled[sidebarTab(next)] {
			return sidebarTab(next)
		}
	}
	return m.tab
}

func (m model) anyTabsEnabled() bool {
	for _, enabled := range m.tabsEnabled {
		if enabled {
			return true
		}
	}
	return false
}

func (m *model) moveSidebarSelection(delta int) {
	switch m.tab {
	case changesTab:
		m.selected = clamp(m.selected+delta, 0, max(0, len(m.files)-1))
		m.keepSelectedFileVisible()
	case branchesTab:
		m.branchSelected = clamp(m.branchSelected+delta, 0, max(0, len(m.branches)-1))
	case commitsTab:
		m.commitSelected = clamp(m.commitSelected+delta, 0, max(0, len(m.commits)-1))
	case stashTab:
		m.stashSelected = clamp(m.stashSelected+delta, 0, max(0, len(m.stashes)-1))
	}
}

func (m *model) clampSidebarSelections() {
	m.selected = clamp(m.selected, 0, max(0, len(m.files)-1))
	m.branchSelected = clamp(m.branchSelected, 0, max(0, len(m.branches)-1))
	m.commitSelected = clamp(m.commitSelected, 0, max(0, len(m.commits)-1))
	m.stashSelected = clamp(m.stashSelected, 0, max(0, len(m.stashes)-1))
	if !m.tabsEnabled[m.tab] {
		m.tab = m.nextVisibleTab(1)
	}
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading diffmate..."
	}

	header := m.renderHeader()
	headerHeight := lipgloss.Height(header)
	bodyHeight := max(5, m.height-headerHeight-2)
	sidebarWidth := clamp(34, 26, m.width/2)
	diffWidth := max(24, m.width-sidebarWidth)

	files := m.renderSidebar(sidebarWidth, bodyHeight)
	diff := m.renderDiff(diffWidth, bodyHeight)
	footer := m.renderFooter()
	body := lipgloss.JoinHorizontal(lipgloss.Top, files, diff)

	if m.mode == commitMode {
		body = overlayCommitBox(body, m.renderCommitBox())
	}
	if m.showHelp {
		body = overlayCommitBox(body, m.renderHelpBox())
	}
	if m.showTabs {
		body = overlayCommitBox(body, m.renderTabsBox())
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
		branches, err := m.repo.Branches(ctx)
		if err != nil {
			return refreshMsg{files: files, err: err}
		}
		commits, err := m.repo.Commits(ctx)
		if err != nil {
			return refreshMsg{files: files, branches: branches, err: err}
		}
		stashes, err := m.repo.Stashes(ctx)
		if err != nil {
			return refreshMsg{files: files, branches: branches, commits: commits, err: err}
		}

		m.files = files
		m.branches = branches
		m.commits = commits
		m.stashes = stashes
		m.clampSidebarSelections()

		diff, err := m.preview(ctx)
		if err != nil {
			return refreshMsg{files: files, branches: branches, commits: commits, stashes: stashes, err: err}
		}

		return refreshMsg{files: files, branches: branches, commits: commits, stashes: stashes, diff: diff}
	}
}

func (m model) loadDiff() tea.Cmd {
	return func() tea.Msg {
		diff, err := m.preview(context.Background())
		return refreshMsg{
			files:    m.files,
			branches: m.branches,
			commits:  m.commits,
			stashes:  m.stashes,
			diff:     diff,
			err:      err,
		}
	}
}

func (m model) preview(ctx context.Context) (string, error) {
	switch m.tab {
	case changesTab:
		if len(m.files) == 0 {
			return "", nil
		}
		return m.repo.Diff(ctx, m.files[m.selected])
	case branchesTab:
		if len(m.branches) == 0 {
			return "", nil
		}
		return m.repo.BranchPreview(ctx, m.branches[m.branchSelected])
	case commitsTab:
		if len(m.commits) == 0 {
			return "", nil
		}
		return m.repo.CommitDiff(ctx, m.commits[m.commitSelected])
	case stashTab:
		if len(m.stashes) == 0 {
			return "", nil
		}
		return m.repo.StashDiff(ctx, m.stashes[m.stashSelected])
	default:
		return "", nil
	}
}

func (m model) stage() tea.Cmd {
	if m.tab != changesTab {
		return nil
	}
	return m.withSelected("staged", func(ctx context.Context, file git.FileStatus) error {
		return m.repo.Stage(ctx, file)
	})
}

func (m model) unstage() tea.Cmd {
	if m.tab != changesTab {
		return nil
	}
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

func (m model) push() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.Push(context.Background())
		return actionMsg{status: "pushed current branch", err: err}
	}
}

func (m model) suggestCommitMessage() tea.Cmd {
	return func() tea.Msg {
		message, err := suggest.CommitMessage(context.Background(), m.repo.Root)
		return suggestMsg{message: message, err: err}
	}
}

func (m model) openEditor() tea.Cmd {
	if m.tab != changesTab || len(m.files) == 0 {
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
			{"ctrl+g", "codex suggest"},
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
		{"1/2", "focus"},
		{"tab", "swap"},
		{"t", "tabs"},
		{"j/k", "files"},
		{"[/]", "diff line"},
		{"space", "diff page"},
		{"g/G", "top/bottom"},
		{"s/u", "file stage"},
		{"S/U", "all stage"},
		{"c", "commit"},
		{"p", "push"},
		{"?", "all keys"},
	}
}

type keySegment struct {
	key   string
	label string
}

func (m model) renderKeySegments(segments []keySegment) string {
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		parts = append(parts, keyStyle.Render(segment.key)+" "+segment.label)
	}

	content := strings.Join(parts, mutedStyle.Render("  "))
	logo := miniLogo()
	available := max(1, m.width-lipgloss.Width(logo)-2)
	return keyBarStyle.Width(m.width).Render(logo + " " + truncate(content, available))
}

func (m model) renderSidebar(width, height int) string {
	contentHeight := max(1, height-2)
	repo := titleStyle.Render(truncate(m.repoName(), width-4))
	focusMark := ""
	if m.focus == sidebarFocus {
		focusMark = selectedStyle.Render("1")
	}

	lines := []string{repo + " " + focusMark, m.renderSidebarTabs(width - 4)}
	lines = append(lines, m.renderSidebarItems(width, contentHeight-2)...)

	return panelStyle.
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(fitLines(lines, contentHeight))
}

func (m model) renderSidebarTabs(width int) string {
	parts := make([]string, 0, 4)
	for _, tab := range []sidebarTab{changesTab, branchesTab, commitsTab, stashTab} {
		if !m.tabsEnabled[tab] {
			continue
		}
		label := tabLabel(tab)
		if tab == m.tab {
			label = selectedStyle.Render(label)
		} else {
			label = subtleStyle.Render(label)
		}
		parts = append(parts, label)
	}
	return truncate(strings.Join(parts, " "), width)
}

func (m model) renderSidebarItems(width, height int) []string {
	if height <= 0 {
		return nil
	}

	switch m.tab {
	case changesTab:
		return m.renderChangeItems(width, height)
	case branchesTab:
		return m.renderBranchItems(width, height)
	case commitsTab:
		return m.renderCommitItems(width, height)
	case stashTab:
		return m.renderStashItems(width, height)
	default:
		return nil
	}
}

func (m model) renderChangeItems(width, height int) []string {
	lines := []string{subtleStyle.Render("Changes")}
	if len(m.files) == 0 {
		return append(lines, mutedStyle.Render("No changes"))
	}

	visibleFiles := m.visibleFiles(height - 1)
	for visibleIndex, file := range visibleFiles {
		i := m.fileOffset + visibleIndex
		line := m.renderFileLine(file, width-4)
		if i == m.selected {
			line = selectedStyle.Width(width - 4).Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}

func (m model) renderBranchItems(width, _ int) []string {
	lines := []string{subtleStyle.Render("Local branches")}
	if len(m.branches) == 0 {
		return append(lines, mutedStyle.Render("No branches"))
	}
	for i, branch := range m.branches {
		prefix := "  "
		if branch.Current {
			prefix = "* "
		}
		line := truncate(prefix+branch.Name, width-4)
		if i == m.branchSelected {
			line = selectedStyle.Width(width - 4).Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}

func (m model) renderCommitItems(width, _ int) []string {
	lines := []string{subtleStyle.Render("Commits")}
	if len(m.commits) == 0 {
		return append(lines, mutedStyle.Render("No commits"))
	}
	for i, commit := range m.commits {
		line := truncate(commit.Hash+" "+commit.Subject, width-4)
		if i == m.commitSelected {
			line = selectedStyle.Width(width - 4).Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}

func (m model) renderStashItems(width, _ int) []string {
	lines := []string{subtleStyle.Render("Stash")}
	if len(m.stashes) == 0 {
		return append(lines, mutedStyle.Render("No stash entries"))
	}
	for i, stash := range m.stashes {
		line := truncate(stash.Name+" "+stash.Subject, width-4)
		if i == m.stashSelected {
			line = selectedStyle.Width(width - 4).Render(line)
		}
		lines = append(lines, line)
	}
	return lines
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
	title := m.previewTitle()
	if title != "" {
		scroll := fmt.Sprintf("  %d/%d", min(m.diffOffset+1, len(lines)), max(1, len(lines)))
		header = subtleStyle.Render(truncate(title, max(1, width-4-len(scroll)))) + mutedStyle.Render(scroll)
	}

	out := append([]string{header}, visible...)
	for i := 1; i < len(out); i++ {
		out[i] = colorDiffLine(truncate(out[i], width-4), m.selectedFilePath())
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
		"1 / 2              focus sidebar / diff",
		"tab                toggle focused panel",
		"t                  manage visible sidebar tabs",
		"left/right         switch tabs when sidebar is focused",
		"k/j, up/down       move between files",
		"[, ], left/right   scroll diff by line",
		"space, f/b         scroll diff by page",
		"ctrl+d/ctrl+u      scroll diff by page",
		"g/G                jump diff top/bottom",
		"s/u                stage/unstage selected file",
		"S/U                stage/unstage all changes",
		"c                  open commit message box",
		"p                  push current branch",
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

func (m model) renderTabsBox() string {
	lines := []string{
		titleStyle.Render("Sidebar Tabs"),
		"",
		mutedStyle.Render("space show/hide  enter open  esc close"),
		"",
	}

	for i, tab := range []sidebarTab{changesTab, branchesTab, commitsTab, stashTab} {
		check := "[ ]"
		if m.tabsEnabled[tab] {
			check = "[x]"
		}
		line := check + " " + tabLabel(tab)
		if i == m.tabMenuSelected {
			line = selectedStyle.Width(30).Render(line)
		}
		lines = append(lines, line)
	}

	return lipgloss.NewStyle().
		Width(38).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))
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
	if m.suggesting {
		help = mutedStyle.Render(fmt.Sprintf("asking codex for a commit message... %ds", m.suggestElapsed))
	} else {
		help = mutedStyle.Render("ctrl+g suggest  ctrl+s commit  esc cancel")
	}
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(1, 2).
		Render(titleStyle.Render("Commit") + "\n" + messageBody + "\n" + help)
}

func tabLabel(tab sidebarTab) string {
	switch tab {
	case changesTab:
		return "Changes"
	case branchesTab:
		return "Branches"
	case commitsTab:
		return "Commits"
	case stashTab:
		return "Stash"
	default:
		return "Tab"
	}
}

func suggestTick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return suggestTickMsg{}
	})
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

func (m model) diffHeight() int {
	return max(1, m.height-6)
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

func miniLogo() string {
	return titleStyle.Render("diffmate")
}

func (m model) selectedFilePath() string {
	if m.tab != changesTab || len(m.files) == 0 {
		return ""
	}
	return m.files[m.selected].Path
}

func (m model) previewTitle() string {
	switch m.tab {
	case changesTab:
		if len(m.files) == 0 {
			return ""
		}
		return m.files[m.selected].Path
	case branchesTab:
		if len(m.branches) == 0 {
			return ""
		}
		return m.branches[m.branchSelected].Name
	case commitsTab:
		if len(m.commits) == 0 {
			return ""
		}
		return m.commits[m.commitSelected].Hash
	case stashTab:
		if len(m.stashes) == 0 {
			return ""
		}
		return m.stashes[m.stashSelected].Name
	default:
		return ""
	}
}

func colorDiffLine(line, path string) string {
	switch {
	case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
		return codeMuted.Render(line)
	case strings.HasPrefix(line, "+"):
		return addStyle.Render("+") + highlightCode(line[1:], path)
	case strings.HasPrefix(line, "-"):
		return delStyle.Render("-") + highlightCode(line[1:], path)
	case strings.HasPrefix(line, " "):
		return " " + highlightCode(line[1:], path)
	case strings.HasPrefix(line, "@@"):
		return hunkStyle.Render(line)
	default:
		return highlightCode(line, path)
	}
}

func highlightCode(line, path string) string {
	if !highlightable(path) || strings.TrimSpace(line) == "" {
		return line
	}

	comment := ""
	if match := commentPattern.FindStringIndex(line); match != nil {
		comment = commentStyle.Render(line[match[0]:])
		line = line[:match[0]]
	}

	line = stringPattern.ReplaceAllStringFunc(line, func(match string) string {
		return stringStyle.Render(match)
	})
	line = tagPattern.ReplaceAllStringFunc(line, func(match string) string {
		return tagStyle.Render(match)
	})
	line = keywordPattern.ReplaceAllStringFunc(line, func(match string) string {
		return keywordStyle.Render(match)
	})

	return line + comment
}

func highlightable(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".json", ".md", ".css", ".scss", ".html", ".vue", ".svelte", ".py", ".rb", ".rs", ".java", ".kt", ".swift", ".php", ".sh", ".yml", ".yaml", ".toml":
		return true
	default:
		return false
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

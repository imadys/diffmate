package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/imadys/diffmate/internal/git"
	"github.com/imadys/diffmate/internal/settings"
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
	settings        settings.Settings
	showWelcome     bool
	showHelp        bool
	showTabs        bool
	configSection   int
	tabsEnabled     [4]bool
	tabMenuSelected int
	commitMessage   string
	commitError     string
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

func New(repo git.Repo) model {
	userSettings, err := settings.Load()
	if err != nil {
		userSettings = settings.Defaults()
	}

	return model{
		repo:        repo,
		loading:     true,
		focus:       sidebarFocus,
		tab:         changesTab,
		settings:    userSettings,
		showWelcome: !userSettings.SeenWelcome,
		tabsEnabled: tabsFromSettings(userSettings),
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
			m.commitError = ""
			m.status = msg.status
			if m.mode == commitMode {
				m.mode = reviewMode
				m.commitMessage = ""
			}
		} else if m.mode == commitMode {
			m.commitError = firstLine(msg.err.Error())
			m.status = "commit failed"
		}
		return m, m.refresh()
	case suggestMsg:
		m.suggesting = false
		m.suggestElapsed = 0
		m.err = msg.err
		if msg.err != nil {
			m.commitError = firstLine(msg.err.Error())
			m.status = "commit suggestion failed"
			return m, nil
		}
		m.commitMessage = msg.message
		m.commitError = ""
		m.status = "commit message suggested"
		return m, nil
	case suggestTickMsg:
		if !m.suggesting {
			return m, nil
		}
		m.suggestElapsed = int(time.Since(m.suggestStarted).Round(time.Second).Seconds())
		return m, suggestTick()
	case tea.KeyMsg:
		if m.showWelcome {
			return m.updateWelcomeMode(msg)
		}
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

func (m model) updateWelcomeMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	default:
		m.showWelcome = false
		m.settings.SeenWelcome = true
		_ = settings.Save(m.settings)
		return m, nil
	}
}

func (m model) updateReviewMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	case ",", "t":
		m.showTabs = true
		m.configSection = 0
		m.tabMenuSelected = int(m.tab)
		return m, nil
	case "tab":
		m.cycleFocus()
		return m, m.loadDiff()
	case "1":
		m.focusCard(changesTab)
		return m, m.loadDiff()
	case "2":
		m.focusCard(branchesTab)
		return m, m.loadDiff()
	case "3":
		m.focusCard(commitsTab)
		return m, m.loadDiff()
	case "4":
		m.focusCard(stashTab)
		return m, m.loadDiff()
	case "5":
		m.focus = diffFocus
		return m, nil
	case "shift+tab":
		m.cycleFocusBackward()
		return m, m.loadDiff()
	case "ctrl+1":
		m.focus = sidebarFocus
		return m, nil
	case "ctrl+2":
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
	case "left":
		if m.focus == diffFocus {
			m.focus = sidebarFocus
			return m, nil
		}
		m.moveTab(-1)
		return m, m.loadDiff()
	case "h":
		if m.focus == diffFocus {
			m.scrollDiff(-1)
			return m, nil
		}
		m.moveTab(-1)
		return m, m.loadDiff()
	case "right":
		if m.focus == sidebarFocus {
			m.focus = diffFocus
			return m, nil
		}
		m.scrollDiff(1)
	case "l":
		if m.focus == diffFocus {
			m.scrollDiff(1)
			return m, nil
		}
		m.moveTab(1)
		return m, m.loadDiff()
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
		m.commitError = ""
		m.status = "write commit message"
		return m, nil
	case "p":
		m.loading = true
		m.status = "pushing"
		return m, m.push()
	case "o":
		return m, m.openPreferredEditor()
	case "a":
		return m, m.openPreferredAgent()
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
		m.commitError = ""
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
		m.commitError = ""
		m.status = "asking " + m.settings.Agent + " for commit message"
		return m, tea.Batch(m.suggestCommitMessage(), suggestTick())
	case "enter":
		m.commitMessage += "\n"
	case "backspace":
		if len(m.commitMessage) > 0 {
			runes := []rune(m.commitMessage)
			m.commitMessage = string(runes[:len(runes)-1])
		}
	case " ":
		m.commitMessage += " "
	default:
		if msg.Type == tea.KeyRunes {
			m.commitMessage += msg.String()
		}
	}

	return m, nil
}

func (m model) updateTabsMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", ",", "t":
		m.showTabs = false
	case "q", "ctrl+c":
		return m, tea.Quit
	case "left", "h":
		m.configSection = max(0, m.configSection-1)
		m.alignConfigSelection()
	case "right", "l":
		m.configSection = min(2, m.configSection+1)
		m.alignConfigSelection()
	case "up", "k":
		m.moveConfigSelection(-1)
	case "down", "j":
		m.moveConfigSelection(1)
	case " ", "enter":
		cmd := m.applyConfigSelection()
		if cmd != nil {
			m.showTabs = false
			return m, cmd
		}
	}

	return m, nil
}

func (m *model) moveConfigSelection(delta int) {
	switch m.configSection {
	case 0:
		m.tabMenuSelected = clamp(m.tabMenuSelected+delta, 0, 3)
	case 1:
		m.tabMenuSelected = clamp(m.tabMenuSelected+delta, 0, len(editorOptions)-1)
	case 2:
		m.tabMenuSelected = clamp(m.tabMenuSelected+delta, 0, len(agentOptions)-1)
	}
}

func (m *model) alignConfigSelection() {
	switch m.configSection {
	case 0:
		m.tabMenuSelected = int(m.tab)
	case 1:
		m.tabMenuSelected = indexToolOption(editorOptions, m.settings.Editor)
	case 2:
		m.tabMenuSelected = indexToolOption(agentOptions, m.settings.Agent)
	}
}

func (m *model) applyConfigSelection() tea.Cmd {
	switch m.configSection {
	case 0:
		tab := sidebarTab(m.tabMenuSelected)
		m.tabsEnabled[tab] = !m.tabsEnabled[tab]
		if !m.anyTabsEnabled() {
			m.tabsEnabled[tab] = true
		}
		m.settings.Tabs = tabsToSettings(m.tabsEnabled)
		_ = settings.Save(m.settings)
		if !m.tabsEnabled[m.tab] {
			m.tab = m.nextVisibleTab(1)
			m.diffOffset = 0
			return m.loadDiff()
		}
	case 1:
		m.settings.Editor = editorOptions[m.tabMenuSelected].Command
		_ = settings.Save(m.settings)
		m.status = "preferred editor set to " + editorOptions[m.tabMenuSelected].Label
	case 2:
		m.settings.Agent = agentOptions[m.tabMenuSelected].Command
		_ = settings.Save(m.settings)
		m.status = "preferred agent set to " + agentOptions[m.tabMenuSelected].Label
	}
	return nil
}

func (m *model) focusCard(tab sidebarTab) {
	if !m.tabsEnabled[tab] {
		return
	}
	m.focus = sidebarFocus
	m.tab = tab
	m.diffOffset = 0
}

func (m *model) cycleFocus() {
	tabs := m.visibleTabs()
	if len(tabs) == 0 {
		m.focus = diffFocus
		return
	}
	if m.focus == diffFocus {
		m.focusCard(tabs[0])
		return
	}
	for i, tab := range tabs {
		if tab == m.tab {
			if i == len(tabs)-1 {
				m.focus = diffFocus
				return
			}
			m.focusCard(tabs[i+1])
			return
		}
	}
	m.focusCard(tabs[0])
}

func (m *model) cycleFocusBackward() {
	tabs := m.visibleTabs()
	if len(tabs) == 0 {
		m.focus = diffFocus
		return
	}
	if m.focus == diffFocus {
		m.focusCard(tabs[len(tabs)-1])
		return
	}
	for i, tab := range tabs {
		if tab == m.tab {
			if i == 0 {
				m.focus = diffFocus
				return
			}
			m.focusCard(tabs[i-1])
			return
		}
	}
	m.focusCard(tabs[0])
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

func (m model) visibleTabs() []sidebarTab {
	tabs := make([]sidebarTab, 0, 4)
	for _, tab := range []sidebarTab{changesTab, branchesTab, commitsTab, stashTab} {
		if m.tabsEnabled[tab] {
			tabs = append(tabs, tab)
		}
	}
	return tabs
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

func tabsFromSettings(userSettings settings.Settings) [4]bool {
	return [4]bool{
		userSettings.Tabs["changes"],
		userSettings.Tabs["branches"],
		userSettings.Tabs["commits"],
		userSettings.Tabs["stash"],
	}
}

func tabsToSettings(enabled [4]bool) map[string]bool {
	return map[string]bool{
		"changes":  enabled[changesTab],
		"branches": enabled[branchesTab],
		"commits":  enabled[commitsTab],
		"stash":    enabled[stashTab],
	}
}

func indexToolOption(options []toolOption, command string) int {
	for i, option := range options {
		if option.Command == command {
			return i
		}
	}
	return 0
}

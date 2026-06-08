package tui

import (
	"errors"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/imadys/diffmate/internal/git"
	"github.com/imadys/diffmate/internal/settings"
)

type screenMode int

const (
	reviewMode screenMode = iota
	commitMode
	confirmMode
	branchInputMode
	mergePickerMode
	conflictMode
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

type confirmAction int

const (
	confirmNone confirmAction = iota
	confirmDiscardChange
	confirmDeleteBranch
	confirmDeleteRemoteBranch
	confirmDeleteBranchBoth
	confirmAbortMerge
)

type model struct {
	repo             git.Repo
	files            []git.FileStatus
	branches         []git.Branch
	commits          []git.Commit
	stashes          []git.Stash
	conflicts        []git.FileStatus
	mergeInProgress  bool
	selected         int
	conflictSelected int
	branchSelected   int
	commitSelected   int
	stashSelected    int
	diff             string
	diffOffset       int
	diffViewport     viewport.Model
	conflictContent  string
	conflictViewport viewport.Model
	width            int
	height           int
	err              error
	status           string
	loading          bool
	suggesting       bool
	suggestStarted   time.Time
	suggestElapsed   int
	mode             screenMode
	focus            focusArea
	tab              sidebarTab
	settings         settings.Settings
	showWelcome      bool
	showHelp         bool
	showTabs         bool
	searchActive     bool
	searchQuery      string
	consoleVisible   bool
	consoleLines     []string
	configSection    int
	tabsEnabled      [4]bool
	tabMenuSelected  int
	commitMessage    string
	commitError      string
	cursorVisible    bool
	confirmAction    confirmAction
	confirmTitle     string
	confirmMessage   string
	branchNameInput  string
	mergeSelected    int
	inputError       string
}

type refreshMsg struct {
	files           []git.FileStatus
	branches        []git.Branch
	commits         []git.Commit
	stashes         []git.Stash
	diff            string
	conflictContent string
	mergeInProgress bool
	err             error
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
type cursorTickMsg struct{}
type autoRefreshMsg struct{}

func New(repo git.Repo) model {
	userSettings, err := settings.Load()
	if err != nil {
		userSettings = settings.Defaults()
	}

	return model{
		repo:           repo,
		loading:        true,
		focus:          sidebarFocus,
		tab:            changesTab,
		settings:       userSettings,
		showWelcome:    !userSettings.SeenWelcome,
		consoleVisible: true,
		tabsEnabled:    tabsFromSettings(userSettings),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.refresh(), autoRefreshTick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncDiffViewport()
		m.syncConflictViewport()
		return m, nil
	case refreshMsg:
		m.loading = false
		m.err = msg.err
		if msg.err != nil {
			m.appendConsoleError("refresh", msg.err)
			return m, nil
		}
		m.files = msg.files
		m.branches = msg.branches
		m.commits = msg.commits
		m.stashes = msg.stashes
		m.conflicts = conflictFiles(msg.files)
		m.mergeInProgress = msg.mergeInProgress
		if m.conflictSelected >= len(m.conflicts) {
			m.conflictSelected = len(m.conflicts) - 1
		}
		if m.conflictSelected < 0 {
			m.conflictSelected = 0
		}
		m.conflictContent = msg.conflictContent
		if m.mergeInProgress && (m.mode == reviewMode || m.mode == mergePickerMode) {
			m.mode = conflictMode
			if len(m.conflicts) > 0 {
				m.status = "merge conflict"
			} else {
				m.status = "merge ready"
			}
		}
		if !m.mergeInProgress && m.mode == conflictMode {
			m.mode = reviewMode
			m.status = "conflicts resolved"
		}
		if m.selected >= len(m.files) {
			m.selected = len(m.files) - 1
		}
		if m.selected < 0 {
			m.selected = 0
		}
		m.clampSidebarSelections()
		m.diff = msg.diff
		m.diffOffset = 0
		m.syncDiffViewport()
		m.syncConflictViewport()
		m.updateDiffViewportContent()
		m.diffViewport.GotoTop()
		if len(m.files) == 0 {
			m.status = "working tree clean"
		}
		return m, nil
	case actionMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.commitError = ""
			m.inputError = ""
			m.status = msg.status
			if m.mode == commitMode {
				m.mode = reviewMode
				m.commitMessage = ""
			}
			if m.mode == branchInputMode {
				m.mode = reviewMode
				m.branchNameInput = ""
			}
			if m.mode == mergePickerMode {
				m.mode = reviewMode
			}
		} else if m.mode == commitMode {
			m.commitError = firstLine(msg.err.Error())
			m.status = "commit failed"
			m.appendConsoleError("commit", msg.err)
		} else if m.mode == branchInputMode {
			m.inputError = firstLine(msg.err.Error())
			m.status = "new branch failed"
			m.appendConsoleError("branch", msg.err)
		} else {
			m.status = "action failed"
			m.appendConsoleError("git", msg.err)
		}
		return m, m.refresh()
	case suggestMsg:
		m.suggesting = false
		m.suggestElapsed = 0
		m.err = msg.err
		if msg.err != nil {
			m.commitError = firstLine(msg.err.Error())
			m.status = "commit suggestion failed"
			m.appendConsoleError("suggest", msg.err)
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
	case cursorTickMsg:
		if (m.mode != commitMode && m.mode != branchInputMode) || m.suggesting {
			return m, nil
		}
		m.cursorVisible = !m.cursorVisible
		return m, cursorTick()
	case autoRefreshMsg:
		if (m.mode != reviewMode && m.mode != conflictMode) || m.showWelcome || m.showTabs || m.showHelp || m.loading {
			return m, autoRefreshTick()
		}
		m.status = "auto refreshing"
		return m, tea.Batch(m.refresh(), autoRefreshTick())
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
		if m.searchActive {
			return m.updateSearchMode(msg)
		}
		if m.mode == commitMode {
			return m.updateCommitMode(msg)
		}
		if m.mode == confirmMode {
			return m.updateConfirmMode(msg)
		}
		if m.mode == branchInputMode {
			return m.updateBranchInputMode(msg)
		}
		if m.mode == mergePickerMode {
			return m.updateMergePickerMode(msg)
		}
		if m.mode == conflictMode {
			return m.updateConflictMode(msg)
		}
		return m.updateReviewMode(msg)
	}

	return m, nil
}

func (m model) updateWelcomeMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
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
	case "/":
		if m.focus != sidebarFocus {
			m.focus = sidebarFocus
		}
		m.searchActive = true
		m.searchQuery = ""
		m.status = "search " + strings.ToLower(sectionTitle(m.tab))
		return m, nil
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "~":
		m.consoleVisible = !m.consoleVisible
		m.syncDiffViewport()
		m.updateDiffViewportContent()
		return m, nil
	case "ctrl+l":
		m.consoleLines = nil
		m.status = "console cleared"
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
			m.diffViewport.GotoTop()
			return m, m.loadDiff()
		}
		m.scrollDiff(-1)
	case "down", "j":
		if m.focus == sidebarFocus {
			m.moveSidebarSelection(1)
			m.diffOffset = 0
			m.diffViewport.GotoTop()
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
	case "pgdown", "f":
		m.scrollDiff(m.diffHeight())
	case "ctrl+d":
		if m.focus == sidebarFocus && m.tab == branchesTab {
			if isProtectedBranch(m.selectedBranchName()) {
				m.err = errors.New("main and master are protected branches")
				m.status = "protected branch"
				m.appendConsoleError("branch", m.err)
				return m, nil
			}
			if len(m.branches) > 0 && m.branches[m.branchSelected].Current {
				m.err = errors.New("checkout another branch before deleting the current branch")
				m.status = "cannot delete current branch"
				m.appendConsoleError("branch", m.err)
				return m, nil
			}
			m.openConfirm(confirmDeleteBranchBoth, "Delete local and remote branch", "Delete "+m.selectedBranchName()+" locally and on GitHub?")
			return m, nil
		}
		m.scrollDiff(m.diffHeight())
	case " ":
		if m.focus == sidebarFocus {
			switch m.tab {
			case changesTab:
				return m, m.toggleStage()
			case branchesTab:
				return m, m.checkoutBranch()
			}
		}
		m.scrollDiff(m.diffHeight())
	case "g", "home":
		m.diffOffset = 0
		m.diffViewport.GotoTop()
	case "G", "end":
		m.diffViewport.GotoBottom()
		m.diffOffset = m.diffViewport.YOffset
	case "s":
		if m.focus == sidebarFocus && m.tab == changesTab {
			return m, m.stashChanges()
		}
		return m, m.stage()
	case "u":
		if m.focus == sidebarFocus && m.tab == branchesTab {
			return m, m.updateFromUpstream()
		}
		return m, m.unstage()
	case "S":
		return m, m.stageAll()
	case "U":
		return m, m.unstageAll()
	case "D":
		if m.focus == sidebarFocus && m.tab == changesTab {
			m.openConfirm(confirmDiscardChange, "Discard change", "Discard all changes for "+m.selectedChangeLabel()+"?")
			return m, nil
		}
		if m.focus == sidebarFocus && m.tab == branchesTab {
			if isProtectedBranch(m.selectedBranchName()) {
				m.err = errors.New("main and master are protected branches")
				m.status = "protected branch"
				m.appendConsoleError("branch", m.err)
				return m, nil
			}
			m.openConfirm(confirmDeleteRemoteBranch, "Delete remote branch", "Delete origin/"+m.selectedBranchName()+" on GitHub?")
			return m, nil
		}
	case "n":
		if m.focus == sidebarFocus && m.tab == branchesTab {
			m.mode = branchInputMode
			m.branchNameInput = ""
			m.cursorVisible = true
			m.inputError = ""
			m.status = "new branch"
			return m, cursorTick()
		}
	case "d":
		if m.focus == sidebarFocus && m.tab == branchesTab {
			if isProtectedBranch(m.selectedBranchName()) {
				m.err = errors.New("main and master are protected branches")
				m.status = "protected branch"
				m.appendConsoleError("branch", m.err)
				return m, nil
			}
			if len(m.branches) > 0 && m.branches[m.branchSelected].Current {
				m.err = errors.New("checkout another branch before deleting the current branch")
				m.status = "cannot delete current branch"
				m.appendConsoleError("branch", m.err)
				return m, nil
			}
			m.openConfirm(confirmDeleteBranch, "Delete branch", "Delete branch "+m.selectedBranchName()+"?")
			return m, nil
		}
	case "m":
		if m.focus == sidebarFocus && m.tab == branchesTab {
			if isProtectedBranch(m.currentBranchName()) {
				m.err = errors.New("main and master are protected from receiving merges in diffmate")
				m.status = "protected branch"
				m.appendConsoleError("branch", m.err)
				return m, nil
			}
			if len(m.branches) <= 1 {
				m.err = errors.New("no other branch available to merge")
				m.status = "merge unavailable"
				m.appendConsoleError("branch", m.err)
				return m, nil
			}
			m.mode = mergePickerMode
			m.mergeSelected = m.defaultMergeSelection()
			m.err = nil
			m.status = "select branch to merge"
			return m, nil
		}
	case "c":
		m.mode = commitMode
		m.cursorVisible = true
		m.err = nil
		m.commitError = ""
		m.status = "write commit message"
		return m, cursorTick()
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

func (m model) updateSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchActive = false
		m.searchQuery = ""
		m.status = "search cleared"
		return m, m.loadDiff()
	case "enter":
		m.searchActive = false
		m.status = "search applied"
		return m, m.loadDiff()
	case "backspace":
		if len(m.searchQuery) > 0 {
			runes := []rune(m.searchQuery)
			m.searchQuery = string(runes[:len(runes)-1])
			m.selectFirstSearchMatch()
			return m, m.loadDiff()
		}
	case "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.moveSidebarSelection(-1)
		return m, m.loadDiff()
	case "down", "j":
		m.moveSidebarSelection(1)
		return m, m.loadDiff()
	}

	if msg.Type == tea.KeyRunes {
		m.searchQuery += msg.String()
		m.selectFirstSearchMatch()
		return m, m.loadDiff()
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
	case "ctrl+d":
		m.commitMessage = ""
		m.commitError = ""
		m.status = "commit message cleared"
		return m, nil
	case "ctrl+g":
		m.suggesting = true
		m.suggestStarted = time.Now()
		m.suggestElapsed = 0
		m.err = nil
		m.commitError = ""
		m.status = "asking " + m.settings.Agent + " for commit message"
		return m, tea.Batch(m.suggestCommitMessage(), suggestTick())
	case "enter":
		m.commitError = ""
		m.commitMessage += "\n"
	case "backspace":
		m.commitError = ""
		if len(m.commitMessage) > 0 {
			runes := []rune(m.commitMessage)
			m.commitMessage = string(runes[:len(runes)-1])
		}
	case " ":
		m.commitError = ""
		m.commitMessage += " "
	default:
		if msg.Type == tea.KeyRunes {
			m.commitError = ""
			m.commitMessage += msg.String()
		}
	}

	return m, nil
}

func (m model) updateConfirmMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.closeConfirm("cancelled")
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	case "y", "enter", " ":
		action := m.confirmAction
		m.closeConfirm("confirmed")
		switch action {
		case confirmDiscardChange:
			return m, m.discardChange()
		case confirmDeleteBranch:
			return m, m.deleteBranch()
		case confirmDeleteRemoteBranch:
			return m, m.deleteRemoteBranch()
		case confirmDeleteBranchBoth:
			return m, m.deleteBranchBoth()
		case confirmAbortMerge:
			return m, m.abortMerge()
		default:
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateBranchInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = reviewMode
		m.branchNameInput = ""
		m.inputError = ""
		m.status = "new branch cancelled"
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		return m, m.createBranch()
	case "backspace":
		m.inputError = ""
		if len(m.branchNameInput) > 0 {
			runes := []rune(m.branchNameInput)
			m.branchNameInput = string(runes[:len(runes)-1])
		}
	case " ":
		m.inputError = ""
		m.branchNameInput += " "
	default:
		if msg.Type == tea.KeyRunes {
			m.inputError = ""
			m.branchNameInput += msg.String()
		}
	}
	return m, nil
}

func (m model) updateMergePickerMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = reviewMode
		m.status = "merge cancelled"
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.moveMergeSelection(-1)
	case "down", "j":
		m.moveMergeSelection(1)
	case "enter", " ":
		return m, m.mergeSelectedBranch()
	}
	return m, nil
}

func (m model) updateConflictMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.moveConflictSelection(-1)
		return m, m.loadConflictContent()
	case "down", "j":
		m.moveConflictSelection(1)
		return m, m.loadConflictContent()
	case "[", "left":
		m.scrollConflict(-1)
	case "]", "right":
		m.scrollConflict(1)
	case "pgup", "b", "ctrl+u":
		m.scrollConflict(-m.conflictHeight())
	case "pgdown", "f", " ":
		m.scrollConflict(m.conflictHeight())
	case "g", "home":
		m.conflictViewport.GotoTop()
	case "G", "end":
		m.conflictViewport.GotoBottom()
	case "o":
		return m, m.acceptOurs()
	case "t":
		return m, m.acceptTheirs()
	case "s":
		return m, m.stageConflict()
	case "e", "enter":
		return m, m.openConflictEditor()
	case "a":
		m.openConfirm(confirmAbortMerge, "Abort merge", "Abort the current merge?")
		return m, nil
	case "c":
		if len(m.conflicts) > 0 {
			m.err = errors.New("stage all conflicted files before continuing the merge")
			m.status = "merge blocked"
			m.appendConsoleError("merge", m.err)
			return m, nil
		}
		return m, m.continueMerge()
	case "r":
		m.loading = true
		m.status = "refreshing"
		return m, m.refresh()
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
	if m.tab != tab {
		m.searchActive = false
		m.searchQuery = ""
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
	m.searchActive = false
	m.searchQuery = ""
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
	if m.searchQuery != "" {
		m.moveFilteredSidebarSelection(delta)
		return
	}
	switch m.tab {
	case changesTab:
		m.selected = clamp(m.selected+delta, 0, max(0, len(m.files)-1))
	case branchesTab:
		m.branchSelected = clamp(m.branchSelected+delta, 0, max(0, len(m.branches)-1))
	case commitsTab:
		m.commitSelected = clamp(m.commitSelected+delta, 0, max(0, len(m.commits)-1))
	case stashTab:
		m.stashSelected = clamp(m.stashSelected+delta, 0, max(0, len(m.stashes)-1))
	}
}

func (m *model) moveMergeSelection(delta int) {
	if len(m.branches) == 0 {
		return
	}
	next := m.mergeSelected
	for range len(m.branches) {
		next = (next + delta + len(m.branches)) % len(m.branches)
		if !m.branches[next].Current {
			m.mergeSelected = next
			return
		}
	}
}

func (m *model) openConfirm(action confirmAction, title, message string) {
	m.mode = confirmMode
	m.confirmAction = action
	m.confirmTitle = title
	m.confirmMessage = message
	m.status = "confirm " + strings.ToLower(title)
}

func (m *model) closeConfirm(status string) {
	m.mode = reviewMode
	if m.mergeInProgress {
		m.mode = conflictMode
	}
	m.confirmAction = confirmNone
	m.confirmTitle = ""
	m.confirmMessage = ""
	m.status = status
}

func (m model) selectedChangeLabel() string {
	if len(m.files) == 0 {
		return "selected file"
	}
	return m.files[m.selected].Path
}

func (m model) selectedBranchName() string {
	if len(m.branches) == 0 {
		return "selected branch"
	}
	return m.branches[m.branchSelected].Name
}

func (m model) currentBranchName() string {
	for _, branch := range m.branches {
		if branch.Current {
			return branch.Name
		}
	}
	return ""
}

func (m model) defaultMergeSelection() int {
	for i, branch := range m.branches {
		if !branch.Current {
			return i
		}
	}
	return 0
}

func isProtectedBranch(name string) bool {
	return name == "main" || name == "master"
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

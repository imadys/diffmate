package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/imadys/diffmate/internal/updater"
	"github.com/imadys/diffmate/internal/version"
)

type docsMode int

const (
	docsBrowseMode docsMode = iota
	docsSearchMode
	docsEditMode
	docsConfirmExitMode
	docsNewFileMode
)

type docFile struct {
	Path string
}

type docLineKind int

const (
	docDirectoryLine docLineKind = iota
	docFileLine
)

type docLine struct {
	Label string
	Path  string
	Kind  docLineKind
}

type docsModel struct {
	root          string
	files         []docFile
	selected      int
	sidebarIndex  int
	content       string
	viewport      viewport.Model
	width         int
	height        int
	err           error
	status        string
	focus         focusArea
	collapsed     map[string]bool
	searchQuery   string
	mode          docsMode
	editLines     []string
	cursorLine    int
	cursorCol     int
	confirmQuit   bool
	switchReview  bool
	newFileInput  string
	newFileCursor int
	inputError    string
	selecting     bool
	selectLine    int
	selectCol     int
}

func NewDocs(root string) tea.Model {
	files, err := findMarkdownFiles(root)
	model := docsModel{
		root:      root,
		files:     files,
		err:       err,
		focus:     sidebarFocus,
		collapsed: map[string]bool{},
		status:    "ready",
	}
	if len(files) > 0 {
		model.content, model.err = readDocFile(root, files[0].Path)
	}
	return model
}

func (m docsModel) Init() tea.Cmd {
	return nil
}

func (m docsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncDocsViewport()
		m.updateDocsViewportContent()
	case updateMsg:
		if msg.err != nil {
			m.err = msg.err
			m.status = "update failed"
			return m, nil
		}
		m.err = nil
		m.status = msg.result.Message
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case docsSearchMode:
			return m.updateDocsSearchMode(msg)
		case docsEditMode:
			return m.updateDocsEditMode(msg)
		case docsConfirmExitMode:
			return m.updateDocsConfirmExitMode(msg)
		case docsNewFileMode:
			return m.updateDocsNewFileMode(msg)
		default:
			return m.updateDocsBrowseMode(msg)
		}
	}
	return m, nil
}

func (m docsModel) updateDocsBrowseMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch commandKey(msg) {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "R":
		m.switchReview = true
		return m, tea.Quit
	case "V", "v":
		m.status = "checking for update"
		return m, checkDocsUpdate()
	case "/":
		m.mode = docsSearchMode
		m.searchQuery = ""
		m.status = "search docs"
	case "n":
		m.mode = docsNewFileMode
		m.newFileInput = m.defaultNewDocPath()
		m.newFileCursor = len([]rune(m.newFileInput))
		m.inputError = ""
		m.status = "new doc"
	case "tab":
		if m.focus == sidebarFocus {
			m.focus = diffFocus
		} else {
			m.focus = sidebarFocus
		}
	case "right", "l":
		m.focus = diffFocus
	case "left", "h":
		if m.focus == diffFocus {
			m.focus = sidebarFocus
		} else {
			m.toggleSelectedDirectory()
		}
	case "up", "k":
		if m.focus == sidebarFocus {
			m.moveDocsSidebarSelection(-1)
		} else {
			m.scrollDocs(-1)
		}
	case "down", "j":
		if m.focus == sidebarFocus {
			m.moveDocsSidebarSelection(1)
		} else {
			m.scrollDocs(1)
		}
	case "shift+up":
		if m.focus == diffFocus {
			m.scrollDocs(-fastScrollAmount(m.docsHeight()))
		}
	case "shift+down":
		if m.focus == diffFocus {
			m.scrollDocs(fastScrollAmount(m.docsHeight()))
		}
	case " ":
		if m.focus == sidebarFocus {
			if m.selectedDocLine().Kind == docDirectoryLine {
				m.toggleSelectedDirectory()
			}
		} else {
			m.scrollDocs(m.docsHeight())
		}
	case "enter":
		if m.focus == sidebarFocus && m.selectedDocLine().Kind == docDirectoryLine {
			m.toggleSelectedDirectory()
			return m, nil
		}
		m.startDocsEdit()
	case "[", "pgup", "b", "ctrl+u":
		m.scrollDocs(-m.docsScrollAmount(msg.String()))
	case "]", "pgdown", "f", "ctrl+d":
		m.scrollDocs(m.docsScrollAmount(msg.String()))
	case "g", "home":
		m.viewport.GotoTop()
	case "G", "end":
		m.viewport.GotoBottom()
	case "r":
		m.refreshDocs()
	}
	return m, nil
}

func (m docsModel) updateDocsNewFileMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = docsBrowseMode
		m.newFileInput = ""
		m.newFileCursor = 0
		m.inputError = ""
		m.status = "new doc cancelled"
	case "enter":
		if err := m.createNewDocFile(); err != nil {
			m.inputError = firstLine(err.Error())
			m.status = "new doc failed"
			return m, nil
		}
	case "backspace":
		m.inputError = ""
		m.backspaceNewFileInput()
	case "left":
		m.newFileCursor = max(0, m.newFileCursor-1)
	case "right":
		m.newFileCursor = min(len([]rune(m.newFileInput)), m.newFileCursor+1)
	case "home", "ctrl+a":
		m.newFileCursor = 0
	case "end", "ctrl+e":
		m.newFileCursor = len([]rune(m.newFileInput))
	case " ":
		m.inputError = ""
		m.insertNewFileInput("-")
	default:
		if msg.Type == tea.KeyRunes {
			m.inputError = ""
			m.insertNewFileInput(msg.String())
		}
	}
	return m, nil
}

func (m docsModel) updateDocsSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = docsBrowseMode
		m.searchQuery = ""
		m.sidebarIndex = 0
		m.status = "search cancelled"
	case "enter":
		m.mode = docsBrowseMode
		m.status = "search applied"
	case "backspace":
		if len(m.searchQuery) > 0 {
			runes := []rune(m.searchQuery)
			m.searchQuery = string(runes[:len(runes)-1])
			m.sidebarIndex = 0
			m.selectCurrentDocLine()
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.searchQuery += msg.String()
			m.sidebarIndex = 0
			m.selectCurrentDocLine()
		}
	}
	return m, nil
}

func (m docsModel) updateDocsEditMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		if m.docsDirty() {
			m.mode = docsConfirmExitMode
			m.confirmQuit = true
			m.status = "unsaved changes"
			return m, nil
		}
		return m, tea.Quit
	case "q":
		if m.docsDirty() {
			m.mode = docsConfirmExitMode
			m.confirmQuit = true
			m.status = "unsaved changes"
			return m, nil
		}
		return m, tea.Quit
	case "esc":
		if m.docsDirty() {
			m.mode = docsConfirmExitMode
			m.confirmQuit = false
			m.status = "unsaved changes"
			return m, nil
		}
		m.mode = docsBrowseMode
		m.status = "edit cancelled"
	case "ctrl+s":
		m.saveEditedDoc()
	case "enter":
		m.insertEditNewline()
	case "backspace":
		m.backspaceEdit()
	case "left":
		m.clearEditSelection()
		m.moveEditCursorLeft()
	case "right":
		m.clearEditSelection()
		m.moveEditCursorRight()
	case "up":
		m.clearEditSelection()
		m.moveEditCursorUp()
	case "down":
		m.clearEditSelection()
		m.moveEditCursorDown()
	case "shift+left":
		m.startEditSelection()
		m.moveEditCursorLeft()
	case "shift+right":
		m.startEditSelection()
		m.moveEditCursorRight()
	case "shift+up":
		m.startEditSelection()
		m.moveEditCursorUp()
	case "shift+down":
		m.startEditSelection()
		m.moveEditCursorDown()
	default:
		if msg.Type == tea.KeyRunes {
			m.deleteEditSelection()
			m.insertEditText(msg.String())
		}
	}
	m.keepEditCursorVisible()
	return m, nil
}

func (m docsModel) updateDocsConfirmExitMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch commandKey(msg) {
	case "s", "ctrl+s":
		quit := m.confirmQuit
		m.saveEditedDoc()
		if quit && m.err == nil {
			return m, tea.Quit
		}
	case "d":
		quit := m.confirmQuit
		m.mode = docsBrowseMode
		m.editLines = nil
		m.status = "changes discarded"
		if quit {
			return m, tea.Quit
		}
	case "esc", "c":
		m.mode = docsEditMode
		m.confirmQuit = false
		m.status = "edit mode"
	}
	return m, nil
}

func (m docsModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading diffmate docs..."
	}

	sideInset := 1
	bodyGap := 1
	bodyWidth := max(1, m.width-sideInset*2)
	header := indentBlock(m.renderDocsHeader(bodyWidth), sideInset)
	footer := indentBlock(m.renderDocsFooter(bodyWidth), sideInset)
	bodyHeight := max(1, m.height-lipgloss.Height(header)-lipgloss.Height(footer))
	sidebarWidth := clamp(38, 26, bodyWidth/2)
	contentWidth := max(1, bodyWidth-sidebarWidth-bodyGap)
	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderDocsSidebar(sidebarWidth, bodyHeight),
		strings.Repeat(" ", bodyGap),
		m.renderDocsContent(contentWidth, bodyHeight),
	)
	if m.mode == docsConfirmExitMode {
		body = overlayCommitBox(body, m.renderDocsUnsavedBox())
	}
	if m.mode == docsNewFileMode {
		body = overlayCommitBox(body, m.renderDocsNewFileBox())
	}
	body = indentBlock(body, sideInset)
	return appStyle.Render(header + "\n" + body + "\n" + footer)
}

func (m docsModel) renderDocsHeader(width int) string {
	count := fmt.Sprintf("%d docs", len(m.filteredDocFiles()))
	if len(m.filteredDocFiles()) == 1 {
		count = "1 doc"
	}
	content := titleStyle.Render(filepath.Base(m.root)) + subtleStyle.Render("  "+count)
	if m.searchQuery != "" {
		content += subtleStyle.Render("  /" + m.searchQuery)
	}
	return headerStyle.Render(truncate(content, width))
}

func (m docsModel) renderDocsFooter(width int) string {
	if m.mode == docsSearchMode {
		return keyBarStyle.Render(truncate(miniLogo()+" "+subtleStyle.Render(version.Version)+" docs | search: /"+m.searchQuery+" | enter apply | esc cancel", width))
	}
	if m.mode == docsEditMode {
		dirty := ""
		if m.docsDirty() {
			dirty = " modified"
		}
		return keyBarStyle.Render(truncate(miniLogo()+" "+subtleStyle.Render(version.Version)+" edit"+dirty+" | ctrl+s save | shift+arrows select | esc close", width))
	}
	if m.mode == docsConfirmExitMode {
		return keyBarStyle.Render(truncate(miniLogo()+" "+subtleStyle.Render(version.Version)+" unsaved | s save | d discard | esc cancel", width))
	}
	if m.mode == docsNewFileMode {
		return keyBarStyle.Render(truncate(miniLogo()+" "+subtleStyle.Render(version.Version)+" new doc | enter create | esc cancel", width))
	}

	parts := []string{
		miniLogo() + " " + subtleStyle.Render(version.Version) + " docs",
		m.status,
		keyStyle.Render("j/k") + " files",
		keyStyle.Render("right") + " content",
		keyStyle.Render("space") + " folder/page",
		keyStyle.Render("/") + " search",
		keyStyle.Render("n") + " new",
		keyStyle.Render("R") + " review",
		keyStyle.Render("V") + " update",
		keyStyle.Render("enter") + " edit",
		keyStyle.Render("q") + " quit",
	}
	return keyBarStyle.Render(truncate(strings.Join(parts, " | "), width))
}

func checkDocsUpdate() tea.Cmd {
	return func() tea.Msg {
		result, err := updater.CheckAndInstall(context.Background(), version.Version)
		return updateMsg{result: result, err: err}
	}
}

func (m docsModel) renderDocsSidebar(width, height int) string {
	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	contentWidth := max(1, innerWidth-2)
	title := "Docs"
	if m.searchQuery != "" {
		title += "  /" + m.searchQuery
	}
	itemHeight := max(1, innerHeight-1)
	itemWidth := max(1, contentWidth-1)
	tree := m.visibleDocLines(itemWidth)
	itemLines := []string{}
	total := 0
	offset := 0
	if m.err != nil {
		itemLines = append(itemLines, errorStyle.Render(firstLine(m.err.Error())))
	} else if len(tree) == 0 {
		itemLines = append(itemLines, mutedStyle.Render("No markdown files"))
	} else {
		total = len(tree)
		visibleCount := itemHeight
		offset = keepIndexVisible(m.sidebarIndex, len(tree), visibleCount)
		end := min(len(tree), offset+visibleCount)
		for i, line := range tree[offset:end] {
			index := offset + i
			label := truncate(line.Label, itemWidth)
			if index == m.sidebarIndex {
				label = selectedLineStyle(m.focus == sidebarFocus && m.mode == docsBrowseMode, itemWidth).Render(label)
			} else if line.Kind == docDirectoryLine {
				label = subtleStyle.Render(label)
			}
			itemLines = append(itemLines, label)
		}
	}
	itemBlock := lipgloss.NewStyle().Width(itemWidth).Render(fitLines(itemLines, itemHeight))
	scrollbar := renderScrollbar(total, itemHeight, offset)
	out := titleStyle.Render(title) + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, itemBlock, scrollbar)

	border := lipgloss.Color("238")
	if m.focus == sidebarFocus && m.mode == docsBrowseMode {
		border = lipgloss.Color("86")
	}
	return lipgloss.NewStyle().
		Width(innerWidth).
		Height(innerHeight).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Render(fitLines(strings.Split(out, "\n"), innerHeight))
}

func (m docsModel) renderDocsContent(width, height int) string {
	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	contentWidth := max(1, innerWidth-2)
	title := subtleStyle.Render("Content")
	viewportWidth := max(1, contentWidth-1)
	lines := m.formattedDocContent(viewportWidth)
	if m.mode == docsEditMode || m.mode == docsConfirmExitMode {
		lines = m.formattedEditContent(viewportWidth)
	}
	if len(m.files) > 0 {
		scroll := fmt.Sprintf("  %d/%d", min(m.viewport.YOffset+1, len(lines)), max(1, len(lines)))
		title = subtleStyle.Render(truncate(m.files[m.selected].Path, max(1, contentWidth-len(scroll)))) + mutedStyle.Render(scroll)
	}

	vp := m.viewport
	vp.Width = viewportWidth
	vp.Height = max(1, innerHeight-1)
	vp.SetContent(strings.Join(lines, "\n"))

	border := lipgloss.Color("238")
	if m.focus == diffFocus || m.mode == docsEditMode || m.mode == docsConfirmExitMode {
		border = lipgloss.Color("86")
	}
	scrollbar := renderScrollbar(len(lines), vp.Height, vp.YOffset)
	out := title + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, vp.View(), scrollbar)
	return lipgloss.NewStyle().
		Width(innerWidth).
		Height(innerHeight).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Render(fitLines(strings.Split(out, "\n"), innerHeight))
}

func (m docsModel) renderDocsUnsavedBox() string {
	lines := []string{
		titleStyle.Render("Unsaved changes"),
		"",
		m.docsUnsavedMessage(),
		"",
		mutedStyle.Render("s save  d discard  esc cancel"),
	}
	return lipgloss.NewStyle().
		Width(clamp(64, 42, max(42, m.width-8))).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("203")).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))
}

func (m docsModel) renderDocsNewFileBox() string {
	input := m.renderNewFileInput()
	help := mutedStyle.Render("enter create  esc cancel")
	if m.inputError != "" {
		help = errorStyle.Render(m.inputError) + "\n" + help
	}
	lines := []string{
		titleStyle.Render("New doc"),
		"",
		input,
		"",
		help,
	}
	return lipgloss.NewStyle().
		Width(clamp(64, 40, max(40, m.width-8))).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))
}

func (m docsModel) renderNewFileInput() string {
	if m.newFileInput == "" {
		return inputCursor(true) + mutedStyle.Render("docs/new-file.md")
	}
	runes := []rune(m.newFileInput)
	cursor := clamp(m.newFileCursor, 0, len(runes))
	before := string(runes[:cursor])
	if cursor == len(runes) {
		return before + inputCursor(true)
	}
	return before + selectedStyle.Render(string(runes[cursor])) + string(runes[cursor+1:])
}

func (m docsModel) docsUnsavedMessage() string {
	if m.confirmQuit {
		return "Save changes before quitting?"
	}
	return "Save changes before closing edit mode?"
}

func (m docsModel) SwitchToReview() bool {
	return m.switchReview
}

func (m docsModel) defaultNewDocPath() string {
	line := m.selectedDocLine()
	dir := ""
	if line.Kind == docDirectoryLine {
		dir = line.Path
	} else if line.Kind == docFileLine {
		dir = filepath.Dir(line.Path)
		if dir == "." {
			dir = ""
		}
	}
	if dir == "" {
		return "new-doc.md"
	}
	return filepath.ToSlash(filepath.Join(dir, "new-doc.md"))
}

func (m docsModel) visibleDocLines(width int) []docLine {
	lines := make([]docLine, 0, len(m.files)*2)
	seenDirs := map[string]bool{}
	for _, file := range m.filteredDocFiles() {
		dir := filepath.Dir(file.Path)
		if dir == "." {
			dir = ""
		}
		hidden := false
		if dir != "" {
			parts := strings.Split(dir, "/")
			for depth := range parts {
				currentDir := strings.Join(parts[:depth+1], "/")
				if !seenDirs[currentDir] {
					seenDirs[currentDir] = true
					icon := "▾ "
					if m.collapsed[currentDir] {
						icon = "▸ "
					}
					lines = append(lines, docLine{
						Label: strings.Repeat("  ", depth) + icon + parts[depth] + "/",
						Path:  currentDir,
						Kind:  docDirectoryLine,
					})
				}
				if m.collapsed[currentDir] {
					hidden = true
					break
				}
			}
		}
		if hidden {
			continue
		}
		indent := 0
		if dir != "" {
			indent = len(strings.Split(dir, "/"))
		}
		lines = append(lines, docLine{
			Label: strings.Repeat("  ", indent) + filepath.Base(file.Path),
			Path:  file.Path,
			Kind:  docFileLine,
		})
	}
	for i := range lines {
		lines[i].Label = truncate(lines[i].Label, width)
	}
	return lines
}

func (m docsModel) docTreeLines(width int) []docLine {
	return m.visibleDocLines(width)
}

func (m docsModel) filteredDocFiles() []docFile {
	query := strings.ToLower(strings.TrimSpace(m.searchQuery))
	if query == "" {
		return m.files
	}
	filtered := make([]docFile, 0, len(m.files))
	for _, file := range m.files {
		if strings.Contains(strings.ToLower(file.Path), query) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func (m docsModel) selectedDocLine() docLine {
	lines := m.visibleDocLines(1000)
	if len(lines) == 0 {
		return docLine{}
	}
	return lines[clamp(m.sidebarIndex, 0, len(lines)-1)]
}

func (m docsModel) formattedDocContent(width int) []string {
	if len(m.files) == 0 {
		return []string{mutedStyle.Render("No markdown files found.")}
	}
	if rendered, err := renderMarkdownContent(m.content, width); err == nil {
		return rendered
	}
	return m.formattedRawDocContent(width)
}

func (m docsModel) formattedRawDocContent(width int) []string {
	lines := m.contentLines()
	formatted := make([]string, 0, len(lines))
	for _, line := range lines {
		for _, wrapped := range wrapLine(line, width) {
			formatted = append(formatted, highlightMarkdownLine(visualText(wrapped)))
		}
	}
	return formatted
}

func (m docsModel) formattedEditContent(width int) []string {
	lines := m.editLines
	if len(lines) == 0 {
		lines = []string{""}
	}
	formatted := make([]string, 0, len(lines))
	for index := range lines {
		line := truncate(lines[index], max(1, width-2))
		if index == m.cursorLine {
			line = m.renderEditLine(index, line)
		} else if m.lineHasSelection(index) {
			line = m.renderEditLine(index, line)
		} else {
			line = highlightMarkdownLine(line)
		}
		formatted = append(formatted, line)
	}
	return formatted
}

func renderMarkdownContent(content string, width int) ([]string, error) {
	if strings.TrimSpace(content) == "" {
		return []string{""}, nil
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(diffmateMarkdownStyle()),
		glamour.WithWordWrap(max(1, width)),
	)
	if err != nil {
		return nil, err
	}
	rendered, err := renderer.Render(strings.ReplaceAll(content, "\t", "    "))
	if err != nil {
		return nil, err
	}
	rendered = strings.TrimRight(rendered, "\n")
	if rendered == "" {
		return []string{""}, nil
	}
	return strings.Split(rendered, "\n"), nil
}

func diffmateMarkdownStyle() ansi.StyleConfig {
	style := styles.DarkStyleConfig
	style.H1.Prefix = ""
	style.H1.Suffix = ""
	style.H1.BackgroundColor = nil
	style.H2.Prefix = ""
	style.H3.Prefix = ""
	style.H4.Prefix = ""
	style.H5.Prefix = ""
	style.H6.Prefix = ""
	return style
}

func (m docsModel) contentLines() []string {
	if strings.TrimSpace(m.content) == "" {
		return []string{""}
	}
	return strings.Split(strings.ReplaceAll(m.content, "\t", "    "), "\n")
}

func highlightMarkdownLine(line string) string {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "#"):
		return titleStyle.Render(line)
	case strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* "):
		return addStyle.Render(line)
	case strings.HasPrefix(trimmed, ">"):
		return subtleStyle.Render(line)
	case strings.HasPrefix(trimmed, "```"):
		return hunkStyle.Render(line)
	default:
		return highlightCode(line, ".md")
	}
}

func (m *docsModel) moveDocsSidebarSelection(delta int) {
	lines := m.visibleDocLines(1000)
	if len(lines) == 0 {
		return
	}
	m.sidebarIndex = clamp(m.sidebarIndex+delta, 0, len(lines)-1)
	m.selectCurrentDocLine()
}

func (m *docsModel) selectCurrentDocLine() {
	line := m.selectedDocLine()
	if line.Kind != docFileLine {
		return
	}
	for i, file := range m.files {
		if file.Path == line.Path {
			if m.selected != i {
				m.selected = i
				m.viewport.GotoTop()
				m.loadSelectedDoc()
			}
			return
		}
	}
}

func (m *docsModel) toggleSelectedDirectory() {
	line := m.selectedDocLine()
	if line.Kind != docDirectoryLine || line.Path == "" {
		return
	}
	m.collapsed[line.Path] = !m.collapsed[line.Path]
	lines := m.visibleDocLines(1000)
	if m.sidebarIndex >= len(lines) {
		m.sidebarIndex = max(0, len(lines)-1)
	}
	m.selectCurrentDocLine()
}

func (m *docsModel) startDocsEdit() {
	if len(m.files) == 0 {
		return
	}
	m.mode = docsEditMode
	m.focus = diffFocus
	m.editLines = strings.Split(strings.ReplaceAll(m.content, "\t", "    "), "\n")
	if len(m.editLines) == 0 {
		m.editLines = []string{""}
	}
	m.cursorLine = clamp(m.viewport.YOffset, 0, len(m.editLines)-1)
	m.cursorCol = 0
	m.viewport.SetContent(strings.Join(m.editLines, "\n"))
	m.viewport.SetYOffset(m.cursorLine)
	m.status = "edit mode"
}

func (m *docsModel) keepEditCursorVisible() {
	if m.mode != docsEditMode && m.mode != docsConfirmExitMode {
		return
	}
	height := max(1, m.viewport.Height)
	m.viewport.SetContent(strings.Join(m.editLines, "\n"))
	m.viewport.SetYOffset(keepIndexVisible(m.cursorLine, len(m.editLines), height))
}

func (m *docsModel) saveEditedDoc() {
	if len(m.files) == 0 {
		return
	}
	content := strings.Join(m.editLines, "\n")
	if err := os.WriteFile(filepath.Join(m.root, filepath.FromSlash(m.files[m.selected].Path)), []byte(content), 0o644); err != nil {
		m.err = err
		m.status = "save failed"
		return
	}
	m.content = content
	m.err = nil
	m.status = "saved"
	m.mode = docsBrowseMode
	m.editLines = nil
	m.clearEditSelection()
	m.updateDocsViewportContent()
}

func (m docsModel) docsDirty() bool {
	if m.mode != docsEditMode && m.mode != docsConfirmExitMode {
		return false
	}
	return strings.Join(m.editLines, "\n") != m.content
}

func (m docsModel) renderEditLine(lineIndex int, line string) string {
	runes := []rune(line)
	cursorCol := -1
	if lineIndex == m.cursorLine {
		cursorCol = clamp(m.cursorCol, 0, len(runes))
	}
	startCol, endCol, selected := m.selectionColsForLine(lineIndex, len(runes))
	var out strings.Builder
	for i, r := range runes {
		char := string(r)
		if i == cursorCol {
			out.WriteString(selectedStyle.Render(char))
			continue
		}
		if selected && i >= startCol && i < endCol {
			out.WriteString(linkedStyle.Render(char))
			continue
		}
		out.WriteString(char)
	}
	if cursorCol == len(runes) {
		out.WriteString(selectedStyle.Render(" "))
	}
	return out.String()
}

func (m docsModel) lineHasSelection(lineIndex int) bool {
	if !m.selecting {
		return false
	}
	startLine, startCol, endLine, endCol := m.normalizedSelection()
	if startLine == endLine && startCol == endCol {
		return false
	}
	return lineIndex >= startLine && lineIndex <= endLine
}

func (m docsModel) selectionColsForLine(lineIndex, lineLen int) (int, int, bool) {
	if !m.selecting {
		return 0, 0, false
	}
	startLine, startCol, endLine, endCol := m.normalizedSelection()
	if lineIndex < startLine || lineIndex > endLine {
		return 0, 0, false
	}
	if startLine == endLine && startCol == endCol {
		return 0, 0, false
	}
	from := 0
	to := lineLen
	if lineIndex == startLine {
		from = min(startCol, lineLen)
	}
	if lineIndex == endLine {
		to = min(endCol, lineLen)
	}
	return from, to, from < to
}

func (m docsModel) normalizedSelection() (int, int, int, int) {
	startLine, startCol := m.selectLine, m.selectCol
	endLine, endCol := m.cursorLine, m.cursorCol
	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		return endLine, endCol, startLine, startCol
	}
	return startLine, startCol, endLine, endCol
}

func (m *docsModel) startEditSelection() {
	if m.selecting {
		return
	}
	m.selecting = true
	m.selectLine = m.cursorLine
	m.selectCol = m.cursorCol
}

func (m *docsModel) clearEditSelection() {
	m.selecting = false
	m.selectLine = 0
	m.selectCol = 0
}

func (m *docsModel) deleteEditSelection() bool {
	if !m.selecting {
		return false
	}
	startLine, startCol, endLine, endCol := m.normalizedSelection()
	if startLine == endLine && startCol == endCol {
		m.clearEditSelection()
		return false
	}
	startRunes := []rune(m.editLines[startLine])
	endRunes := []rune(m.editLines[endLine])
	startCol = clamp(startCol, 0, len(startRunes))
	endCol = clamp(endCol, 0, len(endRunes))
	merged := string(startRunes[:startCol]) + string(endRunes[endCol:])
	next := append([]string{}, m.editLines[:startLine]...)
	next = append(next, merged)
	next = append(next, m.editLines[endLine+1:]...)
	m.editLines = next
	m.cursorLine = startLine
	m.cursorCol = startCol
	m.clearEditSelection()
	return true
}

func (m *docsModel) insertEditText(value string) {
	if len(m.editLines) == 0 {
		m.editLines = []string{""}
	}
	line := []rune(m.editLines[m.cursorLine])
	col := clamp(m.cursorCol, 0, len(line))
	insert := []rune(value)
	next := append([]rune{}, line[:col]...)
	next = append(next, insert...)
	next = append(next, line[col:]...)
	m.editLines[m.cursorLine] = string(next)
	m.cursorCol = col + len(insert)
}

func (m *docsModel) insertEditNewline() {
	m.deleteEditSelection()
	line := []rune(m.editLines[m.cursorLine])
	col := clamp(m.cursorCol, 0, len(line))
	before := string(line[:col])
	after := string(line[col:])
	next := append([]string{}, m.editLines[:m.cursorLine]...)
	next = append(next, before, after)
	next = append(next, m.editLines[m.cursorLine+1:]...)
	m.editLines = next
	m.cursorLine++
	m.cursorCol = 0
}

func (m *docsModel) backspaceEdit() {
	if len(m.editLines) == 0 {
		return
	}
	if m.deleteEditSelection() {
		return
	}
	line := []rune(m.editLines[m.cursorLine])
	if m.cursorCol > 0 {
		col := clamp(m.cursorCol, 0, len(line))
		next := append([]rune{}, line[:col-1]...)
		next = append(next, line[col:]...)
		m.editLines[m.cursorLine] = string(next)
		m.cursorCol--
		return
	}
	if m.cursorLine == 0 {
		return
	}
	prevLen := len([]rune(m.editLines[m.cursorLine-1]))
	m.editLines[m.cursorLine-1] += m.editLines[m.cursorLine]
	m.editLines = append(m.editLines[:m.cursorLine], m.editLines[m.cursorLine+1:]...)
	m.cursorLine--
	m.cursorCol = prevLen
}

func (m *docsModel) moveEditCursorLeft() {
	if m.cursorCol > 0 {
		m.cursorCol--
		return
	}
	if m.cursorLine > 0 {
		m.cursorLine--
		m.cursorCol = len([]rune(m.editLines[m.cursorLine]))
	}
}

func (m *docsModel) moveEditCursorRight() {
	lineLen := len([]rune(m.editLines[m.cursorLine]))
	if m.cursorCol < lineLen {
		m.cursorCol++
		return
	}
	if m.cursorLine < len(m.editLines)-1 {
		m.cursorLine++
		m.cursorCol = 0
	}
}

func (m *docsModel) moveEditCursorUp() {
	if m.cursorLine > 0 {
		m.cursorLine--
		m.cursorCol = min(m.cursorCol, len([]rune(m.editLines[m.cursorLine])))
	}
}

func (m *docsModel) moveEditCursorDown() {
	if m.cursorLine < len(m.editLines)-1 {
		m.cursorLine++
		m.cursorCol = min(m.cursorCol, len([]rune(m.editLines[m.cursorLine])))
	}
}

func (m *docsModel) loadSelectedDoc() {
	if len(m.files) == 0 {
		m.content = ""
		return
	}
	content, err := readDocFile(m.root, m.files[m.selected].Path)
	m.content = content
	m.err = err
	m.updateDocsViewportContent()
}

func (m *docsModel) createNewDocFile() error {
	path, err := normalizeNewDocPath(m.newFileInput)
	if err != nil {
		return err
	}
	fullPath := filepath.Join(m.root, filepath.FromSlash(path))
	if _, err := os.Stat(fullPath); err == nil {
		return errors.New("file already exists")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(fullPath, []byte("# "+newDocTitle(path)+"\n"), 0o644); err != nil {
		return err
	}
	m.searchQuery = ""
	m.refreshDocs()
	for i, file := range m.files {
		if file.Path == path {
			m.selected = i
			m.selectDocPath(path)
			break
		}
	}
	m.newFileInput = ""
	m.newFileCursor = 0
	m.inputError = ""
	m.loadSelectedDoc()
	m.startDocsEdit()
	m.status = "created " + path
	return nil
}

func (m *docsModel) insertNewFileInput(value string) {
	runes := []rune(m.newFileInput)
	cursor := clamp(m.newFileCursor, 0, len(runes))
	insert := []rune(value)
	next := append([]rune{}, runes[:cursor]...)
	next = append(next, insert...)
	next = append(next, runes[cursor:]...)
	m.newFileInput = string(next)
	m.newFileCursor = cursor + len(insert)
}

func (m *docsModel) backspaceNewFileInput() {
	runes := []rune(m.newFileInput)
	cursor := clamp(m.newFileCursor, 0, len(runes))
	if cursor == 0 {
		return
	}
	next := append([]rune{}, runes[:cursor-1]...)
	next = append(next, runes[cursor:]...)
	m.newFileInput = string(next)
	m.newFileCursor = cursor - 1
}

func (m *docsModel) selectDocPath(path string) {
	lines := m.visibleDocLines(1000)
	for i, line := range lines {
		if line.Kind == docFileLine && line.Path == path {
			m.sidebarIndex = i
			return
		}
	}
}

func (m *docsModel) refreshDocs() {
	files, err := findMarkdownFiles(m.root)
	m.files = files
	m.err = err
	if m.selected >= len(m.files) {
		m.selected = len(m.files) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
	m.sidebarIndex = 0
	m.loadSelectedDoc()
	m.status = "refreshed"
}

func (m *docsModel) syncDocsViewport() {
	if m.width == 0 || m.height == 0 {
		return
	}
	bodyWidth := max(1, m.width-2)
	sidebarWidth := clamp(38, 26, bodyWidth/2)
	contentWidth := max(1, bodyWidth-sidebarWidth-1)
	bodyHeight := max(1, m.height-2)
	innerWidth := max(1, contentWidth-4)
	innerHeight := max(1, bodyHeight-2)
	m.viewport.Width = max(1, innerWidth-3)
	m.viewport.Height = max(1, innerHeight-1)
}

func (m *docsModel) updateDocsViewportContent() {
	m.syncDocsViewport()
	m.viewport.SetContent(strings.Join(m.formattedDocContent(max(1, m.viewport.Width)), "\n"))
}

func (m *docsModel) scrollDocs(delta int) {
	m.syncDocsViewport()
	m.updateDocsViewportContent()
	if delta > 0 {
		m.viewport.ScrollDown(delta)
	} else if delta < 0 {
		m.viewport.ScrollUp(-delta)
	}
}

func (m docsModel) docsHeight() int {
	return max(1, m.viewport.Height)
}

func (m docsModel) docsScrollAmount(key string) int {
	switch key {
	case "[", "]":
		return 1
	default:
		return m.docsHeight()
	}
}

func findMarkdownFiles(root string) ([]docFile, error) {
	var files []docFile
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() {
			if shouldSkipDocsDir(name) && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(name), ".md") {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, docFile{Path: filepath.ToSlash(rel)})
		}
		return nil
	})
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, err
}

func shouldSkipDocsDir(name string) bool {
	switch name {
	case ".git", "node_modules", "dist", "vendor", ".next", ".nuxt", ".astro", "coverage", "tmp":
		return true
	default:
		return false
	}
}

func readDocFile(root, path string) (string, error) {
	if path == "" {
		return "", errors.New("no document selected")
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func normalizeNewDocPath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	value = strings.TrimPrefix(value, "/")
	value = filepath.Clean(value)
	if value == "." || value == "" {
		return "", errors.New("file path cannot be empty")
	}
	value = filepath.ToSlash(value)
	if strings.HasPrefix(value, "../") || value == ".." || strings.Contains(value, "/../") {
		return "", errors.New("file path must stay inside the project")
	}
	if strings.EqualFold(filepath.Ext(value), "") {
		value += ".md"
	}
	if !strings.EqualFold(filepath.Ext(value), ".md") {
		return "", errors.New("docs files must use .md")
	}
	return value, nil
}

func newDocTitle(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	parts := strings.Fields(base)
	for i, part := range parts {
		runes := []rune(part)
		if len(runes) == 0 {
			continue
		}
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		parts[i] = string(runes)
	}
	if len(parts) == 0 {
		return "New Doc"
	}
	return strings.Join(parts, " ")
}

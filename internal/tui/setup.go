package tui

import (
	"context"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/imadys/diffmate/internal/git"
)

type setupModel struct {
	dir         string
	repoName    string
	width       int
	height      int
	err         error
	loading     bool
	initialized bool
}

type initRepoMsg struct {
	err error
}

func NewSetup(dir string) setupModel {
	return setupModel{
		dir:      dir,
		repoName: filepath.Base(dir),
	}
}

func (m setupModel) Initialized() bool {
	return m.initialized
}

func (m setupModel) Init() tea.Cmd {
	return nil
}

func (m setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case initRepoMsg:
		m.loading = false
		m.err = msg.err
		if msg.err != nil {
			return m, nil
		}
		m.initialized = true
		return m, tea.Quit
	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		case "enter":
			m.loading = true
			m.err = nil
			return m, m.initRepo()
		case "backspace":
			if len(m.repoName) > 0 {
				runes := []rune(m.repoName)
				m.repoName = string(runes[:len(runes)-1])
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.repoName += msg.String()
			}
		}
	}

	return m, nil
}

func (m setupModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading diffmate..."
	}

	logo := renderSetupLogo(m.width)
	message := "currently no git is setup here do you want to init git repo ?"
	input := m.renderInput()
	help := mutedStyle.Render("enter init repo  esc cancel")

	lines := []string{
		logo,
		"",
		titleStyle.Render(message),
		"",
		input,
		"",
		help,
	}

	if m.loading {
		lines = append(lines, "", subtleStyle.Render("initializing git repository..."))
	}
	if m.err != nil {
		lines = append(lines, "", errorStyle.Render(m.err.Error()))
	}

	boxWidth := clamp(76, 44, max(44, m.width-8))
	content := strings.Join(lines, "\n")
	box := panelStyle.
		Width(boxWidth).
		Padding(1, 2).
		Render(content)

	topPadding := strings.Repeat("\n", max(0, (m.height-lipgloss.Height(box))/2))
	return appStyle.Render(topPadding + lipgloss.PlaceHorizontal(m.width, lipgloss.Center, box))
}

func (m setupModel) renderInput() string {
	value := m.repoName
	if value == "" {
		value = " "
	}
	cursor := ""
	if !m.loading {
		cursor = selectedStyle.Render(" ")
	}

	label := subtleStyle.Render("repo name")
	field := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("60")).
		Padding(0, 1).
		Width(42).
		Render(value + cursor)

	return label + "\n" + field
}

func (m setupModel) initRepo() tea.Cmd {
	name := strings.TrimSpace(m.repoName)
	if name == "" {
		name = filepath.Base(m.dir)
	}

	return func() tea.Msg {
		_, err := git.Init(context.Background(), m.dir, name)
		return initRepoMsg{err: err}
	}
}

func renderSetupLogo(width int) string {
	return miniLogo()
}

package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/marshallku/tmux-powertools/internal/project"
	"github.com/marshallku/tmux-powertools/internal/tmux"
)

// Catppuccin Mocha palette
var (
	violet   = lipgloss.Color("#cba6f7")
	purple   = lipgloss.Color("#89b4fa")
	lavender = lipgloss.Color("#b4befe")
	green    = lipgloss.Color("#a6e3a1")
	red      = lipgloss.Color("#f38ba8")
	yellow   = lipgloss.Color("#f9e2af")
	blue     = lipgloss.Color("#89b4fa")
	muted    = lipgloss.Color("#6c7086")
	surface  = lipgloss.Color("#313244")
	text     = lipgloss.Color("#cdd6f4")

	titleStyle = lipgloss.NewStyle().
			Foreground(violet).
			Bold(true).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(violet).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(text)

	mutedStyle = lipgloss.NewStyle().
			Foreground(muted)

	branchStyle = lipgloss.NewStyle().
			Foreground(purple)

	dirtyStyle = lipgloss.NewStyle().
			Foreground(yellow)

	typeStyles = map[string]lipgloss.Style{
		"go":      lipgloss.NewStyle().Foreground(blue),
		"node":    lipgloss.NewStyle().Foreground(green),
		"rust":    lipgloss.NewStyle().Foreground(lipgloss.Color("#fab387")),
		"python":  lipgloss.NewStyle().Foreground(yellow),
		"generic": lipgloss.NewStyle().Foreground(muted),
	}

	statusBarStyle = lipgloss.NewStyle().
			Background(surface).
			Foreground(muted).
			Padding(0, 1)
)

type Model struct {
	projects  []project.Project
	filtered  []project.Project
	cursor    int
	search    textinput.Model
	quitting  bool
	selected  *project.Project
	width     int
	height    int
}

func NewModel(projects []project.Project) Model {
	ti := textinput.New()
	ti.Placeholder = "Search projects..."
	ti.Focus()
	ti.PromptStyle = lipgloss.NewStyle().Foreground(violet)
	ti.TextStyle = lipgloss.NewStyle().Foreground(text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(muted)

	return Model{
		projects: projects,
		filtered: projects,
		search:   ti,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if len(m.filtered) > 0 {
				m.selected = &m.filtered[m.cursor]
				return m, tea.Quit
			}

		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		}
	}

	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)

	// Filter projects
	query := strings.ToLower(m.search.Value())
	if query == "" {
		m.filtered = m.projects
	} else {
		m.filtered = nil
		for _, p := range m.projects {
			if fuzzyMatch(strings.ToLower(p.Name), query) {
				m.filtered = append(m.filtered, p)
			}
		}
	}

	// Keep cursor in bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}

	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("  tmux-powertools"))
	b.WriteString("\n\n")

	b.WriteString("  " + m.search.View())
	b.WriteString("\n\n")

	// Calculate visible window
	maxVisible := m.height - 7
	if maxVisible < 5 {
		maxVisible = 10
	}

	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := start; i < end; i++ {
		p := m.filtered[i]
		cursor := "  "
		nameStyle := normalStyle
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(violet).Render("▸ ")
			nameStyle = selectedStyle
		}

		// Project type badge
		typeStyle, ok := typeStyles[p.Type]
		if !ok {
			typeStyle = typeStyles["generic"]
		}
		typeBadge := typeStyle.Render(fmt.Sprintf("[%s]", p.Type))

		// Git info
		gitInfo := branchStyle.Render(" " + p.GitBranch)
		if p.GitDirty {
			gitInfo += dirtyStyle.Render(" ●")
		}
		if p.GitAhead > 0 {
			gitInfo += lipgloss.NewStyle().Foreground(green).Render(fmt.Sprintf(" ↑%d", p.GitAhead))
		}
		if p.GitBehind > 0 {
			gitInfo += lipgloss.NewStyle().Foreground(red).Render(fmt.Sprintf(" ↓%d", p.GitBehind))
		}

		line := fmt.Sprintf("%s%s %s%s", cursor, nameStyle.Render(p.Name), typeBadge, gitInfo)
		b.WriteString(line + "\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString(mutedStyle.Render("  No projects found"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(statusBarStyle.Render(fmt.Sprintf(" %d/%d projects  ↑↓ navigate  ⏎ select  esc quit", len(m.filtered), len(m.projects))))

	return b.String()
}

func (m Model) Selected() *project.Project {
	return m.selected
}

func fuzzyMatch(s, query string) bool {
	qi := 0
	for si := 0; si < len(s) && qi < len(query); si++ {
		if s[si] == query[qi] {
			qi++
		}
	}
	return qi == len(query)
}

func RunProjectSelector(projects []project.Project) (*project.Project, error) {
	m := NewModel(projects)
	p := tea.NewProgram(m, tea.WithAltScreen())

	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	finalModel := result.(Model)
	return finalModel.Selected(), nil
}

func OpenProject(p *project.Project) error {
	sessionName := strings.ReplaceAll(p.Name, ".", "_")

	if tmux.SessionExists(sessionName) {
		return tmux.SwitchSession(sessionName)
	}

	if err := tmux.CreateSession(sessionName, p.Path); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	if err := tmux.ApplyLayout(sessionName, p.Type); err != nil {
		return fmt.Errorf("failed to apply layout: %w", err)
	}

	return tmux.SwitchSession(sessionName)
}

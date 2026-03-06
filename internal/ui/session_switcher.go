package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/marshallku/tmux-powertools/internal/tmux"
)

type SessionModel struct {
	sessions []tmux.Session
	filtered []tmux.Session
	cursor   int
	search   textinput.Model
	quitting bool
	selected *tmux.Session
	width    int
	height   int
}

func NewSessionModel(sessions []tmux.Session) SessionModel {
	ti := textinput.New()
	ti.Placeholder = "Search sessions..."
	ti.Focus()
	ti.PromptStyle = lipgloss.NewStyle().Foreground(violet)
	ti.TextStyle = lipgloss.NewStyle().Foreground(text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(muted)

	return SessionModel{
		sessions: sessions,
		filtered: sessions,
		search:   ti,
	}
}

func (m SessionModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SessionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	query := strings.ToLower(m.search.Value())
	if query == "" {
		m.filtered = m.sessions
	} else {
		m.filtered = nil
		for _, s := range m.sessions {
			if fuzzyMatch(strings.ToLower(s.Name), query) {
				m.filtered = append(m.filtered, s)
			}
		}
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}

	return m, cmd
}

func (m SessionModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("  tmux sessions"))
	b.WriteString("\n\n")

	b.WriteString("  " + m.search.View())
	b.WriteString("\n\n")

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
		s := m.filtered[i]
		cursor := "  "
		nameStyle := normalStyle
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(violet).Render("▸ ")
			nameStyle = selectedStyle
		}

		info := mutedStyle.Render(fmt.Sprintf(" %d window(s)", s.Windows))
		attached := ""
		if s.Attached {
			attached = lipgloss.NewStyle().Foreground(green).Render(" ●")
		}

		line := fmt.Sprintf("%s%s%s%s", cursor, nameStyle.Render(s.Name), info, attached)
		b.WriteString(line + "\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString(mutedStyle.Render("  No sessions found"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(statusBarStyle.Render(fmt.Sprintf(" %d/%d sessions  ↑↓ navigate  ⏎ select  esc quit", len(m.filtered), len(m.sessions))))

	return b.String()
}

func (m SessionModel) Selected() *tmux.Session {
	return m.selected
}

func RunSessionSwitcher(sessions []tmux.Session) (*tmux.Session, error) {
	m := NewSessionModel(sessions)
	p := tea.NewProgram(m, tea.WithAltScreen())

	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	finalModel := result.(SessionModel)
	return finalModel.Selected(), nil
}

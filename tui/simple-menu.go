package tui

import (
	"github.com/charmbracelet/lipgloss"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Items  []string
	Cursor int
}

type MenuSelected struct {
	Item string
}

var (
	// Normal menu item style
	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")). // light gray text
			Background(lipgloss.Color("236")). // dark background
			Padding(0, 2).                     // more padding = looks bigger
			Margin(1, 0).                      // extra space between items
			Width(40).                         // force a consistent width
			Align(lipgloss.Center).            // center text
			Bold(true)

	// Highlighted menu item style
	menuItemSelected = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")). // bright yellow text
				Background(lipgloss.Color("57")).  // blue background
				Padding(0, 2).
				Margin(1, 0).
				Width(40).
				Align(lipgloss.Center).
				Bold(true)

	// Instruction text
	menuInstructionText = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Italic(true).
				Align(lipgloss.Center).
				Render("↑/↓ to navigate • Enter to select • q to quit")
)

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "up":
			if len(m.Items) == 0 {
				break
			}
			if m.Cursor == 0 {
				m.Cursor = len(m.Items) - 1
			} else {
				m.Cursor--
			}
		case "down":
			if len(m.Items) == 0 {
				break
			}
			m.Cursor = (m.Cursor + 1) % len(m.Items)
		case "enter":
			return m, func() tea.Msg {
				return MenuSelected{Item: m.Items[m.Cursor]} // send the selected item as a message
			}
		}
	}
	return m, nil
}

func (m *Model) View() string {
	b := strings.Builder{}
	b.WriteString("\n" + menuInstructionText + "\n\n\n\n")
	for i, item := range m.Items {
		if i == m.Cursor {
			b.WriteString(menuItemSelected.Render("  > " + item))
		} else {
			b.WriteString(menuItemStyle.Render("    " + item))
		}
		b.WriteString("\n")
	}
	return b.String()
}

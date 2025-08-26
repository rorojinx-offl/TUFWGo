package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	colorReset    = "\x1b[0m"
	colorSelected = "\x1b[95m"
	colorDim      = "\x1b[90m"
)

type Model struct {
	Items  []string
	Cursor int
}

type MenuSelected struct {
	Item string
}

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
	b.WriteString("\n Use up/down to navigate, q to quit.\n\n")
	for i, item := range m.Items {
		if i == m.Cursor {
			b.WriteString("  > ")
			b.WriteString(colorSelected)
			b.WriteString(item)
			b.WriteString(colorReset)
		} else {
			b.WriteString("    ")
			b.WriteString(colorDim)
			b.WriteString(item)
			b.WriteString(colorReset)
		}
		b.WriteString("\n")
	}
	return b.String()
}

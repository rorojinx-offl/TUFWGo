package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TabModel struct {
	Tabs        []string
	TabContent  []*Model
	EnumContent []*EnumModel
	activeTab   int
	selected    string
	Width       int
	Height      int
	child       tea.Model
}

func (m *TabModel) Init() tea.Cmd {
	return nil
}

func (m *TabModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.child != nil {
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "esc":
				m.child = nil
				return m, nil
			}
		}
		next, cmd := m.child.Update(msg)
		m.child = next
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q":
			return m, tea.Quit
		case "right":
			if m.selected != "" {
				return m, nil
			}
			m.activeTab = minimum(m.activeTab+1, len(m.Tabs)-1)
			return m, nil
		case "left":
			if m.selected != "" {
				return m, nil
			}
			m.activeTab = maximum(m.activeTab-1, 0)
			return m, nil
		default:
			menu, cmd := m.TabContent[m.activeTab].Update(msg)
			m.TabContent[m.activeTab] = menu.(*Model)

			return m, cmd
		}
	case MenuSelected:
		switch msg.Item {
		case "List Current Rules":
			m.child = NewModel()
			m.selected = ""
		default:
			m.selected = msg.Item
		}
		return m, nil
	case tea.WindowSizeMsg:
		// Track full terminal size
		m.Width = msg.Width
		m.Height = msg.Height

		// Forward an adjusted size to the active inner menu so it fills the inner window.
		if len(m.TabContent) > 0 && m.activeTab >= 0 && m.activeTab < len(m.TabContent) {
			inW := m.Width - docStyle.GetHorizontalFrameSize()
			if inW < 0 {
				inW = 0
			}
			inH := m.Height - docStyle.GetVerticalFrameSize()
			if inH < 0 {
				inH = 0
			}

			// Tabs row is roughly a single line.
			rowH := 1

			innerW := inW - windowStyle.GetHorizontalFrameSize()
			if innerW < 0 {
				innerW = 0
			}
			innerH := inH - rowH - windowStyle.GetVerticalFrameSize()
			if innerH < 0 {
				innerH = 0
			}

			menu, cmd := m.TabContent[m.activeTab].Update(tea.WindowSizeMsg{
				Width:  innerW,
				Height: innerH,
			})
			m.TabContent[m.activeTab] = menu.(*Model)
			return m, cmd
		}
	}

	return m, nil
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

var (
	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")
	docStyle          = lipgloss.NewStyle().Padding(1, 2, 1, 2)
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(highlightColor).Padding(0, 1)
	activeTabStyle    = inactiveTabStyle.Border(activeTabBorder, true)
	windowStyle       = lipgloss.NewStyle().BorderForeground(highlightColor).Padding(2, 0).Align(lipgloss.Center).Border(lipgloss.NormalBorder()).UnsetBorderTop()
)

func (m *TabModel) View() string {
	doc := strings.Builder{}

	var renderedTabs []string

	for i, t := range m.Tabs {
		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(m.Tabs)-1, i == m.activeTab
		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}
		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		} else if isLast && isActive {
			border.BottomRight = "│"
		} else if isLast && !isActive {
			border.BottomRight = "┤"
		}
		style = style.Border(border)
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	doc.WriteString(row)
	doc.WriteString("\n")

	content := ""
	/*if m.selected != "" {
		//content = "You selected: " + m.selected
	} else if len(m.TabContent) > 0 && m.activeTab >= 0 && m.activeTab < len(m.TabContent) {
		content = m.TabContent[m.activeTab].View()
	}*/

	if m.child != nil {
		content = m.child.View()
	} else if m.selected != "" {
		// optional: show a banner or a static screen for non-child actions
		content = "You selected: " + m.selected
	} else if len(m.TabContent) > 0 && m.activeTab >= 0 && m.activeTab < len(m.TabContent) {
		content = m.TabContent[m.activeTab].View() // default: simple menu in this tab
	}

	if m.Width > 0 && m.Height > 0 {
		inW := m.Width - docStyle.GetHorizontalFrameSize()
		if inW < 0 {
			inW = 0
		}
		inH := m.Height - docStyle.GetVerticalFrameSize()
		if inH < 0 {
			inH = 0
		}
		rowH := lipgloss.Height(row)
		winH := inH - rowH
		if winH < 0 {
			winH = 0
		}

		window := windowStyle.Width(inW).Height(winH).Render(content)

		doc.WriteString(row)
		doc.WriteString("\n")
		doc.WriteString(window)
		return docStyle.Render(doc.String())
	}
	window := windowStyle.Render(content)
	doc.WriteString(row)
	doc.WriteString("\n")
	doc.WriteString(window)
	return docStyle.Render(doc.String())
}

func maximum(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minimum(a, b int) int {
	if a < b {
		return a
	}
	return b
}

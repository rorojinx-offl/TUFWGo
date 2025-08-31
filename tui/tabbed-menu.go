package tui

import (
	"TUFWGo/ufw"
	"strings"
	"time"

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
	toast       string
	toastUntil  time.Time
}

type confirmDeclined struct{ ReturnTo tea.Model }

type confirmModel struct {
	prompt   string
	cmd      string
	data     FormData
	choice   int
	returnTo tea.Model
}

type clearToast struct{}

func (m *TabModel) Init() tea.Cmd {
	return nil
}

func (m *TabModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.child != nil {
		/*if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "esc":
				m.child = nil
				return m, nil
			}
		}*/

		switch child := msg.(type) {
		case tea.KeyMsg:
			if child.String() == "esc" {
				m.child = nil
				return m, nil
			}
		case FormCancelled:
			m.child = nil
			return m, nil
		case FormConfirmation:
			formStruct := child.Data
			structPass := ufw.Form{
				Action:     formStruct.Action,
				Direction:  formStruct.Direction,
				Interface:  formStruct.Interface,
				FromIP:     formStruct.FromIP,
				ToIP:       formStruct.ToIP,
				Port:       formStruct.Port,
				Protocol:   formStruct.Protocol,
				AppProfile: formStruct.App,
			}
			var cmd string
			if cmdCheck, err := structPass.ParseForm(); err != nil {
				cmd = "Error: " + err.Error()
				m.toast = cmd
				m.toastUntil = time.Now().Add(5 * time.Second)
				m.child = nil
				return m, tea.Tick(time.Until(m.toastUntil), func(time.Time) tea.Msg { return clearToast{} })
			} else {
				cmd = cmdCheck
			}
			if structPass.AppProfile == "" {
				m.child = newConfirmModel("Are you sure you want to submit the following command?", cmd, formStruct, m.child)
			} else {
				// Warn user about automatic IPv6 rule addition
				m.child = newConfirmModel("Are you sure you want to submit the following command?\n\nNote: Directly configuring an app profile will automatically add an IPv6 rule as well!", cmd, formStruct, m.child)
			}
			return m, nil
		case confirmDeclined:
			m.child = child.ReturnTo
			return m, nil

		case FormSubmitted:
			formStruct := child.Data
			structPass := ufw.Form{
				Action:     formStruct.Action,
				Direction:  formStruct.Direction,
				Interface:  formStruct.Interface,
				FromIP:     formStruct.FromIP,
				ToIP:       formStruct.ToIP,
				Port:       formStruct.Port,
				Protocol:   formStruct.Protocol,
				AppProfile: formStruct.App,
			}

			if cmd, err := structPass.ParseForm(); err != nil {
				m.toast = "Error: " + err.Error()
			} else {
				m.toast = "Successfully Parsed: " + cmd
			}

			m.toastUntil = time.Now().Add(5 * time.Second)
			m.child = nil
			return m, tea.Tick(time.Until(m.toastUntil), func(time.Time) tea.Msg { return clearToast{} })
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
		case "Add Rule":
			m.child = initialFormModel()
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
	case clearToast:
		m.toast = ""
		return m, nil
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
	//doc.WriteString(row)
	//doc.WriteString("\n")

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

		if winH < 20 || inW < 50 { // tweak thresholds as you like
			content = lipgloss.NewStyle().
				Align(lipgloss.Center).
				Render("Terminal too small — enlarge window and restart app to see tabs + content.")
		}

		window := windowStyle.Width(inW).Height(winH).Render(content)

		doc.WriteString(row)
		doc.WriteString("\n")
		doc.WriteString(window)

		if m.toast != "" && time.Now().Before(m.toastUntil) {
			toastStyle := lipgloss.NewStyle().
				MarginTop(1).
				Padding(0, 1).
				Bold(true)

			toastStyle = toastStyle.Foreground(highlightColor)
			doc.WriteString("\n")
			doc.WriteString(toastStyle.Render(m.toast))
		}

		return docStyle.Render(doc.String())
	}
	window := windowStyle.Render(content)
	doc.WriteString(row)
	doc.WriteString("\n")
	doc.WriteString(window)

	return docStyle.Render(doc.String())
}

func newConfirmModel(prompt, cmd string, data FormData, returnTo tea.Model) *confirmModel {
	return &confirmModel{
		prompt:   prompt,
		cmd:      cmd,
		data:     data,
		choice:   1,
		returnTo: returnTo,
	}
}

func (c *confirmModel) Init() tea.Cmd { return nil }

func (c *confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "right":
			if c.choice == 0 {
				c.choice = 1
			} else {
				c.choice = 0
			}
			return c, nil
		case "enter":
			if c.choice == 0 {
				return c, func() tea.Msg { return FormSubmitted{Data: c.data} }
			}
			return c, func() tea.Msg { return confirmDeclined{ReturnTo: c.returnTo} }
		case "esc":
			return c, func() tea.Msg { return confirmDeclined{ReturnTo: c.returnTo} }
		}
	}
	return c, nil
}

func (c *confirmModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("Confirm Submission")
	body := c.prompt + "\n\n" + lipgloss.NewStyle().Faint(true).Render(c.cmd)
	yes := "[ Yes ]"
	no := "[ No ]"

	if c.choice == 0 {
		yes = lipgloss.NewStyle().Bold(true).Render(yes)
	} else {
		no = lipgloss.NewStyle().Bold(true).Render(no)
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, yes+"  ", no)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlightColor).
		Padding(1, 2).
		Width(60)

	content := strings.Join([]string{title, body, "", buttons}, "\n")
	return lipgloss.Place(
		0, 0,
		lipgloss.Center, lipgloss.Center,
		box.Render(content))
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

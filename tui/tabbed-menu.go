package tui

import (
	"TUFWGo/system/local"
	"TUFWGo/system/ssh"
	"TUFWGo/ufw"
	"errors"
	"strconv"
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
	cmd         string
	rule        string
}

type confirmDeclined struct{ ReturnTo tea.Model }

type confirmModel struct {
	prompt   string
	cmd      string
	choice   int
	returnTo tea.Model
	onYes    func() tea.Msg
}

type errorBoxModel struct {
	prompt   string
	stderr   string
	returnTo tea.Model
}

type successBoxModel struct {
	prompt   string
	cmd      string
	returnTo tea.Model
}

type clearToast struct{}

func (m *TabModel) Init() tea.Cmd {
	return nil
}

func (m *TabModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.child != nil {
		switch child := msg.(type) {
		case tea.KeyMsg:
			if child.String() == "esc" {
				m.child = nil
				return m, nil
			}
			if child.String() == "r" {
				if _, ok := m.child.(EnumModel); ok {
					m.child = NewModel()
					return m, nil
				}
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
				m.child = newErrorBoxModel("There was an error parsing your input:", err.Error(), m.child)
				return m, nil
			} else {
				cmd = cmdCheck
				m.cmd = cmd
			}
			onYes := func() tea.Msg { return FormSubmitted{Data: formStruct} }
			var note string
			if structPass.AppProfile == "" {
				note = "Are you sure you want to submit the following command?"

			} else {
				// Warn user about automatic IPv6 rule addition
				note = "Are you sure you want to submit the following command?\n\nNote: Directly configuring an app profile will automatically add an IPv6 rule as well!"
			}
			m.child = newConfirmModel(note, cmd, m.child, onYes)
			return m, nil
		case confirmDeclined:
			m.child = child.ReturnTo
			return m, nil

		case FormSubmitted:
			_, err := local.RunCommand(m.cmd)
			if err != nil {
				m.child = newErrorBoxModel("There was an error executing your command!", err.Error(), m.child)
				return m, nil
			}
			// Show success message for 5 seconds
			m.child = newSuccessBoxModel("UFW successfully added the following rule:", m.cmd, nil)
			m.toastUntil = time.Now().Add(5 * time.Second)
			return m, tea.Tick(time.Until(m.toastUntil), func(time.Time) tea.Msg { return clearToast{} })
		case DeleteConfirmation:
			delInt, delError := child.number, child.error
			if delError != nil {
				m.child = newErrorBoxModel("There was an error deleting your rule!", delError.Error(), m.child)
				return m, nil
			}
			delCmd := "ufw delete " + strconv.Itoa(delInt)
			m.cmd = delCmd
			rule, err := ufw.ParseRuleFromNumber(delInt)
			m.rule = rule
			if err != nil {
				m.child = newErrorBoxModel("There was an error deleting your rule!", err.Error(), m.child)
			}
			onYes := func() tea.Msg { return DeleteExecuted{} }
			m.child = newConfirmModel("Are you sure you want to delete the following rule?", rule, m.child, onYes)
			return m, nil
		case DeleteExecuted:
			_, err := local.CommandConversation(m.cmd, "y\n")
			if err != nil {
				m.child = newErrorBoxModel("There was an error executing your command!", err.Error(), m.child)
				return m, nil
			}
			// Show success message for 5 seconds
			m.child = newSuccessBoxModel("UFW successfully deleted the following rule:", m.rule, nil)
			m.toastUntil = time.Now().Add(5 * time.Second)
			return m, tea.Tick(time.Until(m.toastUntil), func(time.Time) tea.Msg { return clearToast{} })
		case clearToast:
			if time.Now().After(m.toastUntil) {
				m.toastUntil = time.Time{}
				m.child = nil
				return m, nil
			}
		}
		if time.Now().Before(m.toastUntil) {
			// Still showing toast, don't process other messages
			return m, nil

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
		case "Remove Rule":
			m.child = DeleteList()
			m.selected = ""
		case "Test SSH Connection":
			if err := sshCheckup(); err != nil {
				m.child = newErrorBoxModel("SSH Connection Failed!", "Unable to connect to SSH server!", m.child)
			}
			m.child = newSuccessBoxModel("SSH Connection Successful!", "You are now connected via SSH!", m.child)
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

func sshCheckup() error {
	ok, _, err := ssh.GlobalClient.SendRequest("keepalive@openssh.com", true, nil)
	if err != nil || !ok {
		return errors.New("SSH Connection Failed!")
	}
	return nil
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

	content := ""

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

		var sshWarning string
		if SSHActive { // tweak thresholds as you like
			sshWarning = lipgloss.NewStyle().
				Align(lipgloss.Center).
				Render("SSH Mode Active!!!")
		}
		content = content + "\n" + sshWarning

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

func newConfirmModel(prompt, cmd string, returnTo tea.Model, onYes func() tea.Msg) *confirmModel {
	return &confirmModel{
		prompt:   prompt,
		cmd:      cmd,
		choice:   1,
		returnTo: returnTo,
		onYes:    onYes,
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
				if c.onYes != nil {
					return c, func() tea.Msg { return c.onYes() }
				}
				return c, func() tea.Msg { return confirmDeclined{ReturnTo: c.returnTo} }
			} else {
				return c, func() tea.Msg { return confirmDeclined{ReturnTo: c.returnTo} }
			}
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
		yes = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#35fc03")).Render(yes)
	} else {
		no = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fc0303")).Render(no)
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

func newErrorBoxModel(prompt, stderr string, returnTo tea.Model) *errorBoxModel {
	return &errorBoxModel{
		prompt:   prompt,
		stderr:   stderr,
		returnTo: returnTo,
	}
}

func (e *errorBoxModel) Init() tea.Cmd { return nil }

func (e *errorBoxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			return e.returnTo, nil
		}
	}
	return e, nil
}

func (e *errorBoxModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("Error!")
	body := e.prompt + "\n\n" + lipgloss.NewStyle().Faint(true).Render(e.stderr)
	back := "[ Back ]"
	back = lipgloss.NewStyle().Bold(true).Render(back)
	button := lipgloss.JoinHorizontal(lipgloss.Top, back)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlightColor).
		Padding(1, 2).
		Width(60)

	content := strings.Join([]string{title, body, "", button}, "\n")
	return lipgloss.Place(
		0, 0,
		lipgloss.Center, lipgloss.Center,
		box.Render(content))

}

func newSuccessBoxModel(prompt, cmd string, returnTo tea.Model) *successBoxModel {
	return &successBoxModel{
		prompt:   prompt,
		cmd:      cmd,
		returnTo: returnTo,
	}
}

func (s *successBoxModel) Init() tea.Cmd { return nil }

func (s *successBoxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearToast:
		return s.returnTo, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			return s.returnTo, nil
		}
	}
	return s, nil
}

func (s *successBoxModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("Success!")
	body := s.prompt + "\n\n" + lipgloss.NewStyle().Faint(true).Render(s.cmd)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlightColor).
		Padding(1, 2).
		Width(60)
	successContent := strings.Join([]string{title, body}, "\n")
	return lipgloss.Place(
		0, 0,
		lipgloss.Center, lipgloss.Center,
		box.Render(successContent))

}

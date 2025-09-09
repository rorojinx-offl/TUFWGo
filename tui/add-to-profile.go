package tui

import (
	"TUFWGo/ufw"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"sort"
	"strings"
)

/* ---------- Screen 1: Profile chooser with a dropdown ---------- */

// Reuse the dropdown UX pattern you already have in ipv4-form.go.
// Here is a tiny, local “list-of-files” wrapper around a dropdown.
type profileSelectModel struct {
	title    string
	dd       dropdown // use your dropdown component struct
	files    []string
	width    int
	height   int
	onChoose func(path string) tea.Msg // callback to produce a msg when chosen
	err      string
}

// NewProfileSelect takes a directory and lists *.json files.
// Leave baseDir empty ("") and you can inject your own later.
func NewProfileSelect(baseDir string, onChoose func(path string) tea.Msg) *profileSelectModel {
	files := listJSONProfiles(baseDir)
	dd := newDropdown("Choose a ruleset", files)
	dd.Width = 64
	return &profileSelectModel{
		title:    "Select Ruleset",
		dd:       dd,
		files:    files,
		onChoose: onChoose,
	}
}

func (m *profileSelectModel) Init() tea.Cmd { return nil }

func (m *profileSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case tea.KeyMsg:
		switch v.String() {
		case "esc", "q":
			return m, tea.Quit
		case "enter":
			// If the dropdown is open, Enter toggles it. If closed, confirm.
			if m.dd.Open {
				m.dd.Open = false
			} else {
				chosen := strings.TrimSpace(m.dd.Value())
				if chosen == "" {
					m.err = "No ruleset selected."
					return m, nil
				}
				if m.onChoose != nil {
					return m, func() tea.Msg { return m.onChoose(chosen) }
				}
			}
		}
	}
	m.dd.Update(msg) // arrow keys etc.
	return m, nil
}

func (m *profileSelectModel) View() string {
	title := focusStyle.Render(m.title)
	body := m.dd.View()
	if m.err != "" {
		body += "\n" + lipgloss.NewStyle().Foreground(errorColor).Render(m.err)
	}
	box := boxStyle.Copy().Width(56).Render(title + "\n\n" + body + "\n\n" + hintStyle.Render("Enter to confirm • ↑/↓ to move • Esc to exit"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func listJSONProfiles(base string) []string {
	// Leave base blank if you want to inject later.
	// This function returns just file basenames (or paths if you prefer).
	if base == "" {
		// placeholder: you can swap this to your own profiles dir resolver
		base = "./profiles"
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

/* ---------- Messages to move between screens ---------- */

type ProfileChosen struct{ Path string }
type RuleAdded struct{ RuleMem []string } // feedback toast: how many in memory
type RuleSubmit struct{}                  // you’ll wire backend later

/* ---------- Screen 2: Simplified rule form (no App Profile) ---------- */

type simpleRuleForm struct {
	// dropdowns
	action    dropdown
	direction dropdown
	iface     dropdown

	// inputs
	fromIP   textinput.Model
	toIP     textinput.Model
	port     textinput.Model
	protocol textinput.Model

	// buttons and focus
	focusIdx int // 0..N widgets (fields + buttons)
	width    int
	height   int

	// state
	pending []string // in-memory array you asked for
	profile string   // selected profile path (display only)
}

const (
	sfAction = iota
	sfDirection
	sfInterface
	sfFromIP
	sfToIP
	sfPort
	sfProtocol
	sfAddBtn
	sfSubmitBtn
	sfCount
)

func NewSimpleRuleForm(profilePath string) *simpleRuleForm {
	actions := []string{"allow", "deny", "reject", "limit"}
	directions := []string{"default", "in", "out"}
	ifaces := []string{"default"}
	ifaces = append(ifaces, listInterfaces()...) // you already have this in ipv4-form.go

	m := &simpleRuleForm{
		action:    newDropdown("Action", actions),
		direction: newDropdown("Direction", directions),
		iface:     newDropdown("Interface", ifaces),
		focusIdx:  sfAction,
		profile:   profilePath,
	}

	m.fromIP = textinput.New()
	m.fromIP.Placeholder = "e.g. 192.168.1.10 or 10.0.0.0/24"
	m.fromIP.Prompt = ""
	m.fromIP.Width = 38

	m.toIP = textinput.New()
	m.toIP.Placeholder = "e.g. any or 203.0.113.5"
	m.toIP.Prompt = ""
	m.toIP.Width = 38

	m.port = textinput.New()
	m.port.Placeholder = "e.g. 22 or 80,443"
	m.port.Prompt = ""
	m.port.Width = 18

	m.protocol = textinput.New()
	m.protocol.Placeholder = "tcp | udp | all | tcp/udp | esp | ah | gre | icmp | ipv6"
	m.protocol.Prompt = ""
	m.protocol.Width = 58

	m.updateFocus()
	return m
}

func (m *simpleRuleForm) Init() tea.Cmd { return textinput.Blink }

func (m *simpleRuleForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case tea.KeyMsg:
		switch v.String() {
		case "esc", "q":
			// go back to parent (tabbed menu or previous child)
			return m, func() tea.Msg { return FormCancelled{} }
		case "tab", "shift+tab":
			dir := 1
			if v.String() == "shift+tab" {
				dir = -1
			}
			m.focusIdx = (m.focusIdx + dir + sfCount) % sfCount
			m.updateFocus()
			return m, nil
		case "enter":
			// When focus is on buttons:
			if m.focusIdx == sfAddBtn {
				// Build a Rule from current fields and push to in-memory slice.
				r, err := m.collectRule()
				if err != nil {
					return newErrorBoxModel("Invalid Rule:", err.Error(), m), nil
				}
				m.pending = append(m.pending, r)
				return m, func() tea.Msg { return RuleAdded{RuleMem: m.pending} }
			}
			if m.focusIdx == sfSubmitBtn {
				// You’ll wire the backend later; we just emit a message.
				return m, func() tea.Msg { return RuleSubmit{} }
			}
		}
	}

	// Route to focused control
	switch m.focusIdx {
	case sfAction:
		m.action.Update(msg)
	case sfDirection:
		m.direction.Update(msg)
	case sfInterface:
		m.iface.Update(msg)
	case sfFromIP:
		var cmd tea.Cmd
		m.fromIP, cmd = m.fromIP.Update(msg)
		return m, cmd
	case sfToIP:
		var cmd tea.Cmd
		m.toIP, cmd = m.toIP.Update(msg)
		return m, cmd
	case sfPort:
		var cmd tea.Cmd
		m.port, cmd = m.port.Update(msg)
		return m, cmd
	case sfProtocol:
		var cmd tea.Cmd
		m.protocol, cmd = m.protocol.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *simpleRuleForm) View() string {
	header := focusStyle.Render("Add Rules to: ") + hintStyle.Render(m.profile)

	left := strings.Join([]string{
		m.action.View(),
		m.direction.View(),
		m.iface.View(),
		renderField("From IP", m.fromIP.View(), false),
	}, "\n\n")

	right := strings.Join([]string{
		renderField("To IP", m.toIP.View(), false),
		renderField("Port", m.port.View(), false),
		renderField("Protocol", m.protocol.View(), false),
	}, "\n\n")

	row := lipgloss.JoinHorizontal(lipgloss.Top, left+"\n\n", right)

	// Buttons
	addBtn := "[ Add ]"
	submitBtn := "[ Submit ]"
	if m.focusIdx == sfAddBtn {
		addBtn = focusStyle.Render(addBtn)
	} else {
		addBtn = hintStyle.Render(addBtn)
	}
	if m.focusIdx == sfSubmitBtn {
		submitBtn = focusStyle.Render(submitBtn)
	} else {
		submitBtn = hintStyle.Render(submitBtn)
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, addBtn+"   ", submitBtn)

	content := strings.Join([]string{
		header,
		sepStyle.Render(strings.Repeat("─", 80)),
		row,
		"",
		buttons,
		"",
		hintStyle.Render(fmt.Sprintf("Pending in memory: %d", len(m.pending))),
	}, "\n")

	outer := boxStyle.Copy().Width(90).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, outer)
}

func (m *simpleRuleForm) updateFocus() {
	m.fromIP.Blur()
	m.toIP.Blur()
	m.port.Blur()
	m.protocol.Blur()

	switch m.focusIdx {
	case sfFromIP:
		m.fromIP.Focus()
	case sfToIP:
		m.toIP.Focus()
	case sfPort:
		m.port.Focus()
	case sfProtocol:
		m.protocol.Focus()
	}
}

func (m *simpleRuleForm) collectRule() (string, error) {
	dir := m.direction.Value()
	if dir == "default" {
		dir = ""
	}
	iface := m.iface.Value()
	if iface == "default" {
		iface = ""
	}
	rf := &ufw.Form{
		Action:    m.action.Value(),
		Direction: dir,
		Interface: iface,
		FromIP:    m.fromIP.Value(),
		ToIP:      m.toIP.Value(),
		Port:      m.port.Value(),
		Protocol:  m.protocol.Value(),
	}
	cmd, err := rf.ParseForm()
	if err != nil {
		return "", err
	}
	return cmd, nil
}

/* ---------- Glue: how to chain the two screens ---------- */

// Example “entry point” model that first shows the profile chooser.
// Replace baseDir with your actual profiles path later.
type profilesFlow struct {
	child tea.Model
}

func NewProfilesFlow() *profilesFlow {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Println("No access to user config dir:", err)
		return nil
	}
	onChoose := func(path string) tea.Msg { return ProfileChosen{Path: path} }
	return &profilesFlow{
		child: NewProfileSelect(baseDir+"/tufwgo/profiles", onChoose),
	}
}

func (m *profilesFlow) Init() tea.Cmd { return nil }

func (m *profilesFlow) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.child != nil {
		switch v := msg.(type) {
		case ProfileChosen:
			m.child = NewSimpleRuleForm(v.Path)
			return m, nil
		case FormCancelled:
			// exit flow
			return m, tea.Quit
		case RuleAdded:
			// Optional: toast/feedback
			return m, nil
		case RuleSubmit:
			// You’ll wire backend later. For now, just quit back to parent.
			return m, tea.Quit
		}
		next, cmd := m.child.Update(msg)
		m.child = next
		return m, cmd
	}
	return m, nil
}

func (m *profilesFlow) View() string {
	if m.child != nil {
		return m.child.View()
	}
	return "Done."
}

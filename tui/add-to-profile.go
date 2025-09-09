package tui

import (
	"TUFWGo/ufw"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type profileSelectModel struct {
	title    string
	dd       dropdown // use your dropdown component struct
	files    []string
	width    int
	height   int
	onChoose func(path string) tea.Msg // callback to produce a msg when chosen
	err      string
}

type InvalidProfile struct{}

// NewProfileSelect takes a directory and lists *.json files.
func NewProfileSelect(baseDir string, onChoose func(path string) tea.Msg) *profileSelectModel {
	files := listJSONProfiles(baseDir)
	dd := newDropdown("Choose a ruleset", files)
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
			if !m.dd.Open {
				m.dd.Open = true
				return m, nil
			}

			chosen := strings.TrimSpace(m.dd.Value())
			if chosen == "" {
				m.err = "No ruleset selected."
				return m, nil
			}

			path := filepath.Join(baseDir+"/tufwgo/profiles", chosen)
			sealed, err := profileIsSealed(path)
			if err != nil {
				m.err = fmt.Sprintf("Failed to check profile: %v", err)
				return m, nil
			}
			if sealed {
				return m, func() tea.Msg { return InvalidProfile{} }
			}

			if m.onChoose != nil {
				return m, func() tea.Msg { return m.onChoose(chosen) }
			}
			return m, nil
		case "up", "down":
			if !m.dd.Open {
				m.dd.Open = true
			}
		}
	}
	m.dd.Update(msg) // arrow keys etc.
	return m, nil
}

func (m *profileSelectModel) View() string {
	var b strings.Builder
	b.WriteString(focusStyle.Render(m.title) + "\n")
	b.WriteString(hintStyle.Render("Enter to open/confirm • ↑/↓ to choose • Esc to close") + "\n")
	b.WriteString(sepStyle.Render(strings.Repeat("─", 80)) + "\n\n")
	b.WriteString(m.dd.View())
	if m.err != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(errorColor).Render(m.err))
	}
	return b.String()
}

func listJSONProfiles(base string) []string {
	if base == "" {
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
type RuleAdded struct {
	CmdMem  []string
	RuleMem []ruleFormat
}
type RuleSubmit struct{}
type RulesetConfirm struct {
	CmdMem  []string
	RuleMem []ruleFormat
}
type RulesetCancel struct {
	CmdMem  []string
	RuleMem []ruleFormat
}
type ReturnFromProfile struct{}

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
	pendingCommands []string
	pendingRules    []ruleFormat
	profile         string // selected profile path (display only)
}

type ruleFormat struct {
	Action    string
	Direction string
	Interface string
	FromIP    string
	ToIP      string
	Port      string
	Protocol  string
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
	ifaces = append(ifaces, listInterfaces()...)

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
			return m, func() tea.Msg { return RulesetCancel{CmdMem: m.pendingCommands} }
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
				c, r, err := m.collectRule()
				if err != nil {
					return newErrorBoxModel("Invalid Rule:", err.Error(), m), nil
				}
				m.pendingCommands = append(m.pendingCommands, c)
				m.pendingRules = append(m.pendingRules, *r)
				return m, func() tea.Msg { return RuleAdded{CmdMem: m.pendingCommands, RuleMem: m.pendingRules} }
			}
			if m.focusIdx == sfSubmitBtn {
				// You’ll wire the backend later; we just emit a message.
				return m, func() tea.Msg { return RulesetConfirm{CmdMem: m.pendingCommands, RuleMem: m.pendingRules} }
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

	return strings.Join([]string{
		header,
		sepStyle.Render(strings.Repeat("─", 80)),
		row,
		"",
		buttons,
		"",
		hintStyle.Render(fmt.Sprintf("Pending in memory: %d", len(m.pendingCommands))),
	}, "\n")
}

func (m *simpleRuleForm) updateFocus() {
	m.action.Focused = m.focusIdx == sfAction
	m.direction.Focused = m.focusIdx == sfDirection
	m.iface.Focused = m.focusIdx == sfInterface

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

func (m *simpleRuleForm) collectRule() (string, *ruleFormat, error) {
	dir := m.direction.Value()
	if dir == "default" {
		dir = ""
	}
	iface := m.iface.Value()
	if iface == "default" {
		iface = ""
	}

	rf := &ruleFormat{
		Action:    m.action.Value(),
		Direction: m.direction.Value(),
		Interface: m.iface.Value(),
		FromIP:    m.fromIP.Value(),
		ToIP:      m.toIP.Value(),
		Port:      m.port.Value(),
		Protocol:  m.protocol.Value(),
	}

	cmdFields := &ufw.Form{
		Action:    m.action.Value(),
		Direction: dir,
		Interface: iface,
		FromIP:    m.fromIP.Value(),
		ToIP:      m.toIP.Value(),
		Port:      m.port.Value(),
		Protocol:  m.protocol.Value(),
	}
	cmd, err := cmdFields.ParseForm()
	if err != nil {
		return "", &ruleFormat{}, err
	}
	return cmd, rf, nil
}

/* ---------- Parent flow to manage the two screens ---------- */
type profilesFlow struct {
	child           tea.Model
	commands        []string
	rules           []ruleFormat
	selectedProfile string
}

type sealProbe struct {
	Commands []string `json:"commands"`
}

func profileIsSealed(path string) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return false, nil
	}

	var sp *sealProbe
	if err = json.Unmarshal(b, &sp); err == nil {
		if len(sp.Commands) > 0 {
			return true, nil
		}
		return false, nil
	}

	var anyData map[string]interface{}
	if err = json.Unmarshal(b, &anyData); err == nil && len(anyData) > 0 {
		return true, nil
	}
	return false, nil
}

var baseDir string

func NewProfilesFlow() *profilesFlow {
	var err error
	baseDir, err = os.UserConfigDir()
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
			m.selectedProfile = v.Path
			m.child = NewSimpleRuleForm(v.Path)
			return m, nil
		case InvalidProfile:
			m.child = newErrorBoxModel("Profile is sealed:", "The profile already contains data and is write-once. Please select an empty profile", m.child)
			return m, nil
		case RulesetCancel:
			//Clear rules in memory and go back to profile select
			v.CmdMem = nil
			v.RuleMem = nil
			return m, tea.Quit
		case RuleAdded:
			return m, nil
		case RulesetConfirm:
			m.commands = v.CmdMem
			m.rules = v.RuleMem
			cmdList := strings.Join(m.commands, "\n")

			onYes := func() tea.Msg { return RuleSubmit{} }
			m.child = newConfirmModel("Are you sure you want to add these commands/rules to the selected profile?", cmdList, m.child, onYes)
			return m, nil
		case RuleSubmit:
			var rs *RuleSet
			rs = &RuleSet{}
			rs.Commands = m.commands
			rs.Rules = m.rules
			profile := m.selectedProfile
			rs.Name = strings.Trim(profile, ".json")
			rs.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
			profilePath := baseDir + "/tufwgo/profiles/" + profile

			data, err := json.MarshalIndent(rs, "", "  ")
			if err != nil {
				m.child = newErrorBoxModel("Failed to serialize ruleset:", err.Error(), m)
				return m, nil
			}
			err = os.WriteFile(profilePath, data, 0o644)
			if err != nil {
				m.child = newErrorBoxModel("Failed to write ruleset to file:", err.Error(), m)
				return m, nil
			}

			m.child = newSuccessBoxModel(fmt.Sprintf("Successfully wrote ruleset to profile: %s", strings.Trim(profile, ".json")), fmt.Sprintf("Profile is located at: %s", profilePath), returnMsg(ProfileDone{}))
			return m, nil
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
	return "Press Esc to exit."
}

type returnMsgModel struct{ msg tea.Msg }

func (m *returnMsgModel) Init() tea.Cmd                         { return func() tea.Msg { return m.msg } }
func (m *returnMsgModel) Update(_ tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *returnMsgModel) View() string                          { return "" }
func returnMsg(msg tea.Msg) tea.Model                           { return &returnMsgModel{msg: msg} }

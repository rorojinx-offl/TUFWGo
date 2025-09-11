package tui

import tea "github.com/charmbracelet/bubbletea"

type IACPreflightOK struct{}
type IACPreflightFailed struct{ Err error }
type IACProfileChosen struct{ Path string }
type IACAction int
type IACActionChosen struct{ Action IACAction }
type AnsibleStdout struct{ line string }
type AnsibleStderr struct{ line string }
type AnsibleDone struct{ err error }
type AnsibleConfig struct {
	WorkDir        string //The working directory where the playbook and inventory are located
	Inventory      string //The inventory file
	SendPlaybook   string //The send playbook file (will be in a subpath)
	DeployPlaybook string //The deploy playbook file (will be in a subpath)
}
type CmdStruct struct {
	Name    string   //ansible binary name
	Args    []string //Arguments for the command
	WorkDir string   //Working directory for the command
}
type iacFlow struct {
	child           tea.Model
	selectedProfile string
	preflightOK     bool
	ansibleCfg      AnsibleConfig
}

const (
	ActionSend IACAction = iota
	ActionDeploy
	ActionPing
)

func NewIACFlow(cfg AnsibleConfig) *iacFlow {
	m := &iacFlow{ansibleCfg: cfg}
	m.child = NewPreflightModel()
	return m
}

func (m *iacFlow) Init() tea.Cmd { return nil }

func (m *iacFlow) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case IACPreflightOK:
		m.preflightOK = true
		m.child = NewAnsibleProfilePicker(func(path string) tea.Msg {
			return IACProfileChosen{Path: path}
		})
		return m, nil
	case IACPreflightFailed:
		m.child = newErrorBoxModel("Preflight Check Failed", v.Err.Error(), m.child)
		return m, nil
	case IACProfileChosen:
		ok, err := profileIsSealed(v.Path)
		if err != nil {
			m.child = newErrorBoxModel("Profile Check Failed", err.Error(), m.child)
			return m, nil
		}
		if !ok {
			m.child = newErrorBoxModel("Empty Profile", "The selected profile is empty and cannot be used.", m.child)
			return m, nil
		}
		m.selectedProfile = v.Path
		m.child = NewActionMenuModel()
		return m, nil
	case IACActionChosen:
		m.child = NewAnsibleRunModel(CmdStruct{})
		return m, nil
	}
	if m.child != nil {
		next, cmd := m.child.Update(msg)
		m.child = next
		return m, cmd
	}
	return m, nil
}

func (m *iacFlow) View() string {
	if m.child != nil {
		return m.child.View()
	}
	return "Profile Deployment Center. Press Esc"
}

// Stubs will be placed below to keep compile errors away for now
type preflightStub struct{}

func NewPreflightModel() tea.Model { return &preflightStub{} }
func (p *preflightStub) Init() tea.Cmd {
	return func() tea.Msg { return IACPreflightOK{} }
}
func (p *preflightStub) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return p, nil }
func (p *preflightStub) View() string                            { return "Preflight… (stub)" }

// --- Profile Picker (stub): pressing Enter picks a fake path ---
type profilePickerStub struct {
	onChoose func(path string) tea.Msg
}

func NewAnsibleProfilePicker(onChoose func(path string) tea.Msg) tea.Model {
	return &profilePickerStub{onChoose: onChoose}
}
func (p *profilePickerStub) Init() tea.Cmd { return nil }
func (p *profilePickerStub) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "enter" {
		// In the real picker you’ll return the actual selected JSON path.
		return p, func() tea.Msg { return p.onChoose("dummy.json") }
	}
	return p, nil
}
func (p *profilePickerStub) View() string {
	return "Profile Picker… (stub)\nPress Enter to choose dummy.json"
}

// --- Action Menu (stub): 1/2/3 select actions ---
type actionMenuStub struct{}

func NewActionMenuModel() tea.Model     { return &actionMenuStub{} }
func (a *actionMenuStub) Init() tea.Cmd { return nil }
func (a *actionMenuStub) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "1":
			return a, func() tea.Msg { return IACActionChosen{Action: ActionSend} }
		case "2":
			return a, func() tea.Msg { return IACActionChosen{Action: ActionDeploy} }
		case "3":
			return a, func() tea.Msg { return IACActionChosen{Action: ActionPing} }
		}
	}
	return a, nil
}
func (a *actionMenuStub) View() string {
	return "Action Menu… (stub)\n[1] Send Profile\n[2] Send + Deploy\n[3] Test Connection"
}

// --- Runner (stub): shows placeholder and returns on Esc ---
type runnerStub struct{}

func NewAnsibleRunModel(_ CmdStruct) tea.Model { return &runnerStub{} }
func (r *runnerStub) Init() tea.Cmd            { return nil }
func (r *runnerStub) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "esc" {
		// In the real app you’ll return to the action menu. For now, noop.
	}
	return r, nil
}
func (r *runnerStub) View() string {
	return "Runner… (stub)\n(Streaming will appear here.)  Press Esc to return."
}

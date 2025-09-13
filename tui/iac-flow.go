package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type IACPreflightOK struct{}
type IACPreflightFailed struct{ Err error }

type IACProfileChosen struct{ File string } // file name (not full path), like your profile flow

type IACAction int

const (
	ActionSend IACAction = iota
	ActionSendAndDeploy
	ActionPing
)

type IACActionChosen struct{ Action IACAction }

type IACRunStart struct{ Plan *CmdPlan }
type IACRunDone struct {
	Out string
	Err error
}
type IACReturnToAction struct{}

type AnsibleConfig struct {
	WorkDir        string // e.g. ~/.config/tufwgo-infra
	Inventory      string // e.g. ~/.config/tufwgo-infra/inventory/hosts.ini
	SendPlaybook   string // e.g. ~/.config/tufwgo-infra/playbooks/send_profile.yml
	DeployPlaybook string // e.g. ~/.config/tufwgo-infra/playbooks/deploy_profile.yml
}

type CmdPlan struct {
	Name    string
	Args    []string
	WorkDir string
}

type iacFlow struct {
	child           tea.Model
	cfg             AnsibleConfig
	selectedProfile string // file name only (same pattern as add-to-profile)
}

func NewIACFlow(cfg *AnsibleConfig) *iacFlow {
	return &iacFlow{
		cfg:   *cfg,
		child: newPreflightView(),
	}
}

func (m *iacFlow) Init() tea.Cmd { return runPreflightCmd(&m.cfg) }

func (m *iacFlow) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.child != nil {
		switch v := msg.(type) {
		case IACPreflightOK:
			// Move to "select a NON-empty profile" (sealed==true).
			onChoose := func(file string) tea.Msg { return IACProfileChosen{File: file} }
			dir, err := getConfigDir()
			if err != nil {
				m.child = newErrorBoxModel("Config Error", fmt.Sprintf("Failed to get config dir: %v", err), m.child)
				return m, nil
			}
			m.child = NewDeployProfileSelect(filepath.Join(dir, "tufwgo", "profiles"), onChoose)
			return m, nil

		case IACPreflightFailed:
			m.child = newErrorBoxModel("Preflight Check Failed", v.Err.Error(), m.child)
			return m, nil

		case IACProfileChosen:
			m.selectedProfile = v.File
			m.child = NewIACActionMenu()
			return m, nil

		case IACActionChosen:
			cfgDir, _ := getConfigDir()
			fullProfile := filepath.Join(cfgDir, "tufwgo", "profiles", m.selectedProfile)
			plan := buildAnsiblePlan(&m.cfg, fullProfile, v.Action)

			summary := fmt.Sprintf("%s\n%s",
				lipgloss.NewStyle().Bold(true).Render("Will run:"),
				lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("%s %s\ncwd: %s", plan.Name, strings.Join(plan.Args, " "), plan.WorkDir)),
			)
			onYes := func() tea.Msg { return IACRunStart{Plan: plan} }
			m.child = newConfirmModel("Proceed with Ansible task?", summary, m.child, onYes)
			return m, nil

		case IACRunStart:
			runner := NewIACRunner()
			m.child = runner
			return m, tea.Batch(startAnsibleProc(v.Plan, runner.channel), listenAnsible(runner.channel))

		case IACRunDone:
			if v.Err != nil {
				m.child = newErrorBoxModel("Ansible Failed", v.Err.Error()+"\n\n"+v.Out, m.child)
				return m, nil
			}
			// Show output then jump back to the Action menu
			m.child = newSuccessBoxModel("Ansible task completed.", v.Out, returnMsg(IACReturnToAction{}))
			return m, nil

		case IACReturnToAction:
			m.child = NewIACActionMenu()
			return m, nil
		}

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
	return "Profile Deployment Center"
}

func runPreflightCmd(cfg *AnsibleConfig) tea.Cmd {
	cfgDir, err := getConfigDir()
	if err != nil {
		return func() tea.Msg { return IACPreflightFailed{Err: fmt.Errorf("failed to get config dir: %v", err)} }
	}
	binPath := filepath.Join(cfgDir, "tufwgo", "pdc", "tufwgo-deploy")
	baseCfgPath := filepath.Join(cfgDir, "tufwgo")

	return func() tea.Msg {
		if _, err = os.Stat(baseCfgPath); err != nil {
			return IACPreflightFailed{Err: fmt.Errorf("base config path not found: %s", baseCfgPath)}
		}
		if _, err = exec.LookPath("ansible"); err != nil {
			return IACPreflightFailed{Err: fmt.Errorf("missing binary: ansible")}
		}
		if _, err = exec.LookPath("ansible-playbook"); err != nil {
			return IACPreflightFailed{Err: fmt.Errorf("missing binary: ansible-playbook")}
		}
		for _, p := range []string{cfg.WorkDir, cfg.Inventory, cfg.SendPlaybook, cfg.DeployPlaybook} {
			if _, err = os.Stat(p); err != nil {
				return IACPreflightFailed{Err: fmt.Errorf("required path not found: %s", p)}
			}
		}
		if _, err = os.Stat(binPath); err != nil {
			return IACPreflightFailed{Err: fmt.Errorf("deployment binary not found: %s", binPath)}
		}
		return IACPreflightOK{}
	}
}

func buildAnsiblePlan(cfg *AnsibleConfig, profilePath string, a IACAction) *CmdPlan {
	cfgDir, _ := getConfigDir()
	switch a {
	case ActionSend:
		return &CmdPlan{"ansible-playbook",
			[]string{cfg.SendPlaybook, "-i", cfg.Inventory, "-e", fmt.Sprintf("profile_src=%s dest_dir=%s", profilePath, filepath.Dir(profilePath))},
			cfg.WorkDir,
		}
	case ActionSendAndDeploy:
		return &CmdPlan{"ansible-playbook",
			[]string{cfg.DeployPlaybook, "-i", cfg.Inventory, "-e", fmt.Sprintf("profile_src=%s dest_dir=%s helper_src=%s", profilePath, filepath.Dir(profilePath), filepath.Join(cfgDir, "tufwgo", "pdc", "tufwgo-deploy"))},
			cfg.WorkDir,
		}
	case ActionPing:
		return &CmdPlan{"ansible",
			[]string{"all", "-i", cfg.Inventory, "-m", "ping"},
			cfg.WorkDir,
		}
	}
	return &CmdPlan{}
}

type IACStdout struct{ Line string }
type IACStderr struct{ Line string }
type iacRunner struct {
	channel chan tea.Msg
	lines   []string
}

func NewIACRunner() *iacRunner {
	return &iacRunner{channel: make(chan tea.Msg, 256)}
}

func (r *iacRunner) Init() tea.Cmd { return nil }

func (r *iacRunner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case IACStdout:
		r.lines = append(r.lines, "[OUT] "+v.Line)
		return r, listenAnsible(r.channel)
	case IACStderr:
		r.lines = append(r.lines, "[ERR] "+v.Line)
		return r, listenAnsible(r.channel)
	}
	return r, nil
}

func (r *iacRunner) View() string {
	return lipgloss.NewStyle().Bold(true).Render("Ansible Output\n") +
		strings.Repeat("-", 60) + "\n" + strings.Join(r.lines, "\n") + "\n\n" +
		lipgloss.NewStyle().Faint(true).Render("Press Esc to exit")
}

func startAnsibleProc(plan *CmdPlan, ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command(plan.Name, plan.Args...)
		cmd.Dir = plan.WorkDir

		stdout, err1 := cmd.StdoutPipe()
		stderr, err2 := cmd.StderrPipe()
		if err1 != nil || err2 != nil {
			return IACRunDone{Err: fmt.Errorf("pipe error: %v %v", err1, err2)}
		}
		if err := cmd.Start(); err != nil {
			return IACRunDone{Err: err}
		}

		go scanToMsgs(stdout, func(s string) { ch <- IACStdout{Line: s} })
		go scanToMsgs(stderr, func(s string) { ch <- IACStderr{Line: s} })

		go func() {
			err := cmd.Wait()
			ch <- IACRunDone{Err: err}
			close(ch)
		}()
		return nil
	}
}

func listenAnsible(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		if msg, ok := <-ch; ok {
			return msg
		}
		return IACRunDone{Err: nil}
	}
}

func scanToMsgs(r io.Reader, emit func(string)) {
	s := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	s.Buffer(buf, 1024*1024)
	for s.Scan() {
		emit(s.Text())
	}
}

type preflightView struct{}

func newPreflightView() tea.Model                                { return &preflightView{} }
func (p *preflightView) Init() tea.Cmd                           { return nil }
func (p *preflightView) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return p, nil }
func (p *preflightView) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("Running preflight checks…")
	body := []string{
		"• ansible binary",
		"• ansible-playbook binary",
		"• inventory file",
		"• send_profile playbook",
		"• deploy_profile playbook",
		"• deployment binary",
	}
	return strings.Join([]string{title, strings.Repeat("─", 60), strings.Join(body, "\n")}, "\n")
}

type deployProfileSelectModel struct {
	title    string
	dd       dropdown
	files    []string
	onChoose func(file string) tea.Msg // returns filename (not path), to mirror your flow
	err      string
}

func NewDeployProfileSelect(base string, onChoose func(file string) tea.Msg) tea.Model {
	files := listJSONProfiles(base) // you already have this helper
	dd := newDropdown("Choose a ruleset to deploy", files)
	return &deployProfileSelectModel{
		title:    "Select Non-Empty Ruleset",
		dd:       dd,
		files:    files,
		onChoose: onChoose,
	}
}

func (m *deployProfileSelectModel) Init() tea.Cmd { return nil }

func (m *deployProfileSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
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

			dir, err := getConfigDir()
			if err != nil {
				m.err = fmt.Sprintf("Failed to get config dir: %v", err)
				return m, nil
			}
			// Require SEALED (non-empty)
			full := filepath.Join(dir, "tufwgo", "profiles", chosen)
			ok, err := profileIsSealed(full) // same helper you already use
			if err != nil {
				m.err = fmt.Sprintf("Failed to check profile: %v", err)
				return m, nil
			}
			if !ok {
				m.err = "Profile is empty. Choose a non-empty ruleset."
				return m, nil
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
	m.dd.Update(msg)
	return m, nil
}

func (m *deployProfileSelectModel) View() string {
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

type iacActionMenu struct {
	items []string
	idx   int
}

func NewIACActionMenu() tea.Model {
	return &iacActionMenu{
		items: []string{"Send Profile", "Send + Deploy", "Test Ansible Connection"},
	}
}
func (m *iacActionMenu) Init() tea.Cmd { return nil }

func (m *iacActionMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "up":
			if m.idx > 0 {
				m.idx--
			}
		case "down":
			if m.idx < len(m.items)-1 {
				m.idx++
			}
		case "enter":
			var act IACAction
			switch m.idx {
			case 0:
				act = ActionSend
			case 1:
				act = ActionSendAndDeploy
			case 2:
				act = ActionPing
			}
			return m, func() tea.Msg { return IACActionChosen{Action: act} }
		case "esc", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *iacActionMenu) View() string {
	var rows []string
	for i, it := range m.items {
		line := it
		if i == m.idx {
			line = focusStyle.Render("> " + line)
		} else {
			line = "  " + line
		}
		rows = append(rows, line)
	}
	return strings.Join([]string{
		focusStyle.Render("Choose Action"),
		sepStyle.Render(strings.Repeat("─", 60)),
		strings.Join(rows, "\n"),
		"",
		hintStyle.Render("↑/↓ to move • Enter to select • Esc to exit"),
	}, "\n")
}

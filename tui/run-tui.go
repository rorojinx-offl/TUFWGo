package tui

import (
	"TUFWGo/audit"
	"TUFWGo/system/local"
	"TUFWGo/system/ssh"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func RunTUI() {
	tabs := []string{"General", "IPv6 Rules", "Profile Management", "Settings"}
	var withSSH []string
	if ssh.GetSSHStatus() {
		withSSH = []string{"List Current Rules", "Add Rule", "Remove Rule", "Test SSH Connection", "Fail2Ban Dashboard (Coming Soon!)"}
	} else {
		withSSH = []string{"List Current Rules", "Add Rule", "Remove Rule", "Fail2Ban Dashboard (Coming Soon!)"}
	}

	tabContent := []*Model{
		{Items: withSSH},
		{Items: []string{"Adjust your preferences here.", "Change settings as needed.", "Customize your experience.", "Save your changes."}},
		{Items: []string{"Create Profile", "Add to Profile", "Import a Profile", "Examine Profiles", "Profile Deployment Center"}},
		{Items: []string{"Find answers to common questions.", "Contact support if needed.", "Explore tutorials and guides.", "Get the most out of the app."}},
	}

	m := &TabModel{
		Tabs:       tabs,
		TabContent: tabContent,
	}

	auditor, err := audit.OpenDailyAuditLog()
	if err != nil {
		fmt.Println(err)
		return
	}
	m.SetAuditor(auditor, getActor())
	audit.SetGlobalAuditor(auditor, getActor())

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}

func getActor() string {
	if ssh.GetSSHStatus() {
		if err := ssh.Checkup(); err != nil {
			return "Unknown"
		}
		remActor, err := ssh.CommandStream("echo \"$(whoami)@$(hostname)\"")
		if err != nil {
			return "Unknown"
		}
		return remActor
	}
	locActor, err := local.RunCommand("echo \"$(whoami)@$(hostname)\"")
	if err != nil {
		return "Unknown"
	}
	return locActor
}

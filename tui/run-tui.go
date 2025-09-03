package tui

import (
	"TUFWGo/system/ssh"

	tea "github.com/charmbracelet/bubbletea"
)

//var SSHActive = false

func RunTUI() {
	tabs := []string{"General", "IPv6 Mode", "Profile Management", "Settings"}
	var withSSH []string
	if ssh.GetSSHStatus() {
		withSSH = []string{"List Current Rules", "Add Rule", "Remove Rule", "Test SSH Connection", "Fail2Ban Dashboard (Coming Soon!)"}
	} else {
		withSSH = []string{"List Current Rules", "Add Rule", "Remove Rule", "Fail2Ban Dashboard (Coming Soon!)"}
	}

	tabContent := []*Model{
		{Items: withSSH},
		{Items: []string{"Adjust your preferences here.", "Change settings as needed.", "Customize your experience.", "Save your changes."}},
		{Items: []string{"View and edit your profile information.", "Manage your account details.", "Update your password.", "Set your privacy options."}},
		{Items: []string{"Find answers to common questions.", "Contact support if needed.", "Explore tutorials and guides.", "Get the most out of the app."}},
	}

	m := &TabModel{
		Tabs:       tabs,
		TabContent: tabContent,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}

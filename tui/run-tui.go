package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func RunTUI() {
	tabs := []string{"General", "IPv6 Mode", "Profile Management", "Settings"}
	tabContent := []*Model{
		{Items: []string{"List Current Rules", "Add Rule", "Remove Rule", "Toggle Rules"}},
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

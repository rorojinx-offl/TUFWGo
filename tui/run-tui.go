package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func RunTUI() {
	tabs := []string{"Home", "Settings", "Profile", "Help"}
	tabContent := []*Model{
		{Items: []string{"Welcome to the Home tab!", "Here is some introductory content."}},
		{Items: []string{"Adjust your preferences here.", "Change settings as needed."}},
		{Items: []string{"View and edit your profile information.", "Manage your account details."}},
		{Items: []string{"Find answers to common questions.", "Contact support if needed."}},
	}

	m := &TabModel{
		Tabs:       tabs,
		TabContent: tabContent,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}

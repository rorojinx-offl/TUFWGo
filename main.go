package main

import (
	"TUFWGo/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	tabs := []string{"Home", "Settings", "Profile", "Help"}
	tabContent := []*tui.Model{
		{Items: []string{"Welcome to the Home tab!", "Here is some introductory content."}},
		{Items: []string{"Adjust your preferences here.", "Change settings as needed."}},
		{Items: []string{"View and edit your profile information.", "Manage your account details."}},
		{Items: []string{"Find answers to common questions.", "Contact support if needed."}},
	}

	m := &tui.TabModel{
		Tabs:       tabs,
		TabContent: tabContent,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}

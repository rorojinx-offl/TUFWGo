package main

import (
	"TUFWGo/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	tabs := []string{"Home", "Settings", "Profile", "Help"}
	tabContent := []string{"Welcome to the Home tab!", "Adjust your settings here.", "This is your profile.", "Here is some help information."}

	m := &tui.TabModel{
		Tabs:       tabs,
		TabContent: tabContent,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}

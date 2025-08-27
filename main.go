package main

import (
	"TUFWGo/ufw"
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	/*tabs := []string{"Home", "Settings", "Profile", "Help"}
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
	}*/

	form := ufw.Form{}
	reader := bufio.NewReader(os.Stdin)

	form.Action = readRequired(reader, "Action: ")
	form.Direction = readLine(reader, "Direction: ")
	form.Interface = readLine(reader, "Interface: ")
	form.FromIP = readLine(reader, "From IP: ")
	form.ToIP = readLine(reader, "To IP: ")
	form.Port = readLine(reader, "Port: ")
	form.Protocol = readLine(reader, "Protocol: ")
	form.AppProfile = readLine(reader, "App Profile: ")

	finalCommand, err := form.ParseForm()
	if err != nil {
		panic(err)
	}
	fmt.Println(finalCommand)
}

func readLine(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func readRequired(reader *bufio.Reader, prompt string) string {
	for {
		val := readLine(reader, prompt)
		if val != "" {
			return val
		}
		fmt.Println("This field is required. Please enter a value: ")
	}
}

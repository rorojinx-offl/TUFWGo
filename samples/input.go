package samples

import (
	"TUFWGo/system/local"
	"TUFWGo/system/ssh"
	"TUFWGo/ufw"
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func InputUFW() {
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

	ask := readRequired(reader, fmt.Sprintf("The command to be executed is: %s\nDo you want to continue (y/n)? ", finalCommand))
	if ask != "y" {
		return
	}

	_, err = local.RunCommand(finalCommand)
}

func InputSSH() {
	reader := bufio.NewReader(os.Stdin)

	host := readRequired(reader, "Host: ")
	portStr := readLine(reader, "Port: ")
	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		panic(err)
	}
	user := readRequired(reader, "User: ")

	ssh.Connect(host, user, int(port))
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

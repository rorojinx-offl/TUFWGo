package ssh

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

func InputSSH() (*ssh.Client, error) {
	reader := bufio.NewReader(os.Stdin)

	host := readRequired(reader, "Host: ")
	portStr := readLine(reader, "Port: ")
	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		panic(err)
	}
	user := readRequired(reader, "User: ")

	return Connect(host, user, int(port))
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

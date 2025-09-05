package alert

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
var from string

const path = ".config/tufwgo/emails.txt"

func loadEmails() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("unable to determine user home directory: %w", err)
	}
	properPath := filepath.Join(home, path)

	if _, err = os.Stat(properPath); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("email list not found at: %w", err)
	}
	file, err := os.Open(properPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open file: %w", err)
	}
	defer file.Close()

	seen := make(map[string]struct{})
	out := make([]string, 0, 64)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "from:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				from = strings.TrimSpace(parts[1])
				fmt.Println(from)
			} else {
				fmt.Println("Invalid from")
			}
			continue
		}
		low := strings.ToLower(line)
		if !emailRegex.MatchString(low) {
			log.Printf("WARNING: Skipping invalid email: %s", line)
			continue
		}

		if _, ok := seen[low]; ok {
			continue
		}
		seen[low] = struct{}{}
		out = append(out, low)
	}
	return out, nil
}

func batches[T any](in []T, n int) [][]T {
	if n <= 0 {
		return [][]T{in}
	}
	var out [][]T
	for i := 0; i < len(in); i += n {
		j := i + n
		if j > len(in) {
			j = len(in)
		}
		out = append(out, in[i:j])
	}
	return out
}

func SendMail() {
	apiKey := os.Getenv("SENDGRID_API_KEY")
	if apiKey == "" {
		fmt.Println("WARNING: SendGrid API key not set")
		return
	}
	client := sendgrid.NewSendClient(apiKey)

	recips, err := loadEmails()
	if err != nil {
		fmt.Println("WARNING: Unable to load email list:", err)
		return
	}
	if len(recips) == 0 {
		fmt.Println("No email recipients found, skipping email alert")
		return
	}

	from := mail.NewEmail("TUFWGo Alert Manager", "alerts@em2695.tufwgo.store")
	subject := "[TUFWGo] Rule Added - Allow TCP 22 from 192.168.1.1"
	plainTextContent := `Hello,
	An action was performed on your firewall via TUFWGo.
	📌 Action: Rule Added
	📌 Status: Success
	📌 Timestamp: 2024-10-05 14:30:00
	📌 Executed By: root
	📌 Hostname: raspberrypi
	📌 Rule Details:
		- Action: Allow
		- Direction: Inbound
		- From: 192.168.1.1
		- To: Any
		- Port: 22
		- Protocol: TCP
		- App Profile: N/A

	🏷️ Command Executed:
		ufw allow from 192.168.1.5 to any port 22 proto tcp

	TUFWGo Alert Manager
	`

	const batchSize = 500
	for _, batch := range batches(recips, batchSize) {
		msg := mail.NewV3Mail()
		msg.SetFrom(from)
		msg.Subject = subject

		p := mail.NewPersonalization()
		for _, r := range batch {
			p.AddTos(mail.NewEmail("", r))
		}

		msg.AddPersonalizations(p)
		msg.AddContent(mail.NewContent("text/plain", plainTextContent))
		msg.Categories = append(msg.Categories, "ufw-alert")

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		response, clientErr := client.SendWithContext(ctx, msg)
		if clientErr != nil {
			fmt.Printf("send failed: %v\n", clientErr)
			return
		}
		if response.StatusCode >= 400 {
			fmt.Printf("send failed: status %d, body: %s\n", response.StatusCode, response.Body)
			return
		}
	}
}

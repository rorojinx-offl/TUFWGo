package alert

import (
	"TUFWGo/ufw"
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
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
			var tempFrom string
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				tempFrom = strings.TrimSpace(parts[1])
			} else {
				fmt.Println("Invalid from")
				continue
			}

			lowFrom := strings.ToLower(tempFrom)
			if !emailRegex.MatchString(lowFrom) {
				fmt.Printf("WARNING: Skipping invalid from email: %s\n", tempFrom)
				continue
			}
			from = lowFrom
			continue
		}
		low := strings.ToLower(line)
		if !emailRegex.MatchString(low) {
			fmt.Printf("WARNING: Skipping invalid email: %s", line)
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

func (e *EmailInfo) SendMail(action, cmd string, rule *ufw.Form) {
	e.prepareEmailInfo(action, cmd, rule)

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

	//fromTemp := mail.NewEmail("TUFWGo Alert Manager", "alerts@em2695.tufwgo.store")
	sender := mail.NewEmail("TUFWGo Alert Manager", from)
	subject := "[TUFWGo] Rule Added - Allow TCP 22 from 192.168.1.1"
	plainTextContent := e.prepareMessage()

	const batchSize = 500
	for _, batch := range batches(recips, batchSize) {
		msg := mail.NewV3Mail()
		msg.SetFrom(sender)
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
		fmt.Printf("Email batch sent successfully to %d recipients, status code: %d\n", len(batch), response.StatusCode)
	}
}

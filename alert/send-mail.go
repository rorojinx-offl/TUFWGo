package alert

import (
	"TUFWGo/audit"
	"TUFWGo/system/local"
	"TUFWGo/system/ssh"
	"TUFWGo/ufw"
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mailersend/mailersend-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
var from string

const path = ".config/tufwgo/emails.txt"

func loadEmails() ([]string, error) {
	home := local.GlobalUserHomeDir
	properPath := filepath.Join(home, path)

	if _, err := os.Stat(properPath); errors.Is(err, os.ErrNotExist) {
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

func mailersendCompatRecips(recips []string) []mailersend.Recipient {
	var msRecips []mailersend.Recipient

	for _, recip := range recips {
		msRecips = append(msRecips, mailersend.Recipient{Name: "TUFWGo Admin", Email: recip})
	}
	return msRecips
}

func addAudit(action, result, comment, errMsg string) {
	auditor, actor := audit.GetGlobalAuditor()
	if auditor == nil {
		return
	}

	entry := &audit.Entry{
		Actor:   actor,
		Action:  action,
		Command: comment,
		Result:  result,
		Error:   errMsg,
	}

	_ = auditor.Append(entry)
}

func (e *EmailInfo) SendMail(action, cmd string, rule *ufw.Form) {
	e.prepareEmailInfo(action, cmd, rule)

	apiKey := os.Getenv("MAILERSEND_API_KEY")
	if apiKey == "" {
		fmt.Println("$MAILERSEND_API_KEY environment variable not set")
		addAudit("email.api", "error", "", "WARNING: MailerSend API key not set")
		return
	}
	client := mailersend.NewMailersend(apiKey)

	recips, err := loadEmails()
	if err != nil {
		fmt.Println(err)
		addAudit("email.recipients", "error", "WARNING: Unable to load email list:", err.Error())
		return
	}
	if len(recips) == 0 {
		fmt.Println("WARNING: No emails found")
		addAudit("email.recipients", "warning", "No email recipients found, skipping email alert", "")
		return
	}

	var remote string
	if ssh.GetSSHStatus() {
		remote = "over SSH"
	} else {
		remote = ""
	}

	msRecips := mailersendCompatRecips(recips)

	sender := mailersend.From{
		Name:  "TUFWGo Alert Manager",
		Email: from,
	}
	subject := fmt.Sprintf("[TUFWGo] %s %s - %s", action, remote, cmd)
	plainTextContent := e.prepareMessage()

	const batchSize = 500
	for _, batch := range batches(msRecips, batchSize) {
		for _, r := range batch {
			msg := client.Email.NewMessage()
			msg.SetFrom(sender)
			msg.Subject = subject

			p := mail.NewPersonalization()
			p.AddTos(mail.NewEmail(r.Name, r.Email))

			var rcp []mailersend.Recipient
			rcp = append(rcp, r)

			msg.SetText(plainTextContent)
			msg.SetRecipients(rcp)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			response, clientErr := client.Email.Send(ctx, msg)
			if clientErr != nil {
				fmt.Println(clientErr)
				addAudit("email.send", "error", "Send Failed", clientErr.Error())
			}
			if response.StatusCode >= 400 {
				fmt.Printf("status %d, body: %s\n", response.StatusCode, response.Body)
				addAudit("email.send", "error", "Send Failed", fmt.Sprintf("status %d, body: %s", response.StatusCode, response.Body))
			}

			fmt.Printf("Email sent successfully to recipient %s\n", r.Email)
			addAudit("email.send", "success", "Send Succeeded", fmt.Sprintf("recipient %s, status: %d", r.Email, response.StatusCode))
		}
	}
}

package copilot

import (
	"TUFWGo/ufw"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type modelConstructor struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stop    []string       `json:"stop,omitempty"`
	Options map[string]any `json:"options,omitempty"`
	Stream  bool           `json:"stream,omitempty"`
}

type modelResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
	Done     bool   `json:"done"`
}

func buildPrompt(userPrompt string) string {
	system := "You are a UFW assistant. ONLY output a block between <BEGIN-UFW> and <END-UFW>. Each rule MUST be on exactly one line, starting with \"rule:\" and continuing with \"key=value;\" pairs, all separated by semicolons (NOT colons). DO NOT put each field on its own line. Follow the following format: rule: action=<allow|deny|reject|limit>; dir=< |in|out>; iface=<string or empty>; from=< |any|ip/cidr>; to=< |any|ip/cidr>; port=< |single|pipe-separated|range>; proto=< |tcp|udp|tcp/udp|udp/tcp|all|esp|ah|gre|icmp|ipv6>; app=<string or empty>. No prose, no code fences, no markdown. If the request is out of scope, return with an error. Do not output 'ufw ...' commands. Bad example (DO NOT DO): ufw allow in on ALL:1024/tcp FROM 10.0.0.5. Good example: <BEGIN-UFW> rule: action=allow; dir=in; iface=eth0; from=10.0.0.5; to=any; port=22; proto=tcp; app=<END-UFW>"

	return system + "\n\nUSER:\n" + userPrompt + "\nASSISTANT:\n"
}

func generateStructure(baseURL, model, user string, timeout time.Duration) ([]byte, error) {
	construct := modelConstructor{
		Model:  model,
		Prompt: buildPrompt(user),
		Stop:   []string{"<END-UFW>"},
		Options: map[string]any{
			"temperature": 0.0,
			"num_ctx":     2048,
			"num_predict": 512,
		},
		Stream: false,
	}
	b, _ := json.Marshal(construct)
	client := &http.Client{Timeout: timeout}
	resp, err := client.Post(baseURL+"/api/generate", "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		all, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to generate: %s", string(all))
	}

	decode := json.NewDecoder(resp.Body)
	var out strings.Builder
	for {
		var mr modelResponse
		if err = decode.Decode(&mr); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode error: %w", err)
		}
		if mr.Error != "" {
			return nil, fmt.Errorf("ollama error: %s", mr.Error)
		}
		out.WriteString(mr.Response)
		if mr.Done {
			break
		}
	}

	fmt.Printf("MODEL TEXT:\n%s\n", out.String())

	return []byte(out.String()), nil
}

func CompileDSLToUFW(modelText string) ([]string, error) {
	ruleBlock, err := extractBetween(modelText, "<BEGIN-UFW>", "<END-UFW>")
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(ruleBlock))
	scanner.Split(bufio.ScanLines)

	ruleLine := regexp.MustCompile(`^\s*rule:\s*(.+)$`)
	keyVal := regexp.MustCompile(`\s*([a-z_]+)\s*=\s*(.*?)\s*$`)

	var commands []string
	lineNum := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lineNum++

		m := ruleLine.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("line %d: does not start with 'rule:'", lineNum)
		}

		fields := map[string]string{
			"action": "",
			"dir":    "",
			"iface":  "",
			"from":   "",
			"to":     "",
			"port":   "",
			"proto":  "",
			"app":    "",
		}

		for _, part := range strings.Split(m[1], ";") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			keyMatch := keyVal.FindStringSubmatch(part)
			if keyMatch == nil {
				return nil, fmt.Errorf("line %d: invalid key-value pair '%s'", lineNum, part)
			}
			fields[keyMatch[1]] = keyMatch[2]
		}

		form := &ufw.Form{
			Action:     fields["action"],
			Direction:  fields["dir"],
			Interface:  fields["iface"],
			FromIP:     fields["from"],
			ToIP:       fields["to"],
			Port:       fields["port"],
			Protocol:   fields["proto"],
			AppProfile: fields["app"],
		}

		cmd, err := form.ParseForm()
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		commands = append(commands, cmd)
	}
	if len(commands) == 0 {
		return nil, fmt.Errorf("no rules found")
	}
	return commands, nil
}

func extractBetween(body, start, end string) (string, error) {
	i := strings.Index(body, start)
	if i < 0 {
		return "", fmt.Errorf("start pointer not found")
	}

	j := strings.Index(body[i+len(start):], end)
	if j < 0 {
		return "", fmt.Errorf("end pointer not found")
	}

	return body[i+len(start) : i+len(start)+j], nil
}

package copilot

import (
	"TUFWGo/ufw"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type modelConstructor struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Format  any            `json:"format,omitempty"`
	Options map[string]any `json:"options,omitempty"`
	Stream  bool           `json:"stream,omitempty"`
}

type modelResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

type rule struct {
	Action     string `json:"action"`
	Direction  string `json:"direction"`
	Interface  string `json:"interface"`
	From       string `json:"from"`
	To         string `json:"to"`
	Port       string `json:"port"`
	Protocol   string `json:"protocol"`
	AppProfile string `json:"app_profile"`
}

type payload struct {
	Intent string `json:"intent"`
	Rules  []rule `json:"rules"`
}

func buildPrompt(userPrompt string) string {
	system := "You are a UFW assistant. ONLY output valid JSON that matches the schema below. No prose, no code fences, no markdown. If the request is out of scope, return with: {\"intent\":\"reject\",\"rules\":[],\"defaults\":{},\"reason\":\"why\"}. Schema: {\"intent\": \"rule_add\",\"rules\": [{\"action\": \"allow | deny | reject | limit\",\"direction\": \" | in | out\",\"interface\": \"<string or empty>\",\"from\": \" | any | <ip/cidr>\",\"to\": \" | any | <ip/cidr>\",\"port\": \" | <single>|<pipe-separated>|<range>\",\"protocol\": \" | tcp | udp | tcp/udp | udp/tcp | all | esp | ah | gre | icmp | ipv6\",\"app_profile\": \" | <string or empty>\"}]}"

	return system + "\n\nUSER:\n" + userPrompt + "\nASSISTANT:\n"
}

var ufwSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"intent": map[string]any{
			"type": "string",
			"enum": []string{"rule_add"},
		},
		"rules": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action":      map[string]any{"type": "string", "enum": []string{"allow", "deny", "reject", "limit"}},
					"direction":   map[string]any{"type": "string"},
					"interface":   map[string]any{"type": "string"},
					"from":        map[string]any{"type": "string"},
					"to":          map[string]any{"type": "string"},
					"port":        map[string]any{"type": "string"},
					"protocol":    map[string]any{"type": "string", "enum": []string{"", "tcp", "udp", "tcp/udp", "udp/tcp", "all", "esp", "ah", "gre", "icmp", "ipv6"}},
					"app_profile": map[string]any{"type": "string"},
				},
				"required":             []string{"action", "direction", "from", "to", "port", "protocol"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"intent", "rules"},
	"additionalProperties": false,
}

func generateStructure(baseURL, model, user string, timeout time.Duration) ([]byte, error) {
	construct := modelConstructor{
		Model:  model,
		Prompt: buildPrompt(user),
		Format: ufwSchema,
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

	all, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to generate: %s", string(all))
	}

	var mr modelResponse
	dec := json.NewDecoder(bytes.NewReader(all))
	dec.UseNumber()
	if err = dec.Decode(&mr); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}
	if mr.Error != "" {
		return nil, fmt.Errorf("ollama/model error: %s", mr.Error)
	}
	if len(mr.Response) == 0 {
		return nil, fmt.Errorf("no empty response: %s", mr.Response)
	}

	rb := []byte(mr.Response)
	if len(rb) > 0 {
		switch rb[0] {
		case '"':
			unq, err := strconv.Unquote(string(rb))
			if err != nil {
				return nil, fmt.Errorf("invalid response: %w", err)
			}
			return []byte(unq), nil
		case '{', '[':
			return rb, nil
		default:
			return nil, fmt.Errorf("invalid response: %s", string(rb))
		}
	}
	return nil, fmt.Errorf("invalid response: %s", string(rb))
}

func compileJSONToUFW(raw []byte) ([]string, error) {
	s := strings.TrimSpace(string(raw))
	if strings.HasPrefix(s, "```") {
		s = stripCodeFence(s)
	}

	if len(s) > 0 && s[0] == '"' {
		if unq, err := strconv.Unquote(s); err == nil {
			s = unq
		}
	}

	fmt.Printf("RAW MODEL RESPONSE: \n%s\n", s)

	var p payload
	dec := json.NewDecoder(strings.NewReader(s))
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("model did not return valid JSON: %w", err)
	}
	if p.Intent == "" {
		return nil, fmt.Errorf("intent not found: %q", p.Intent)
	}
	if p.Intent != "rule_add" {
		return nil, fmt.Errorf("unsupported intent: %q", p.Intent)
	}
	if len(p.Rules) == 0 {
		return nil, fmt.Errorf("no rules found")
	}

	cmds := make([]string, 0, len(p.Rules))
	for i, r := range p.Rules {
		if r.AppProfile != "" && !(r.Action == "allow" || r.Action == "deny") {
			return nil, fmt.Errorf("rule %d: app profile requires action 'allow' or 'deny' only", i)
		}

		format := &ufw.Form{
			Action:     r.Action,
			Direction:  r.Direction,
			Interface:  r.Interface,
			FromIP:     r.From,
			ToIP:       r.To,
			Port:       r.Port,
			Protocol:   r.Protocol,
			AppProfile: r.AppProfile,
		}

		cmd, err := format.ParseForm()
		if err != nil {
			return nil, fmt.Errorf("rule %d: %w", i, err)
		}
		cmds = append(cmds, cmd)
	}
	return cmds, nil
}

func stripCodeFence(s string) string {
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```JSON")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

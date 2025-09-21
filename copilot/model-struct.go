package copilot

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

func buildPrompt(userPrompt string) string {
	system := "You are a UFW assistant. ONLY output valid JSON that matches the schema below. No prose, no code fences, no markdown. If the request is out of scope, return with: {\"intent\":\"reject\",\"rules\":[],\"defaults\":{},\"reason\":\"why\"}. Schema: {\"intent\": \"rule_add\",\"rules\": [{\"action\": \"allow | deny | reject | limit\",\"direction\": \" | in | out\",\"interface\": \"<string or empty>\",\"from\": \" | any | <ip/cidr>\",\"to\": \" | any | <ip/cidr>\",\"port\": \" | <single>|<pipe-separated>|<range>\",\"protocol\": \" | tcp | udp | tcp/udp | udp/tcp | all | esp | ah | gre | icmp | ipv6\",\"app_profile\": \" | <string or empty>\"}]}"

	return system + "\n\nUSER:\n" + userPrompt + "\nASSISTANT:\n"
}

var ufwSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"intent": map[string]any{
			"type": "string",
			"enum": "rule_add",
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
					"protocol":    map[string]any{"type": "string", "enum": []string{"tcp", "udp", "tcp/udp", "udp/tcp", "all", "esp", "ah", "gre", "icmp", "ipv6"}},
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

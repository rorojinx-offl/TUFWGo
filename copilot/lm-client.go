package copilot

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-skynet/go-llama.cpp"
)

type LM struct {
	model *llama.LLama
}
type LMOptions struct {
	ModelPath string
	Threads   int
	Grammar   string
}

func NewLM(opts *LMOptions) (*LM, error) {
	params := []llama.ModelOption{llama.SetContext(2048)}
	model, err := llama.New(opts.ModelPath, params...)
	if err != nil {
		return nil, fmt.Errorf("error loading language model: %s", err)
	}

	return &LM{model: model}, nil
}

func (l *LM) Close() { l.model.Free() }

func buildPrompt(user string) string {
	system := `You are a UFW assistant. ONLY output valid JSON that matches the schema below. No prose, no code fences, no markdown. If the request is out of scope, return with:
	{"intent":"reject","rules":[],"defaults":{},"reason":"why"}.

	Schema:
	{
		"intent": "rule_add",
		"rules": [
			{
				"action": "allow | deny | reject | limit",
				"direction": " | in | out",
				"interface": "<string or empty>",
				"from": " | any | <ip/cidr>",
				"to": " | any | <ip/cidr>",
				"port": " | <single>|<pipe-separated>|<range>",
				"protocol": " | tcp | udp | tcp/udp | udp/tcp | all | esp | ah | gre | icmp | ipv6"
				"app_profile": " | <string or empty>"
			}
		]
	}`

	return system + "\n\nUSER:\n" + user + "\nASSISTANT:\n"
}

func (l *LM) Call(GBNFGrammar string, userText string) ([]byte, error) {
	prompt := buildPrompt(userText)

	opts := []llama.PredictOption{
		llama.SetTemperature(0.0),
		llama.SetTopK(40),
		llama.SetTopP(0.95),
		llama.SetTokens(512),
		llama.SetStopWords([]string{}...),
		llama.WithGrammar(GBNFGrammar),
	}

	resp, err := l.model.Predict(prompt, opts...)
	if err != nil {
		return nil, fmt.Errorf("prediction error: %s", err)
	}
	return []byte(resp), nil
}

func mustReadFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func ExampleCall() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	modelPath := filepath.Join(homeDir, "gguf-models", "mistral-7b-instruct-v0.2.Q4_K_M.gguf")
	grammar := mustReadFile("copilot/rule-add-only.gbnf")
	lm, err := NewLM(&LMOptions{
		ModelPath: modelPath,
		Threads:   0, // use all available cores
		Grammar:   grammar,
	})
	if err != nil {
		return err
	}
	defer lm.Close()

	response, err := lm.Call(grammar, "Allow incoming TCP traffic on port 22 from any IP address")
	if err != nil {
		return err
	}

	fmt.Println(string(response))
	return nil
}

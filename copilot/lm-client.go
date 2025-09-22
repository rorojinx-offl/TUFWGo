package copilot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/taigrr/systemctl"
)

func checkOllamaDaemon() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	isActive, err := systemctl.IsActive(ctx, "ollama", systemctl.Options{UserMode: false})
	if err != nil {
		return err
	}
	isEnabled, err := systemctl.IsEnabled(ctx, "ollama", systemctl.Options{UserMode: false})
	if err != nil {
		return err
	}
	isFailed, err := systemctl.IsFailed(ctx, "ollama", systemctl.Options{UserMode: false})
	if err != nil {
		return err
	}

	if isActive && isEnabled && !isFailed {
		return nil
	} else {
		return errors.New("ollama is not running")
	}
}
func ollamaPull(baseURL, name string, timeout time.Duration) error {
	body := map[string]any{"name": name, "stream": false}
	b, _ := json.Marshal(body)
	request, _ := http.NewRequest("POST", baseURL+"/api/pull", bytes.NewReader(b))
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: timeout}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	all, _ := ioutil.ReadAll(response.Body)
	if response.StatusCode != 200 {
		return fmt.Errorf("failed to pull: %s", string(all))
	}
	return nil
}

func RunOllama() error {
	if err := checkOllamaDaemon(); err != nil {
		return err
	}
	if err := ollamaPull("http://localhost:11434", "rorojinx/tufwgo-slm", 5*time.Minute); err != nil {
		return err
	}

	raw, err := generateStructure("http://localhost:11434", "rorojinx/tufwgo-slm", "Allow ssh from 10.0.0.5", 30*time.Second)
	if err != nil {
		return err
	}

	cmds, err := compileJSONToUFW(raw)
	if err != nil {
		return fmt.Errorf("invalid model output: %w", err)
	}

	for _, cmd := range cmds {
		fmt.Println(cmd)
	}
	return nil
}

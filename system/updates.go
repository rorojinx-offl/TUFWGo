package system

import (
	"bytes"
	"fmt"
	"os/exec"
)

func checkUpdates() (string, error) {
	cmd := exec.Command("tufwgo-update", "-checko")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("stderr: %s", stderr.String())
	}

	if cmd.ProcessState.ExitCode() != 0 {
		return "", fmt.Errorf("failed to check for updates: %s", stderr.String())
	}

	if string(out) == "There are no updates to install!" {
		return "0", nil
	}

	return string(out), nil
}

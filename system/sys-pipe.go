package system

import (
	"bytes"
	"os/exec"
)

func prepareCommand(cmdStr string) (string, error, *bytes.Buffer) {
	cmd := exec.Command("bash", "-c", cmdStr)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return "", err, &stderr
	}
	return string(output), nil, nil
}

func RunCommand(command string) (string, error, *bytes.Buffer) {
	out, err, stderr := prepareCommand(command)
	if err != nil {
		return "", err, stderr
	}
	return out, nil, nil
}

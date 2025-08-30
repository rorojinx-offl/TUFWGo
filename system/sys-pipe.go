package system

import (
	"bytes"
	"fmt"
	"os/exec"
)

func prepareCommand(cmdStr string) (string, error, *bytes.Buffer) {
	cmd := exec.Command("bash", "-c", cmdStr)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Command failed with:", err)
		fmt.Println("stderr:", stderr.String())
		return "", nil, nil
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

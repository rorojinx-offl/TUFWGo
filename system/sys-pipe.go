package system

import (
	"bytes"
	"fmt"
	"os/exec"
)

func prepareCommand(cmdStr string) string {
	cmd := exec.Command("bash", "-c", cmdStr)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Command failed with:", err)
		fmt.Println("stderr:", stderr.String())
		return ""
	}
	return string(output)
}

func RunCommand(command string) string {
	out := prepareCommand(command)
	return out
}

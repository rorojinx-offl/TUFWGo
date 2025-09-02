package local

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func prepareCommand(cmdStr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdStr)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return "", errors.New(fmt.Sprint("stderr:", stderr.String()))
	}
	return string(output), nil
}

func prepareCommandConversation(cmdStr, input string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdStr)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	err := cmd.Run()
	if err != nil {
		return "", errors.New(fmt.Sprint("stderr:", stderr.String()))
	}
	return stdout.String(), nil
}

func RunCommand(command string) (string, error) {
	out, err := prepareCommand(command)
	if err != nil {
		return "", err
	}
	return out, nil
}

func CommandConversation(command, reply string) (string, error) {
	out, err := prepareCommandConversation(command, reply)
	if err != nil {
		return "", err
	}
	return out, nil
}

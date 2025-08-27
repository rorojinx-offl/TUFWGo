package system

import (
	"fmt"
	"os/exec"
)

func prepareCommand(cmdStr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func RunCommand(command string) {
	out, err := prepareCommand(command)
	if err != nil {
		panic(err)
	}
	fmt.Println(out)
}

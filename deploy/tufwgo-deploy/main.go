package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type ruleSet struct {
	Name      string       `json:"name"`
	CreatedAt string       `json:"created_at"`
	Commands  []string     `json:"commands"`
	Rules     []ruleFormat `json:"rules"`
}

type ruleFormat struct {
	Action    string
	Direction string
	Interface string
	FromIP    string
	ToIP      string
	Port      string
	Protocol  string
}

func main() {
	path := flag.String("profile", "", "Path to file to read rules")
	flag.Parse()
	if *path == "" {
		fmt.Fprintln(os.Stderr, "missing required flag: --profile [path]")
		os.Exit(2)
	}

	_, _, _, cmds, err := showRulesFromProfile(*path)
	if err != nil {
		fmt.Println("Error reading profile:", err)
		os.Exit(1)
	}
	if err = executeProfile(cmds); err != nil {
		fmt.Println("Error executing profile commands:", err)
		os.Exit(1)
	}
}

func showRulesFromProfile(path string) (string, string, string, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", "", nil, err
	}

	var rs ruleSet
	if err = json.Unmarshal(data, &rs); err != nil {
		return "", "", "", nil, err
	}

	rawCommands := rs.Commands
	commands := strings.Join(rs.Commands, "\n")
	return rs.Name, rs.CreatedAt, commands, rawCommands, nil
}

func executeProfile(commands []string) error {
	for _, cmd := range commands {
		_, err := runCommand(cmd)
		if err != nil {
			return fmt.Errorf("failed to execute command %q: %w", cmd, err)
		}
	}
	return nil
}

func getConfigDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return cfg, nil
}

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

func runCommand(command string) (string, error) {
	out, err := prepareCommand(command)
	if err != nil {
		return "", err
	}
	return out, nil
}

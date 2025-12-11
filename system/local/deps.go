package local

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func FindDependencies() ([]string, error) {
	var dependencies []string

	if _, err := exec.LookPath("ufw"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			dependencies = append(dependencies, "ufw")
		} else {
			return nil, fmt.Errorf("unable to find ufw: %w", err)
		}
	}

	if _, err := exec.LookPath("ansible"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			dependencies = append(dependencies, "ansible")
		} else {
			return nil, fmt.Errorf("unable to find ansible: %w", err)
		}
	}

	if _, err := exec.LookPath("ollama"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			dependencies = append(dependencies, "ollama")
		} else {
			return nil, fmt.Errorf("unable to find ollama: %w", err)
		}
	}

	if _, err := exec.LookPath("sshd"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			dependencies = append(dependencies, "sshd")
		} else {
			return nil, fmt.Errorf("unable to find sshd: %w", err)
		}
	}

	return dependencies, nil
}

func DerivePkgMgrKeywords(pkgmgr string) string {
	switch pkgmgr {
	case "apt", "dnf", "zypper":
		return fmt.Sprintf("%s install -y", pkgmgr)
	case "apk":
		return fmt.Sprintf("%s add", pkgmgr)
	case "pacman":
		return fmt.Sprintf("%s -S --noconfirm", pkgmgr)
	default:
		return ""
	}
}

func InstallDependencies(deps []string, pkgMgrKwd string) []error {
	var errorsList []error
	reader := bufio.NewReader(os.Stdin)

	allConf := readLine(reader, fmt.Sprintf("You are missing %d dependencies. Would you like to install them? (y/N) ", len(deps)))
	if allConf == "N" || allConf == "n" || allConf == "" || allConf == "\n" {
		os.Exit(0)
	}

	for _, dep := range deps {
		switch dep {
		case "ollama":
			olConf := readLine(reader, "Do you want to install Ollama? (y/N) ")
			if olConf == "N" || olConf == "n" || olConf == "" || olConf == "\n" {
				continue
			}

			err := CommandLiveOutput("curl -fsSL https://ollama.com/install.sh | sh")
			if err != nil {
				errorsList = append(errorsList, fmt.Errorf("error installing ollama: %w", err))
			}
		case "ufw":
			ufwConf := readLine(reader, "Do you want to install UFW? (y/N) ")
			if ufwConf == "N" || ufwConf == "n" || ufwConf == "" || ufwConf == "\n" {
				continue
			}

			err := CommandLiveOutput(fmt.Sprintf("%s ufw", pkgMgrKwd))
			if err != nil {
				errorsList = append(errorsList, fmt.Errorf("error installing ufw: %w", err))
			}
		case "ansible":
			ansConf := readLine(reader, "Do you want to install Ansible? (y/N) ")
			if ansConf == "N" || ansConf == "n" || ansConf == "" || ansConf == "\n" {
				continue
			}

			err := CommandLiveOutput(fmt.Sprintf("%s ansible", pkgMgrKwd))
			if err != nil {
				errorsList = append(errorsList, fmt.Errorf("error installing ansible: %w", err))
			}
		case "sshd":
			sshConf := readLine(reader, "Do you want to install SSH Server? (y/N) ")
			if sshConf == "N" || sshConf == "n" || sshConf == "" || sshConf == "\n" {
				continue
			}

			err := CommandLiveOutput(fmt.Sprintf("%s sshd", pkgMgrKwd))
			if err != nil {
				errorsList = append(errorsList, fmt.Errorf("error installing sshd: %w", err))
			}
		}
	}
	return errorsList
}

func readLine(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

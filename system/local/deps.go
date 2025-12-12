package local

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
				//Fallback if apt is being used on Debian instead of Ubuntu
				isDeb, err := debianOrUbuntu()
				if err != nil {
					errorsList = append(errorsList, fmt.Errorf("error installing ansible: %w", err))
					continue
				}
				if !isDeb {
					errorsList = append(errorsList, fmt.Errorf("error installing ansible"))
					continue
				}

				if strings.Contains(pkgMgrKwd, "apt") {
					ucd, err := ansibleDebGetUbuntuCodename()
					if err != nil {
						errorsList = append(errorsList, err)
						continue
					}
					cmdList := []string{
						"wget -O- \"https://keyserver.ubuntu.com/pks/lookup?fingerprint=on&op=get&search=0x6125E2A8C77F2818FB7BD15B93C4A3FD7BB9C367\" | sudo gpg --dearmour -o /usr/share/keyrings/ansible-archive-keyring.gpg",
						fmt.Sprintf("echo \"deb [signed-by=/usr/share/keyrings/ansible-archive-keyring.gpg] http://ppa.launchpad.net/ansible/ansible/ubuntu %s main\" | sudo tee /etc/apt/sources.list.d/ansible.list", ucd),
						fmt.Sprintf("apt update -y && apt install ansible -y"),
					}

					for _, cmd := range cmdList {
						err = CommandLiveOutput(cmd)
						if err != nil {
							errorsList = append(errorsList, err)
						}
					}
					continue
				}
				errorsList = append(errorsList, fmt.Errorf("error installing ansible: %w", err))
			}
		case "sshd":
			sshConf := readLine(reader, "Do you want to install SSH Server? (y/N) ")
			if sshConf == "N" || sshConf == "n" || sshConf == "" || sshConf == "\n" {
				continue
			}

			pkgName := specificSSH(pkgMgrKwd)

			err := CommandLiveOutput(fmt.Sprintf("%s %s", pkgMgrKwd, pkgName))
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

func debianOrUbuntu() (bool, error) {
	distro, err := RunCommand("grep -Ei '^id(_like)?=' /etc/os-release | grep -qi 'ubuntu' && echo ubuntu || (grep -Ei '^id(_like)?=' /etc/os-release | grep -qi 'debian' && echo debian || true)")
	if err != nil {
		return false, fmt.Errorf("error deriving distro: %w", err)
	}
	switch distro {
	case "debian":
		return true, nil
	case "ubuntu":
		return false, nil
	default:
		return false, fmt.Errorf("unknown os")
	}
}

func specificSSH(pkgMgrKwd string) string {
	if strings.Contains(pkgMgrKwd, "apt") || strings.Contains(pkgMgrKwd, "dnf") || strings.Contains(pkgMgrKwd, "apk") {
		return "openssh-server"
	} else {
		return "openssh"
	}
}

func ansibleDebGetUbuntuCodename() (string, error) {
	ver, err := RunCommand("cat /etc/debian_version")
	if err != nil {
		return "", fmt.Errorf("unable to get debian version: %w", err)
	}

	verD, err := strconv.ParseFloat(ver, 64)
	if err != nil {
		return "", fmt.Errorf("unable to parse debian version: %w", err)
	}

	verI := int(verD)

	switch verI {
	case 12:
		return "jammy", nil
	case 11:
		return "focal", nil
	case 10:
		return "bionic", nil
	default:
		return "", errors.New("unsupported debian version")
	}
}

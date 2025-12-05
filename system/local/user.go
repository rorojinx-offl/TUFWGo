package local

import (
	"errors"
	"fmt"
	"os"
	"os/user"
)

var GlobalUserHomeDir string
var GlobalUserCfgDir string

func RequireRoot() {
	if os.Geteuid() != 0 {
		fmt.Println("This command requires root/sudo privileges! (try: sudo " + os.Args[0] + ")")
		os.Exit(77)
	}

	home, cfg, err := getNonRootUserHome()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	GlobalUserHomeDir = home
	GlobalUserCfgDir = cfg
}

func getNonRootUserHome() (string, string, error) {
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser == "" {
		return "", "", errors.New("unable to derive $SUDO_USER")
	}

	usrlkp, err := user.Lookup(sudoUser)
	if err != nil {
		return "", "", fmt.Errorf("unable to lookup evoking user %s: %v", sudoUser, err)
	}

	return usrlkp.HomeDir, fmt.Sprintf("%s/.config", usrlkp.HomeDir), nil
}

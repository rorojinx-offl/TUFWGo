package main

import (
	"TUFWGo/alert"
	"TUFWGo/auth"
	"TUFWGo/system/local"
	"TUFWGo/system/ssh"
	"TUFWGo/tui"
	"flag"
	"fmt"
)

var skipTermCheck = flag.Bool("skip-term-check", false, "Skip the terminal size check")
var sshMode = flag.Bool("ssh", false, "Run in SSH mode")

func main() {
	local.RequireRoot()
	//runTUIMode()
	//samples.Input()
	//tui.RunForm()
	//alert.SendSampleMail()
	alert.TestEmailData()
}

func runTUIMode() {
	flag.Parse()
	if *sshMode {
		if !*skipTermCheck && !local.TermCheck() {
			return
		}
		client, err := ssh.InputSSH()
		if err != nil {
			fmt.Println("SSH Connection Failed:", err)
			return
		}

		label, err := local.RunCommand("uname -snrm")
		if err != nil {
			_ = fmt.Errorf("unable to get system name to generate controller ID: %w", err)
			return
		}
		clientID, pubB64, priv, created, err := auth.EnsureControllerKey(label)
		if err != nil {
			fmt.Println("Failed to load or create controller key:", err)
			return
		}
		if created {
			fmt.Println("Controller ID:", clientID)
			fmt.Println("Public Key:", pubB64)
			out, err := ssh.CommandStream(fmt.Sprintf("%s add-controller --pub %q --label %q", "/usr/bin/tufwgo-auth", pubB64, label))
			if err != nil {
				fmt.Println("Failed to add new controller to allowlist:", err, "\n", out)
				return
			}
			fmt.Println("New controller key created.")
		}
		err = auth.AuthenticateOverSSH(client, clientID, "1.0", "tufwgo-auth", priv)
		if err != nil {
			fmt.Println("Authentication Failed:", err)
			return
		}

		ssh.SetSSHStatus(true)
		tui.RunTUI()
		defer client.Close()
		return
	}
	if !*skipTermCheck && !local.TermCheck() {
		return
	}
	tui.RunTUI()
}

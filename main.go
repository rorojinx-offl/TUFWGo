package main

import (
	"TUFWGo/system/local"
	"TUFWGo/system/ssh"
	"TUFWGo/tui"
	"flag"
	"fmt"
)

var skipTermCheck = flag.Bool("skip-term-check", false, "Skip the terminal size check")
var sshMode = flag.Bool("ssh", false, "Run in SSH mode")

func main() {
	runTUIMode()
	//samples.Input()
	//tui.RunForm()
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

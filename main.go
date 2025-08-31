package main

import (
	"TUFWGo/system"
	"TUFWGo/tui"
	"flag"
	"fmt"
)

var skipTermCheck = flag.Bool("skip-term-check", false, "Skip the terminal size check")
var sshMode = flag.Bool("ssh", false, "Run in SSH mode")

func main() {
	runTUIMode()
	//system.Input()
	//tui.RunForm()
}

func runTUIMode() {
	flag.Parse()
	if *skipTermCheck {
		tui.RunTUI()
		return
	} else if *sshMode {
		if !system.TermCheck() {
			return
		}
		fmt.Println("TEST: SSH MODE")
		return
	}

	if !system.TermCheck() {
		return
	}
	tui.RunTUI()
}

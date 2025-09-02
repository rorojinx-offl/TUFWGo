package main

import (
	"TUFWGo/samples"
	"TUFWGo/system/local"
	"TUFWGo/tui"
	"flag"
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
		samples.InputSSH()
		return
	}
	if !*skipTermCheck && !local.TermCheck() {
		return
	}
	tui.RunTUI()
}

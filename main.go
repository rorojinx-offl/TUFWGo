package main

import (
	"TUFWGo/system"
	"TUFWGo/tui"
	"flag"
)

var skipTermCheck = flag.Bool("skip-term-check", false, "Skip the terminal size check")

func main() {
	//runTUIMode()
	//system.Input()
	tui.RunForm()
}

func runTUIMode() {
	flag.Parse()
	if *skipTermCheck {
		tui.RunTUI()
		return
	}

	if !system.TermCheck() {
		return
	}
	tui.RunTUI()
}

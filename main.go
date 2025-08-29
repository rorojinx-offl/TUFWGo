package main

import (
	"TUFWGo/tui"
	"fmt"
	"os"

	"github.com/charmbracelet/x/term"
)

func main() {
	if !termCheck() {
		return
	}
	tui.RunTUI()
	//system.Input()
}

// Enforce terminal check at the very beginning and ensure all users maximise their terminal window before using the TUI
func termCheck() bool {
	w, h, err := getTermSize()
	if err != nil {
		fmt.Println("Unable to get terminal size:", err)
		return false
	}
	safeW, safeH := 236, 55
	if w < safeW || h < safeH {
		fmt.Printf("Your terminal size is too small (%dx%d). Please maximise terminal window!\n", w, h)
		return false
	}
	return true
}

func getTermSize() (width, height int, err error) {
	return term.GetSize(uintptr(int(os.Stdout.Fd())))
}

package main

import (
	"TUFWGo/system"
	"TUFWGo/system/local"
)

func main() {
	local.RequireRoot()
	system.InitSetup()
	/*err := copilot.RunOllama()
	if err != nil {
		fmt.Println("Failed to run copilot:", err)
		return
	}*/
}

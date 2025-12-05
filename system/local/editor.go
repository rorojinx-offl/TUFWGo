package local

import (
	"errors"
	"fmt"
	"os/exec"
)

func newEditor(file string) error {
	editor, err := findEditor()
	if err != nil {
		return fmt.Errorf("failed to find editor: %v", err)
	}
	err = CommandLiveOutput(fmt.Sprintf("%s %s", editor, file))
	if err != nil {
		return fmt.Errorf("failed to launch editor: %v", err)
	}
	return nil
}

func findEditor() (string, error) {
	possibleEditors := []string{"vim", "nano", "nvim", "vi", "emacs"}
	for _, editor := range possibleEditors {
		if _, err := exec.LookPath(editor); err == nil {
			return editor, nil
		}
	}
	return "", errors.New("please install a terminal text editor")
}

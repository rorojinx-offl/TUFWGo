package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

func CommandStream(cmd string) (string, error) {
	session, err := GlobalClient.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func ConversationalCommentStream(cmdStr, input string) (string, error) {
	session, err := GlobalClient.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if input != "" {
		session.Stdin = strings.NewReader(input)
	}

	err = session.Run(cmdStr)
	if err != nil {
		return "", errors.New(fmt.Sprint("stderr:", stderr.String()))
	}
	return stdout.String(), nil
}

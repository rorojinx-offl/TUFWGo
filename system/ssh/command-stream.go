package ssh

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

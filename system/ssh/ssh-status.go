package ssh

import "errors"

var sshStatus = false

func GetSSHStatus() bool {
	return sshStatus
}

func SetSSHStatus(status bool) {
	sshStatus = status
}

func Checkup() error {
	if GlobalClient == nil {
		return errors.New("SSH Mode is not active")
	}

	_, _, err := GlobalClient.SendRequest("keepalive@openssh.com", true, nil)
	if err != nil {
		return errors.New("SSH Connection Failed")
	}
	return nil
}

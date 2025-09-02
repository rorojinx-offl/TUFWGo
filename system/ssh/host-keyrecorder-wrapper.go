package ssh

import (
	"golang.org/x/crypto/ssh"
	"net"
)

type hostKeyRecorder struct {
	inner   ssh.HostKeyCallback
	lastKey ssh.PublicKey
}

func (r *hostKeyRecorder) callback(hostname string, remote net.Addr, key ssh.PublicKey) error {
	r.lastKey = key
	return r.inner(hostname, remote, key)
}

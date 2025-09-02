package ssh

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"os"
	"path/filepath"
)

func findKnownHostsPath() string {
	homePath, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(os.Stderr, "cannot resolve home:", err)
		os.Exit(1)
	}
	return filepath.Join(homePath, ".ssh", "known_hosts")
}

func ensureKnownHostsExists(path string) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		return file.Close()
	}
	return err
}

func hostPattern(host string, port int) string {
	if port == 22 {
		return host
	}
	return fmt.Sprintf("[%s]:%d", host, port)
}

func appendKnownHostLine(path, pattern string, key ssh.PublicKey) error {
	line := knownhosts.Line([]string{pattern}, key)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(line + "\n")
	return err
}

func fingerprintSHA256(pub ssh.PublicKey) string {
	sum := sha256.Sum256(pub.Marshal())
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:])
}

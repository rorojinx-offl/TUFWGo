package ssh

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"net"
	"os"
	"strings"
	"time"
)

func Connect(host, user string, port int) {
	khPath := findKnownHostsPath()
	client, err := connectWithKnownHosts(
		context.Background(),
		host, port, user,
		khPath,
		time.Second*7,  //TCP connect timeout
		time.Second*12, //SSH handshake timeout
		func() (string, error) {
			fmt.Print("SSH password: ")
			pwd, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			return strings.TrimSpace(pwd), nil
		},
	)
	if err != nil {
		panic(fmt.Sprint("Connection error: ", err))
	}
	if client == nil {
		panic("No SSH client returned")
	}
	defer client.Close()
	fmt.Println("SSH connection succeeded")
}

func connectWithKnownHosts(
	ctx context.Context,
	host string,
	port int,
	user string,
	knownHostsPath string,
	connectTimeout time.Duration,
	handshakeTimeout time.Duration,
	getPassword func() (string, error),
) (*ssh.Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)

	err := ensureKnownHostsExists(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("cannot ensure known_hosts file exists: %w", err)
	}

	baseHK, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("cannot create knownhosts callback: %w", err)
	}

	rec := &hostKeyRecorder{inner: baseHK}
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PasswordCallback(getPassword)},
		HostKeyCallback: rec.callback,
		Timeout:         handshakeTimeout,
	}

	client, err := dialSSH(ctx, addr, cfg, connectTimeout)
	if err == nil {
		return client, nil
	}

	//Need to configure host key verification through 2 channels -> unknown vs changed
	var keyErr *knownhosts.KeyError
	if errors.As(err, &keyErr) {
		if len(keyErr.Want) == 0 {
			presented := rec.lastKey
			if presented == nil {
				return nil, fmt.Errorf("unknown host, but no presented key captured: %w", err)
			}
			fp := fingerprintSHA256(presented)
			fmt.Printf("The authenticity of host '%s' can't be established.\nFingerprint (SHA256): %s\nTrust and add to known_hosts? [y/N]: ", host, fp)
			consent, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			if strings.ToLower(strings.TrimSpace(consent)) != "y" {
				return nil, fmt.Errorf("user declined to trust unknown host %s", addr)
			}

			err = appendKnownHostLine(knownHostsPath, hostPattern(host, port), presented)
			if err != nil {
				return nil, fmt.Errorf("appending known_hosts entry: %w", err)
			}

			return dialSSH(ctx, addr, cfg, connectTimeout)
		}
		expectedFPs := []string{}
		for _, want := range keyErr.Want {
			expectedFPs = append(expectedFPs, fingerprintSHA256(want.Key))
		}
		presented := rec.lastKey
		presentedFP := "(unknown)"
		if presented != nil {
			presentedFP = fingerprintSHA256(presented)
		}
		return nil, fmt.Errorf("host key mismatch for %s (possible MITM attack)!\n expected: %s\n presented: %s", addr, strings.Join(expectedFPs, ","), presentedFP)

	}
	return nil, err
}

func dialSSH(ctx context.Context, addr string, cfg *ssh.ClientConfig, connectionTimeout time.Duration) (*ssh.Client, error) {
	dialer := net.Dialer{Timeout: connectionTimeout, KeepAlive: time.Second * 30}
	connection, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	conn, channels, requests, err := ssh.NewClientConn(connection, addr, cfg)
	if err != nil {
		connection.Close()
		return nil, err
	}
	return ssh.NewClient(conn, channels, requests), nil
}

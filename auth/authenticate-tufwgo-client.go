package auth

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

type hello struct {
	Type, ClientID, ClientVersion, Algo string
}
type challenge struct {
	Type, HostID, NonceB64 string
}
type proof struct {
	Type, ClientID, SigB64 string
	TSUnix                 int64
}
type ok struct{ Type string }
type er struct {
	Type, Reason string
}

// AuthenticateOverSSH runs the Ed25519 handshake with the remote helper.
// controllerID: your short ID string (must exist in remote allowlist)
// controllerPriv: your Ed25519 private key (64 bytes)
// remoteCmd: executable on remote (e.g. "tufw-authd")
func AuthenticateOverSSH(client *ssh.Client, controllerID, clientVersion, remoteCmd string, controllerPriv ed25519.PrivateKey) error {
	sess, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	stdin, err := sess.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		return err
	}

	if err = sess.Start(remoteCmd); err != nil {
		return fmt.Errorf("start remote auth helper: %w", err)
	}

	enc := json.NewEncoder(stdin)
	dec := json.NewDecoder(bufio.NewReader(stdout))

	// 1) HELLO
	h := hello{
		Type:          "HELLO",
		ClientID:      controllerID,
		ClientVersion: clientVersion,
		Algo:          "ed25519",
	}
	if err = enc.Encode(h); err != nil {
		return fmt.Errorf("send HELLO: %w", err)
	}

	// 2) CHALLENGE
	var chal challenge
	if err = dec.Decode(&chal); err != nil {
		return fmt.Errorf("read CHALLENGE: %w", err)
	}
	if chal.Type != "CHALLENGE" || chal.NonceB64 == "" {
		return errors.New("invalid CHALLENGE")
	}
	nonce, err := base64.StdEncoding.DecodeString(chal.NonceB64)
	if err != nil {
		return fmt.Errorf("nonce decode: %w", err)
	}
	// optional: mix in a bit of client-side randomness (not required)
	_, _ = rand.Read(make([]byte, 1))

	// 3) PROOF (sign M)
	now := time.Now().Unix()
	M := buildMsg(nonce, chal.HostID, controllerID, now)
	sig := ed25519.Sign(controllerPriv, M)

	p := proof{
		Type:     "PROOF",
		ClientID: controllerID,
		TSUnix:   now,
		SigB64:   base64.StdEncoding.EncodeToString(sig),
	}
	if err = enc.Encode(p); err != nil {
		return fmt.Errorf("send PROOF: %w", err)
	}

	// 4) Receive final
	// The helper returns either OK or ERR; decode into a generic map first.
	var raw map[string]any
	if err = dec.Decode(&raw); err != nil {
		return fmt.Errorf("read final: %w", err)
	}
	if t, _ := raw["type"].(string); t == "OK" {
		// wait for remote to exit cleanly
		_ = sess.Wait()
		return nil
	}
	// try parse reason
	var e er
	b, _ := json.Marshal(raw)
	_ = json.Unmarshal(b, &e)
	_ = sess.Wait()
	if e.Reason == "" {
		e.Reason = "unknown error"
	}
	return fmt.Errorf("remote auth failed: %s", e.Reason)
}

func buildMsg(nonce []byte, hostID, clientID string, ts int64) []byte {
	msg := make([]byte, 0, len("TUFWGO-AUTH\x00")+len(nonce)+len(hostID)+len(clientID)+8)
	msg = append(msg, []byte("TUFWGO-AUTH\x00")...)
	msg = append(msg, nonce...)
	msg = append(msg, []byte(hostID)...)
	msg = append(msg, []byte(clientID)...)
	var tsb [8]byte
	binary.BigEndian.PutUint64(tsb[:], uint64(ts))
	msg = append(msg, tsb[:]...)
	return msg
}

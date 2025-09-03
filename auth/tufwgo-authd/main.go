package main

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	protoTag      = "TUFWGO-AUTH\x00"
	allowListPath = ".config/tufwgo/authorised_controllers.json"
	clockSkew     = 120 * time.Second
)

type Hello struct {
	Type          string `json:"type"`
	ClientID      string `json:"client_id"`
	ClientVersion string `json:"client_version"`
	Algo          string `json:"algo"`
}

type Challenge struct {
	Type        string `json:"type"`
	HostID      string `json:"host_id"`
	NonceBase64 string `json:"nonce_base64"`
}

type Proof struct {
	Type      string `json:"type"`
	ClientID  string `json:"client_id"`
	TSUnix    int64  `json:"ts_unix"`
	Nonce     string `json:"nonce"`
	SigBase64 string `json:"sig_base64"`
}

type OK struct {
	Type string `json:"type"`
}

type ERR struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

type allowListFile struct {
	Version     int          `json:"version"`
	Controllers []allowEntry `json:"controllers"`
}

type allowEntry struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	PubKeyB64 string `json:"pubkey_b64"`
	Revoked   bool   `json:"revoked"`
}

func main() {
	err := run(os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(ERR{Type: "ERR", Reason: err.Error()})
		os.Exit(1)
	}
}

func run(in io.Reader, out io.Writer, _ io.Writer) error {
	decode := json.NewDecoder(bufio.NewReader(in))
	encode := json.NewEncoder(out)

	var hello Hello
	err := decode.Decode(&hello)
	if err != nil {
		return fmt.Errorf("cannot decode hello: %w", err)
	}

	if hello.Type != "HELLO" || strings.ToLower(hello.Algo) != "ed25519" || hello.ClientID == "" {
		return errors.New("invalid hello")
	}

	pubkey, err := lookUpControllerPubKey(hello.ClientID)
	if err != nil {
		return fmt.Errorf("cannot look up controller pubkey: %w", err)
	}

	hostID := getHostID()
	nonce := make([]byte, 32)
	if _, err = rand.Read(nonce); err != nil {
		return fmt.Errorf("cannot generate nonce: %w", err)
	}

	challenge := Challenge{
		Type:        "CHALLENGE",
		HostID:      hostID,
		NonceBase64: base64.StdEncoding.EncodeToString(nonce),
	}
	err = encode.Encode(challenge)
	if err != nil {
		return fmt.Errorf("cannot encode challenge: %w", err)
	}

	var proof Proof
	err = decode.Decode(&proof)
	if err != nil {
		return fmt.Errorf("cannot decode proof: %w", err)
	}
	if proof.Type != "PROOF" || proof.ClientID != hello.ClientID {
		return errors.New("invalid proof")
	}

	sig, err := base64.StdEncoding.DecodeString(proof.SigBase64)
	if err != nil {
		return fmt.Errorf("cannot decode signature: %w", err)
	}

	M := buildMessage(nonce, hostID, hello.ClientID, proof.TSUnix)
	if !ed25519.Verify(pubkey, M, sig) {
		return errors.New("invalid signature")
	}

	now := time.Now()
	ts := time.Unix(proof.TSUnix, 0)
	if ts.Before(now.Add(-clockSkew)) || ts.After(now.Add(clockSkew)) {
		return errors.New("invalid timestamp")
	}

	return encode.Encode(OK{Type: "OK"})
}

func buildMessage(nonce []byte, hostID, clientID string, ts int64) []byte {
	hostIDByte := []byte(hostID)
	clientIDByte := []byte(clientID)

	buf := make([]byte, 0, len(protoTag)+len(nonce)+len(hostIDByte)+len(clientIDByte)+8)
	buf = append(buf, []byte(protoTag)...)
	buf = append(buf, nonce...)
	buf = append(buf, hostIDByte...)
	buf = append(buf, clientIDByte...)

	var tsbuf [8]byte
	binary.BigEndian.PutUint64(tsbuf[:], uint64(ts))
	buf = append(buf, tsbuf[:]...)
	return buf
}

func lookUpControllerPubKey(clientID string) (ed25519.PublicKey, error) {
	path := assertUserFilepath(allowListPath)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open allow list file: %w", err)
	}
	defer file.Close()

	var alf allowListFile
	err = json.NewDecoder(file).Decode(&alf)
	if err != nil {
		return nil, fmt.Errorf("cannot decode allow list file: %w", err)
	}

	for _, e := range alf.Controllers {
		if e.Revoked {
			continue
		}
		if e.ID == clientID {
			return parseEd25519PubKey(e.PubKeyB64)
		}
	}
	return nil, fmt.Errorf("client id not found: %s", clientID)
}

func parseEd25519PubKey(s string) (ed25519.PublicKey, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "ed25519:") {
		s = strings.TrimPrefix(s, "ed25519:")
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("bad public key length: %d", len(raw))
	}
	return raw, nil
}

func assertUserFilepath(path string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, path)
}

func getHostID() string {
	if id, err := os.ReadFile("/etc/machine-id"); err == nil {
		return strings.TrimSpace(string(id))
	}
	hostname, _ := os.Hostname()
	sum := sha256.Sum256([]byte(hostname))
	return "host-" + base64.RawStdEncoding.EncodeToString(sum[:8])
}

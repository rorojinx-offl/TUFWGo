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
	"flag"
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
	Created   string `json:"created,omitempty"`
	LastUsed  string `json:"last_used,omitempty"`
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "add-controller":
			fs := flag.NewFlagSet("add-controller", flag.ExitOnError)
			pub := fs.String("pub", "", "controller public key (ed25519:BASE64 or BASE64)")
			lbl := fs.String("label", "", "label for this controller")
			_ = fs.Parse(os.Args[2:])
			if err := addControllerCmd(*pub, *lbl); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			return
		case "list-controllers":
			if err := listControllersCmd(); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			return
		case "revoke-controller":
			fs := flag.NewFlagSet("revoke-controller", flag.ExitOnError)
			id := fs.String("id", "", "controller id to revoke")
			_ = fs.Parse(os.Args[2:])
			if err := revokeControllerCmd(*id); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			return
		}
	}

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

// Below will be helper functions to write to the allow list file
const allowlistRelPath = ".config/tufwgo/authorised_controllers.json"

func allowlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, allowlistRelPath)
}

func loadAllowlist() (*allowListFile, error) {
	p := allowlistPath()
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return &allowListFile{Version: 1}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open allowlist: %w", err)
	}
	defer f.Close()
	var af allowListFile
	if err := json.NewDecoder(f).Decode(&af); err != nil {
		return nil, fmt.Errorf("parse allowlist: %w", err)
	}
	if af.Version == 0 {
		af.Version = 1
	}
	return &af, nil
}

func saveAllowlist(af *allowListFile) error {
	p := allowlistRelPath
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	tmp := p + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open tmp: %w", err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(af); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("encode: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, p)
}

// Normalise a controller public key string. Accepts "ed25519:BASE64" or bare BASE64.
func normalizePubB64(s string) (string, []byte, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(strings.ToLower(s), "ed25519:") {
		s = s[len("ed25519:"):]
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", nil, fmt.Errorf("bad base64: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return "", nil, fmt.Errorf("bad pubkey length: got %d, want %d", len(raw), ed25519.PublicKeySize)
	}
	return "ed25519:" + base64.StdEncoding.EncodeToString(raw), raw, nil
}

func shortIDFromPub(raw []byte) string {
	sum := sha256.Sum256(raw)
	// first 8 bytes, base64url(without padding) or raw is fine; stay consistent with your client
	return "ed25519:" + base64.RawStdEncoding.EncodeToString(sum[:8])
}

func upsertController(af *allowListFile, id, label, pubB64 string) {
	now := time.Now().UTC().Format(time.RFC3339)
	// update if id exists
	for i := range af.Controllers {
		if af.Controllers[i].ID == id {
			af.Controllers[i].Label = label
			af.Controllers[i].PubKeyB64 = pubB64
			af.Controllers[i].Revoked = false
			if af.Controllers[i].Created == "" {
				af.Controllers[i].Created = now
			}
			return
		}
	}
	af.Controllers = append(af.Controllers, allowEntry{
		ID:        id,
		Label:     label,
		PubKeyB64: pubB64,
		Revoked:   false,
		Created:   now,
	})
}

func listControllersCmd() error {
	af, err := loadAllowlist()
	if err != nil {
		return err
	}
	if len(af.Controllers) == 0 {
		fmt.Println("(no controllers)")
		return nil
	}
	for _, c := range af.Controllers {
		state := "active"
		if c.Revoked {
			state = "revoked"
		}
		fmt.Printf("%s  %-8s  %s\n", c.ID, state, c.Label)
	}
	return nil
}

func addControllerCmd(pubArg, label string) error {
	if pubArg == "" {
		return errors.New("missing --pub")
	}
	pubB64, raw, err := normalizePubB64(pubArg)
	if err != nil {
		return err
	}
	id := shortIDFromPub(raw)

	af, err := loadAllowlist()
	if err != nil {
		return err
	}
	upsertController(af, id, label, pubB64)
	if err := saveAllowlist(af); err != nil {
		return err
	}
	fmt.Println("Added/updated controller:")
	fmt.Println("  ID:  ", id)
	fmt.Println("  Label:", label)
	return nil
}

func revokeControllerCmd(id string) error {
	if id == "" {
		return errors.New("missing --id")
	}
	af, err := loadAllowlist()
	if err != nil {
		return err
	}
	found := false
	for i := range af.Controllers {
		if af.Controllers[i].ID == id {
			af.Controllers[i].Revoked = true
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("id not found: %s", id)
	}
	return saveAllowlist(af)
}

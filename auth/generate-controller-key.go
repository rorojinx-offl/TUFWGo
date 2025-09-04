package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const controllerKeyPath = ".config/tufwgo/controller.key"

func EnsureControllerKey(label string) (string, string, ed25519.PrivateKey, bool, error) {
	path := assertUserKeyPath(controllerKeyPath)

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		clientID, pubkeyB64, priv, genErr := generateControllerID(label)
		if genErr != nil {
			return "", "", nil, false, fmt.Errorf("unable to generate controller ID: %w", genErr)
		}
		return clientID, pubkeyB64, priv, true, nil
	} else if err != nil {
		return "", "", nil, false, fmt.Errorf("unable to find controller key file: %w", err)
	}

	priv, err := loadControllerPrivKey()
	if err != nil {
		return "", "", nil, false, fmt.Errorf("unable to load controller key file: %w", err)
	}

	clientID, pubkeyB64, _, err := deriveIDsFromPrivKey(priv)
	if err != nil {
		return "", "", nil, false, fmt.Errorf("unable to derive IDs from private key: %w", err)
	}

	_ = os.Chmod(path, 0600)
	return clientID, pubkeyB64, priv, false, nil
}

func deriveIDsFromPrivKey(priv ed25519.PrivateKey) (clientID, pubKeyB64 string, pub ed25519.PublicKey, err error) {
	switch n := len(priv); {
	case n == ed25519.PrivateKeySize:
		// ok
	case n == ed25519.SeedSize:
		priv = ed25519.NewKeyFromSeed(priv)
	default:
		return "", "", nil, fmt.Errorf("bad private key size: %d", len(priv))
	}

	pub = make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(pub, priv[ed25519.SeedSize:])

	sum := sha256.Sum256(pub)
	clientID = "ed25519:" + base64.RawStdEncoding.EncodeToString(sum[:8])
	pubKeyB64 = "ed25519:" + base64.StdEncoding.EncodeToString(pub)
	return
}

func generateControllerID(label string) (clientID string, pubKeyB64 string, priv ed25519.PrivateKey, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", nil, fmt.Errorf("unable to generate ed25519 key: %w", err)
	}

	sum := sha256.Sum256(pub)
	clientID = "ed25519:" + base64.RawStdEncoding.EncodeToString(sum[:8])

	pubKeyB64 = "ed25519:" + base64.StdEncoding.EncodeToString(pub)

	path := assertUserKeyPath(controllerKeyPath)
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", "", nil, fmt.Errorf("unable to create directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", nil, fmt.Errorf("unable to create key file: %w", err)
	}
	defer file.Close()

	raw := struct {
		Label   string `json:"label"`
		Private string `json:"private_b64"`
	}{
		Label:   label,
		Private: base64.StdEncoding.EncodeToString(priv),
	}

	if err = json.NewEncoder(file).Encode(raw); err != nil {
		return "", "", nil, fmt.Errorf("unable to write key file: %w", err)
	}
	return clientID, pubKeyB64, priv, nil
}

func loadControllerPrivKey() (ed25519.PrivateKey, error) {
	path := assertUserKeyPath(controllerKeyPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Private string `json:"private_b64"`
	}
	if err = json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	b64, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw.Private))
	if err != nil {
		return nil, err
	}

	if len(b64) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("bad private key size: %d", len(b64))
	}
	return b64, nil

	/*path := assertUserKeyPath(controllerKeyPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Private string `json:"private_b64"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	s := strings.TrimSpace(raw.Private)
	s = strings.TrimPrefix(strings.ToLower(s), "ed25519:") // tolerate prefix
	buf, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode private_b64: %w", err)
	}

	switch len(buf) {
	case ed25519.SeedSize: // 32 -> expand
		return ed25519.NewKeyFromSeed(buf), nil
	case ed25519.PrivateKeySize: // 64 -> OK
		return ed25519.PrivateKey(buf), nil
	default:
		return nil, fmt.Errorf("unexpected ed25519 private key length: got %d (want 32 or 64)", len(buf))
	}*/
}

func assertUserKeyPath(rel string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, rel)
}

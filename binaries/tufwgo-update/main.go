package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/schollz/progressbar/v3"
)

type Manifest struct {
	Version  string            `json:"version"`
	Channel  string            `json:"channel"`
	Binaries map[string]Binary `json:"binaries"`
}

type Binary struct {
	SHA256 string `json:"sha256"`
	URL    string `json:"url"`
}

func main() {
	requireRoot()

	manifest := &Manifest{}
	err := manifest.fetchManifest()
	if err != nil {
		fmt.Println(err)
		return
	}

	if err = manifest.checkMain(); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("TUFWGo Main successfully updated!")
	}

	if err = manifest.checkAuthBin(); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("TUFWGo SSH Auth Helper successfully updated!")
	}
}

func (manifest *Manifest) fetchManifest() error {
	response, err := http.Get("https://dl.tufwgo.store/manifest.json")
	if err != nil {
		return fmt.Errorf("error downloading the file: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download manifest: received status code %d", response.StatusCode)
	}

	dec := json.NewDecoder(response.Body)
	if err = dec.Decode(&manifest); err != nil {
		return fmt.Errorf("error parsing manifest: %w", err)
	}

	return nil
}

func requireRoot() {
	if os.Geteuid() != 0 {
		fmt.Println("This command requires root/sudo privileges! (try: sudo " + os.Args[0] + ")")
		os.Exit(77)
	}
}

func getHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("error opening file: %w", err)
	}

	h := sha256.New()
	if _, err = io.Copy(h, file); err != nil {
		return "", fmt.Errorf("error getting file hash: %s", err)
	}
	_ = file.Close()
	gotHash := hex.EncodeToString(h.Sum(nil))
	return gotHash, nil
}

func downloadFile(dest, expectedSHA256 string) error {
	response, err := http.Get("https://dl.tufwgo.store/binaries/tufwgo")
	if err != nil {
		return fmt.Errorf("error downloading file: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: received status code %d", response.StatusCode)
	}

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}

	var bar *progressbar.ProgressBar
	if response.ContentLength <= 0 {
		bar = progressbar.NewOptions64(-1, progressbar.OptionSetDescription("downloading"),
			progressbar.OptionShowBytes(true),
			progressbar.OptionClearOnFinish())
	} else {
		bar = progressbar.DefaultBytes(response.ContentLength, "downloading")
	}

	h := sha256.New()
	mw := io.MultiWriter(destFile, h, bar)
	if _, err = io.Copy(mw, response.Body); err != nil {
		return fmt.Errorf("error saving file: %w", err)
	}
	_ = destFile.Close()
	_ = bar.Finish()

	gotHash := hex.EncodeToString(h.Sum(nil))
	if expectedSHA256 != "" && gotHash != expectedSHA256 {
		if err = os.Remove(dest); err != nil {
			return fmt.Errorf("error removing file: %w", err)
		}
		return fmt.Errorf("expected SHA256: %s, got SHA256: %s", expectedSHA256, gotHash)
	}

	if err = os.Chmod(dest, 0755); err != nil {
		return fmt.Errorf("error setting file permissions: %w", err)
	}
	return nil
}

func getVersion() (string, error) {
	cmd := exec.Command("bash", "-c", "tufwgo -version")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return "", errors.New(fmt.Sprint("stderr:", stderr.String()))
	}

	out := string(output)

	var list []string
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		list = append(list, scanner.Text())
	}

	return list[0], nil
}

func (manifest *Manifest) checkMain() error {
	path := "/usr/bin/tufwgo"
	expectedSHA256 := manifest.Binaries["tufwgo"].SHA256

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("tufwgo binary not found: %w", err)
	}

	gotHash, err := getHash(path)
	if err != nil {
		return fmt.Errorf("unable to get file hash for %s: %w", path, err)
	}

	if gotHash == expectedSHA256 {
		return fmt.Errorf("TUFWGo is already up to date")
	}

	fmt.Println("Update available for TUFWGo!")
	fmt.Println("Updating TUFWGo...")
	if err = downloadFile(path, expectedSHA256); err != nil {
		return fmt.Errorf("unable to update TUFWGo: %w", err)
	}
	return nil
}

func (manifest *Manifest) checkAuthBin() error {
	path := "/usr/bin/tufwgo-auth"
	expectedSHA256 := manifest.Binaries["tufwgo-auth"].SHA256

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("tufwgo-auth binary not found: %w", err)
	}

	gotHash, err := getHash(path)
	if err != nil {
		return fmt.Errorf("unable to get file hash for %s: %w", path, err)
	}

	if gotHash == expectedSHA256 {
		return fmt.Errorf("TUFWGo SSH Auth Helper is already up to date")
	}

	fmt.Println("Update available for TUFWGo SSH Auth Helper!")
	fmt.Println("Updating TUFWGo SSH Auth Helper...")
	if err = downloadFile(path, expectedSHA256); err != nil {
		return fmt.Errorf("unable to update TUFWGo SSH Auth Helper: %w", err)
	}
	return nil
}

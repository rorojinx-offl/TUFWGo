package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

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

	dest := "/usr/bin/tufwgo"
	expectedSHA256 := manifest.Binaries["tufwgo"].SHA256

	response, err := http.Get("https://dl.tufwgo.store/binaries/tufwgo")
	if err != nil {
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Printf("failed to download file: received status code %d", response.StatusCode)
		return
	}

	destFile, err := os.Create(dest)
	if err != nil {
		fmt.Println("")
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
		fmt.Printf("error saving file: %s", err)
		return
	}
	_ = destFile.Close()
	_ = bar.Finish()

	gotHash := hex.EncodeToString(h.Sum(nil))
	if expectedSHA256 != "" && gotHash != expectedSHA256 {
		fmt.Printf("expected SHA256: %s, got SHA256: %s", expectedSHA256, gotHash)
		return
	}

	if err = os.Chmod(dest, 0755); err != nil {
		fmt.Printf("error setting file permissions: %s", err)
		return
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

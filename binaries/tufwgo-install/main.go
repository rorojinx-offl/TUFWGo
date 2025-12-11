package main

import (
	"TUFWGo/system/local"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"io"
	"net/http"
	"os"
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

	var path, expectedSHA256, url string

	path = "/usr/bin/tufwgo"
	expectedSHA256 = manifest.Binaries["tufwgo"].SHA256
	url = manifest.Binaries["tufwgo"].URL
	if _, err = os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("Installing TUFWGo...")
			err = downloadFile(path, expectedSHA256, url)
			if err != nil {
				fmt.Println(err)
				return
			}
		} else {
			fmt.Println(err)
			return
		}
	} else {
		fmt.Println("TUFWGO is already installed")
	}

	path = "/usr/bin/tufwgo-update"
	expectedSHA256 = manifest.Binaries["tufwgo-update"].SHA256
	url = manifest.Binaries["tufwgo-update"].URL
	if _, err = os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("Installing TUFWGo Updater...")
			err = downloadFile(path, expectedSHA256, url)
			if err != nil {
				fmt.Println(err)
				return
			}
		} else {
			fmt.Println(err)
			return
		}
	} else {
		fmt.Println("TUFWGO Updater is already installed")
	}

	pkg, err := getPkgMr()
	if err != nil {
		fmt.Println(err)
		return
	}

	deps, err := local.FindDependencies()
	if err != nil {
		fmt.Println(err)
	}
	if len(deps) == 0 {
		fmt.Println("You already have all the dependencies installed!!!")
		return
	}
	errs := local.InstallDependencies(deps, local.DerivePkgMgrKeywords(pkg))
	if len(errs) != 0 {
		for _, err = range errs {
			fmt.Println(err)
		}
		fmt.Println("Dependencies installed with some errors!")
		return
	}
	fmt.Println("Dependencies installed successfully!")

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

func downloadFile(dest, expectedSHA256, url string) error {
	response, err := http.Get(url)
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
		return fmt.Errorf("expected SHA256: %s, got SHA256: %s", expectedSHA256, gotHash)
	}

	if err = os.Chmod(dest, 0755); err != nil {
		return fmt.Errorf("error setting file permissions: %w", err)
	}

	return nil
}

func getPkgMr() (string, error) {
	pkg := local.DetectPkgMgr()
	if pkg == local.UNKNOWN {
		return "", errors.New("could not detect package manager")
	}

	return string(pkg), nil
}

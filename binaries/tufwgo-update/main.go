package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

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

var checkOnly = flag.Bool("checko", false, "Only checks if there is an update, not install it")

func main() {
	requireRoot()

	manifest := &Manifest{}
	err := manifest.fetchManifest()
	if err != nil {
		fmt.Println(err)
		return
	}

	flag.Parse()
	if *checkOnly {
		updateCount, err := manifest.justCheck()
		if (err != nil) || (updateCount == -1) {
			fmt.Printf("There was an error in checking updates: %s\n", err)
			os.Exit(1)
		}

		if updateCount == 0 {
			fmt.Printf("There are no updates to install!")
			os.Exit(0)
		}

		if updateCount == 1 {
			fmt.Printf("There is an update to install!")
			os.Exit(0)
		}

		if updateCount > 1 {
			fmt.Printf("There are %d updates to install!", updateCount)
			os.Exit(0)
		}
		return
	}

	//needUpdate, err := manifest.selfCheck()

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

	if err = manifest.checkDeployBin(); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("TUFWGo Profile Deployment Center successfully updated!")
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

func getNonRootUserCfg() (string, error) {
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser == "" {
		return "", errors.New("unable to derive $SUDO_USER")
	}

	usrlkp, err := user.Lookup(sudoUser)
	if err != nil {
		return "", fmt.Errorf("unable to lookup evoking user %s: %v", sudoUser, err)
	}

	return fmt.Sprintf("%s/.config", usrlkp.HomeDir), nil
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

	return string(output), nil
}

func (manifest *Manifest) checkMain() error {
	path := "/usr/bin/tufwgo"
	expectedSHA256 := manifest.Binaries["tufwgo"].SHA256
	url := manifest.Binaries["tufwgo"].URL

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("tufwgo binary not found: %w", err)
	}

	gotHash, err := getHash(path)
	if err != nil {
		return fmt.Errorf("unable to get file hash for %s: %w", path, err)
	}

	vr, err := getVersion()
	if err != nil {
		return fmt.Errorf("unable to get version for TUFWGo: %w", err)
	}

	if (gotHash == expectedSHA256) || (vr == manifest.Version) {
		return fmt.Errorf("TUFWGo is already up to date")
	}

	fmt.Println("Update available for TUFWGo!")
	fmt.Println("Updating TUFWGo...")
	if err = downloadFile(path, expectedSHA256, url); err != nil {
		return fmt.Errorf("unable to update TUFWGo: %w", err)
	}
	return nil
}

func (manifest *Manifest) checkAuthBin() error {
	path := "/usr/bin/tufwgo-auth"
	expectedSHA256 := manifest.Binaries["tufwgo-auth"].SHA256
	url := manifest.Binaries["tufwgo-auth"].URL

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
	if err = downloadFile(path, expectedSHA256, url); err != nil {
		return fmt.Errorf("unable to update TUFWGo SSH Auth Helper: %w", err)
	}
	return nil
}

func (manifest *Manifest) checkDeployBin() error {
	cfg, err := getNonRootUserCfg()
	if err != nil {
		return err
	}
	expectedSHA256 := manifest.Binaries["tufwgo-deploy"].SHA256
	url := manifest.Binaries["tufwgo-deploy"].URL

	baseCfgPath := filepath.Join(cfg, "tufwgo")
	if _, err = os.Stat(baseCfgPath); err != nil {
		return fmt.Errorf("unable to find TUFWGo config path, make sure you run the app first for init setup")
	}

	binDir := filepath.Join(baseCfgPath, "pdc")
	if _, err = os.Stat(binDir); err != nil {
		return fmt.Errorf("unable to find any config for PDC, make sure you run the app to fix broken config")
	}

	binPath := filepath.Join(binDir, "tufwgo-deploy")
	if _, err = os.Stat(binPath); err != nil {
		return fmt.Errorf("unable to find existing PDC binary, make sure you run the app to fix broken config")
	}

	gotHash, err := getHash(binPath)
	if err != nil {
		return fmt.Errorf("unable to get file hash for %s: %w", binPath, err)
	}

	if gotHash == expectedSHA256 {
		return fmt.Errorf("TUFWGo Profile Deployment Center is already up to date")
	}

	fmt.Println("Update available for TUFWGo Profile Deployment Center!")
	fmt.Println("Updating TUFWGo Profile Deployment Center...")
	if err = downloadFile(binPath, expectedSHA256, url); err != nil {
		return fmt.Errorf("unable to update TUFWGo SSH Auth Helper: %w", err)
	}
	return nil
}

func (manifest *Manifest) selfCheck() (bool, error) {
	path := "/usr/bin/tufwgo-update"
	expectedSHA256 := manifest.Binaries["tufwgo-update"].SHA256

	gotHash, err := getHash(path)
	if err != nil {
		return false, fmt.Errorf("unable to get file hash for %s: %w", path, err)
	}

	if gotHash == expectedSHA256 {
		return false, nil
	}

	fmt.Println("Update available for TUFWGo Updater!")
	return true, nil
}

func (manifest *Manifest) justCheck() (int, error) {
	count := 0
	var path string
	var expectedSHA256 string
	var gotHash string

	needUpdate, err := manifest.selfCheck()
	if err != nil {
		return -1, err
	}

	if needUpdate {
		count++
	}

	path = "/usr/bin/tufwgo"
	expectedSHA256 = manifest.Binaries["tufwgo"].SHA256
	if _, err = os.Stat(path); err != nil {
		return -1, fmt.Errorf("tufwgo binary not found: %w", err)
	}
	gotHash, err = getHash(path)
	if err != nil {
		return -1, fmt.Errorf("unable to get file hash for %s: %w", path, err)
	}
	vr, err := getVersion()
	if err != nil {
		return -1, fmt.Errorf("unable to get version for TUFWGo: %w", err)
	}
	if (gotHash != expectedSHA256) || (vr != manifest.Version) {
		count++
	}

	path = "/usr/bin/tufwgo-auth"
	expectedSHA256 = manifest.Binaries["tufwgo-auth"].SHA256
	if _, err = os.Stat(path); err != nil {
		return -1, fmt.Errorf("tufwgo-auth binary not found: %w", err)
	}
	gotHash, err = getHash(path)
	if err != nil {
		return -1, fmt.Errorf("unable to get file hash for %s: %w", path, err)
	}
	if gotHash != expectedSHA256 {
		count++
	}

	cfg, err := getNonRootUserCfg()
	if err != nil {
		return -1, err
	}
	expectedSHA256 = manifest.Binaries["tufwgo-deploy"].SHA256
	baseCfgPath := filepath.Join(cfg, "tufwgo")
	if _, err = os.Stat(baseCfgPath); err != nil {
		return -1, fmt.Errorf("unable to find TUFWGo config path, make sure you restart the app for init setup")
	}

	binDir := filepath.Join(baseCfgPath, "pdc")
	if _, err = os.Stat(binDir); err != nil {
		return -1, fmt.Errorf("unable to find any config for PDC, make sure you restart the app to fix broken config")
	}

	binPath := filepath.Join(binDir, "tufwgo-deploy")
	if _, err = os.Stat(binPath); err != nil {
		return -1, fmt.Errorf("unable to find existing PDC binary, make sure you restart the app to fix broken config")
	}

	gotHash, err = getHash(binPath)
	if err != nil {
		return -1, fmt.Errorf("unable to get file hash for %s: %w", binPath, err)
	}

	if gotHash != expectedSHA256 {
		count++
	}
	return count, nil
}

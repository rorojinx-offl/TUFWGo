package local

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/schollz/progressbar/v3"
)

func prepareCommand(cmdStr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdStr)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return "", errors.New(fmt.Sprint("stderr:", stderr.String()))
	}
	return string(output), nil
}

func prepareCommandConversation(cmdStr, input string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdStr)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	err := cmd.Run()
	if err != nil {
		return "", errors.New(fmt.Sprint("stderr:", stderr.String()))
	}
	return stdout.String(), nil
}

func RunCommand(command string) (string, error) {
	out, err := prepareCommand(command)
	if err != nil {
		return "", err
	}
	return out, nil
}

func CommandConversation(command, reply string) (string, error) {
	out, err := prepareCommandConversation(command, reply)
	if err != nil {
		return "", err
	}
	return out, nil
}

func RequireRoot() {
	if os.Geteuid() != 0 {
		fmt.Println("This command requires root/sudo privileges! (try: sudo " + os.Args[0] + ")")
		os.Exit(77)
	}
}

func DownloadFile(url, dest, expectedSHA256 string) error {
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file: %s", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: received status code %d", response.StatusCode)
	}

	tmpDir, _ := os.UserHomeDir()
	tmpFile, err := os.CreateTemp(tmpDir, "download-*")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %s", err)
	}
	defer os.Remove(tmpFile.Name())

	bar := progressbar.DefaultBytes(response.ContentLength, "downloading")

	h := sha256.New()
	mw := io.MultiWriter(tmpFile, h, bar)
	if _, err = io.Copy(mw, response.Body); err != nil {
		return fmt.Errorf("error saving file: %s", err)
	}
	tmpFile.Close()

	gotHash := hex.EncodeToString(h.Sum(nil))
	if expectedSHA256 != "" && gotHash != expectedSHA256 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, gotHash)
	}

	if err = os.Rename(tmpFile.Name(), dest); err != nil {
		return fmt.Errorf("error moving file to destination: %s", err)
	}
	if err = os.Chmod(dest, 0755); err != nil {
		return fmt.Errorf("error setting file permissions: %s", err)
	}
	return nil
}

func EditEnv(path, key, value string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading env file: %s", err)
	}
	content := string(data)

	regex := regexp.MustCompile(fmt.Sprintf("export %s='.*'", key))
	newContent := regex.ReplaceAllString(content, fmt.Sprintf("export %s='%s'", key, value))

	err = os.WriteFile(path, []byte(newContent), 0644)
	if err != nil {
		return fmt.Errorf("error writing env file: %s", err)
	}
	return nil
}

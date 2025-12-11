package local

import "os/exec"

type PackageManager string

const (
	APT     PackageManager = "apt"
	DNF     PackageManager = "dnf"
	YUM     PackageManager = "yum"
	PACMAN  PackageManager = "pacman"
	ZYPPER  PackageManager = "zypper"
	NIX     PackageManager = "nix"
	APK     PackageManager = "apk"
	UNKNOWN PackageManager = "unknown"
)

func DetectPkgMgr() PackageManager {
	list := []PackageManager{APT, DNF, YUM, PACMAN, ZYPPER, NIX, APK}
	for _, m := range list {
		if _, err := exec.LookPath(string(m)); err != nil {
			continue
		}
		return m
	}
	return UNKNOWN
}

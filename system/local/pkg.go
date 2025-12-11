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
	UNKNOWN PackageManager = "unknown"
)

func DetectPkgMgr() PackageManager {
	list := []PackageManager{APT, DNF, YUM, PACMAN, ZYPPER, NIX}
	for _, m := range list {
		if _, err := exec.LookPath(string(m)); err != nil {
			continue
		}
		return m
	}
	return UNKNOWN
}

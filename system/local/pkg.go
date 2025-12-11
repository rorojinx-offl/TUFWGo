package local

import "os/exec"

type PackageManager string

const (
	APT     PackageManager = "apt"
	DNF     PackageManager = "dnf"
	PACMAN  PackageManager = "pacman"
	ZYPPER  PackageManager = "zypper"
	APK     PackageManager = "apk"
	UNKNOWN PackageManager = "unknown"
)

func DetectPkgMgr() PackageManager {
	list := []PackageManager{APT, DNF, PACMAN, ZYPPER, APK}
	for _, m := range list {
		if _, err := exec.LookPath(string(m)); err != nil {
			continue
		}
		return m
	}
	return UNKNOWN
}

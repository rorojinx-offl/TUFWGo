package audit

import (
	"TUFWGo/system/local"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func loadAuditKey() ([]byte, error) {
	b64 := os.Getenv("TUFWGO_AUDIT_KEY")
	if b64 == "" {
		return nil, errors.New("TUFWGO_AUDIT_KEY environment variable not set")
	}
	key, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode TUFWGO_AUDIT_KEY from base64: %w", err)
	}
	if len(key) < 32 {
		return nil, fmt.Errorf("key length is %d bytes; must be at least 32 bytes", len(key))
	}
	return key, nil
}

func OpenDailyAuditLog() (*Log, error) {
	/*cfgDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config dir: %w", err)
	}*/
	cfgDir := local.GlobalUserCfgDir
	auditDir := filepath.Join(cfgDir, "tufwgo", "audit")
	logPath := filepath.Join(auditDir, time.Now().UTC().Format("audit-2006-01-02.log"))
	key, err := loadAuditKey()
	if err != nil {
		return nil, err
	}

	prevHash, err := prevFileLastHash(auditDir, key, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	return Open(logPath, key, prevHash)
}

func findPreviousLog(auditDir string, day time.Time) (string, error) {
	entries, err := os.ReadDir(auditDir)
	if err != nil {
		return "", err
	}

	prefix := "audit-"
	suffix := ".log"
	var candidates []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			dateStr := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
			date, err := time.Parse("2006-01-02", dateStr)
			if err == nil && date.Before(day.Truncate(24*time.Hour)) {
				candidates = append(candidates, filepath.Join(auditDir, name))
			}
		}
	}
	if len(candidates) == 0 {
		return "", nil
	}
	sort.Strings(candidates)
	return candidates[len(candidates)-1], nil
}

func prevFileLastHash(auditDir string, key []byte, day time.Time) (string, error) {
	prevPath, err := findPreviousLog(auditDir, day)
	if err != nil {
		return "", err
	}
	if prevPath == "" {
		return "", nil
	}

	vrf, err := Verify(prevPath, key)
	if err != nil {
		return "", fmt.Errorf("failed to verify previous log: %w", err)
	}
	if !vrf.OK {
		return "", fmt.Errorf("prev log tampered (line %d: %s)", vrf.FailedLine, vrf.Reason)
	}
	return vrf.LastHashHex, nil
}

var globalAuditor *Log
var globalActor string

func SetGlobalAuditor(auditor *Log, actor string) {
	globalAuditor = auditor
	globalActor = actor
}

func GetGlobalAuditor() (*Log, string) {
	if globalAuditor != nil && globalActor != "" {
		return globalAuditor, globalActor
	}
	return nil, ""
}

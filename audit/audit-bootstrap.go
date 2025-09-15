package audit

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config dir: %w", err)
	}
	auditDir := filepath.Join(cfgDir, "audit")
	logPath := filepath.Join(auditDir, time.Now().UTC().Format("audit-2006-01-02.log"))
	key, err := loadAuditKey()
	if err != nil {
		return nil, err
	}
	return Open(logPath, key, "")
}

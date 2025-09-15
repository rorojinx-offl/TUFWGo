package audit

import (
	"bufio"
	hmac2 "crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type VerifyResult struct {
	OK          bool
	FailedLine  int
	Reason      string
	LastIndex   uint64
	LastHashHex string
}

func Verify(path string, key []byte) (*VerifyResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 10MB max line size

	lineNo := 0

	if !scanner.Scan() {
		return nil, errors.New("audit: empty file")
	}
	lineNo++
	var hdr header
	if err = json.Unmarshal(scanner.Bytes(), &hdr); err != nil || hdr.Kind != "hdr" {
		return &VerifyResult{OK: false, FailedLine: lineNo, Reason: "invalid header"}, nil
	}

	prevHashHex := hdr.SeedHex
	var lastIdx uint64

	for scanner.Scan() {
		lineNo++
		var se signedEntry
		if err = json.Unmarshal(scanner.Bytes(), &se); err != nil {
			return &VerifyResult{OK: false, FailedLine: lineNo, Reason: "corrupted log"}, nil
		}
		if se.Entry.Kind != "entry" {
			return &VerifyResult{OK: false, FailedLine: lineNo, Reason: "invalid log entry kind"}, nil
		}

		if subtle.ConstantTimeCompare([]byte(se.PrevHash), []byte(prevHashHex)) != 1 {
			return &VerifyResult{OK: false, FailedLine: lineNo, Reason: "broken hash chain"}, nil
		}

		entryJSON, _ := json.Marshal(se.Entry)
		prevBytes, err := hex.DecodeString(prevHashHex)
		if err != nil {
			return nil, fmt.Errorf("audit: invalid prev hash: %v", err)
		}
		hash := sha256.New()
		hash.Write(prevBytes)
		hash.Write(entryJSON)
		sum := hash.Sum(nil)
		hashHex := hex.EncodeToString(sum)

		if subtle.ConstantTimeCompare([]byte(se.Hash), []byte(hashHex)) != 1 {
			return &VerifyResult{OK: false, FailedLine: lineNo, Reason: "invalid entry hash"}, nil
		}

		hmac := hmac2.New(sha256.New, key)
		hmac.Write(sum)
		expectedHMAC := hex.EncodeToString(hmac.Sum(nil))
		if subtle.ConstantTimeCompare([]byte(se.HMAC), []byte(expectedHMAC)) != 1 {
			return &VerifyResult{OK: false, FailedLine: lineNo, Reason: "invalid entry HMAC"}, nil
		}

		prevHashHex = hashHex
		lastIdx = se.Entry.Index
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	return &VerifyResult{
		OK:          true,
		FailedLine:  0,
		Reason:      "",
		LastIndex:   lastIdx,
		LastHashHex: prevHashHex,
	}, nil
}

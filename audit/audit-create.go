package audit

import (
	"TUFWGo/ufw"
	"bufio"
	hmac2 "crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const currentVersion = 1.0

type header struct {
	Kind            string `json:"kind"`
	Version         int    `json:"version"`
	Created         string `json:"created"`
	Host            string `json:"host"`
	SeedHex         string `json:"seed"`
	PrevLogLastHash string `json:"prev_log_last_hash"`
}
type Field struct {
	Name        string   `json:"name"`
	Value       string   `json:"value"`
	Rule        ufw.Form `json:"rule,omitempty"`
	DeletedRule string   `json:"deleted_rule,omitempty"`
}
type Entry struct {
	Kind        string   `json:"kind"`
	Index       uint64   `json:"index"`
	Time        string   `json:"time"`
	Actor       string   `json:"actor"`
	Action      string   `json:"action"`
	Command     string   `json:"command,omitempty"`
	ProfCommand []string `json:"prof_command,omitempty"`
	Result      string   `json:"result"`
	Error       string   `json:"error,omitempty"`
	Fields      []Field  `json:"fields,omitempty"`
}
type signedEntry struct {
	Entry    Entry  `json:"entry"`
	PrevHash string `json:"prev_hash"`
	Hash     string `json:"hash"`
	HMAC     string `json:"hmac"`
}
type Log struct {
	mutex       sync.Mutex
	file        *os.File
	path        string
	key         []byte
	lastHashHex string
	nextIndex   uint64
}

func Open(path string, key []byte, prevLogHashHex string) (*Log, error) {
	if len(key) == 0 {
		return nil, errors.New("HMAC key is empty")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	log := &Log{file: file, path: path, key: key}

	stat, _ := file.Stat()
	if stat.Size() == 0 {
		seed := make([]byte, 32)
		if _, err = rand.Read(seed); err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("failed to generate random seed: %w", err)
		}
		hdr := header{
			Kind:            "hdr",
			Version:         currentVersion,
			Created:         time.Now().Format("2006-01-02 15:04:05"),
			Host:            hostname(),
			SeedHex:         hex.EncodeToString(seed),
			PrevLogLastHash: prevLogHashHex,
		}
		if err = writeJSONLine(file, hdr); err != nil {
			_ = file.Close()
			return nil, err
		}
		log.lastHashHex = hdr.SeedHex
		log.nextIndex = 1
	} else {
		lastHash, nextIdx, err := scanTail(path)
		if err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("failed to read existing log: %w", err)
		}
		log.lastHashHex = lastHash
		log.nextIndex = nextIdx
	}
	return log, nil
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

func writeJSONLine(w io.Writer, v any) error {
	bytes, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(append(bytes, '\n'))
	if err2 := syncFile(w); err == nil && err2 != nil {
		err = err2
	}
	return err
}

func syncFile(w io.Writer) error {
	if file, ok := w.(*os.File); ok {
		return file.Sync()
	}
	return nil
}

func scanTail(path string) (lastHashHex string, nextIndex uint64, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	var hdr header
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 10MB max line size

	if !scanner.Scan() {
		return "", 0, errors.New("empty file missing header")
	}
	if err = json.Unmarshal(scanner.Bytes(), &hdr); err != nil || hdr.Kind != "hdr" {
		return "", 0, errors.New("invalid header")
	}

	lastHashHex = hdr.SeedHex
	nextIndex = 1

	for scanner.Scan() {
		line := scanner.Bytes()
		var se signedEntry
		if err = json.Unmarshal(line, &se); err != nil {
			return "", 0, fmt.Errorf("invalid log entry: %w", err)
		}
		if se.Entry.Kind != "entry" {
			return "", 0, errors.New("invalid log entry kind")
		}
		lastHashHex = se.Hash
		nextIndex = se.Entry.Index + 1
	}
	if err = scanner.Err(); err != nil {
		return "", 0, err
	}
	return lastHashHex, nextIndex, nil
}

func (l *Log) Append(e *Entry) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if e.Kind == "" {
		e.Kind = "entry"
	}
	e.Index = l.nextIndex
	e.Time = time.Now().Format("2006-01-02 15:04:05")

	entryJSON, err := json.Marshal(e)
	if err != nil {
		return err
	}

	prevBytes, err := hex.DecodeString(l.lastHashHex)
	if err != nil {
		return fmt.Errorf("failed to decode previous hash: %w", err)
	}

	hash := sha256.New()
	hash.Write(prevBytes)
	hash.Write(entryJSON)
	sum := hash.Sum(nil)
	hashHex := hex.EncodeToString(sum)

	hmac := hmac2.New(sha256.New, l.key)
	hmac.Write(sum)
	hmacHex := hex.EncodeToString(hmac.Sum(nil))

	se := signedEntry{
		Entry:    *e,
		PrevHash: l.lastHashHex,
		Hash:     hashHex,
		HMAC:     hmacHex,
	}
	if err = writeJSONLine(l.file, se); err != nil {
		return err
	}

	l.lastHashHex = hashHex
	l.nextIndex++
	return nil
}

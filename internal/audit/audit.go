package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Entry represents a single audit log entry
type Entry struct {
	Timestamp   time.Time `json:"timestamp"`
	Profile     string    `json:"profile"`
	Protocol    string    `json:"protocol"`
	Host        string    `json:"host"`
	User        string    `json:"user,omitempty"`
	Action      string    `json:"action"` // "connect", "disconnect", "error"
	Duration    string    `json:"duration,omitempty"`
	Error       string    `json:"error,omitempty"`
	ExitCode    int       `json:"exitCode,omitempty"`
}

// Logger handles audit logging
type Logger struct {
	path string
	file *os.File
}

// DefaultPath returns the default audit log path
func DefaultPath() (string, error) {
	cfgHome := os.Getenv("XDG_CONFIG_HOME")
	if cfgHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cfgHome = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgHome, "veessh", "audit.log"), nil
}

// NewLogger creates a new audit logger
func NewLogger() (*Logger, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}

	return &Logger{path: path, file: file}, nil
}

// Log writes an audit entry
func (l *Logger) Log(entry Entry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(l.file, string(data))
	return err
}

// Close closes the audit log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// LogConnect logs a connection start
func LogConnect(profile, protocol, host, user string) {
	logger, err := NewLogger()
	if err != nil {
		return // Silent fail - audit is optional
	}
	defer logger.Close()

	logger.Log(Entry{
		Timestamp: time.Now(),
		Profile:   profile,
		Protocol:  protocol,
		Host:      host,
		User:      user,
		Action:    "connect",
	})
}

// LogDisconnect logs a connection end
func LogDisconnect(profile, protocol, host, user string, startTime time.Time, exitCode int, connErr error) {
	logger, err := NewLogger()
	if err != nil {
		return // Silent fail - audit is optional
	}
	defer logger.Close()

	entry := Entry{
		Timestamp: time.Now(),
		Profile:   profile,
		Protocol:  protocol,
		Host:      host,
		User:      user,
		Action:    "disconnect",
		Duration:  time.Since(startTime).Round(time.Second).String(),
		ExitCode:  exitCode,
	}

	if connErr != nil {
		entry.Action = "error"
		entry.Error = connErr.Error()
	}

	logger.Log(entry)
}

// ReadEntries reads audit log entries (last N entries)
func ReadEntries(limit int) ([]Entry, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}

	var entries []Entry
	lines := splitLines(data)

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // Skip malformed lines
		}
		entries = append(entries, entry)
	}

	// Return last N entries
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	// Reverse to show most recent first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries, nil
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}


package safety

import (
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"
)

// ErrNilWriter is returned by AuditLogger.Log when the logger was constructed
// with a nil writer.
var ErrNilWriter = errors.New("audit logger: writer is nil")

// AuditEntry captures a single tool invocation for the audit log.
type AuditEntry struct {
	Timestamp time.Time      `json:"timestamp"`
	Tool      string         `json:"tool"`
	Params    map[string]any `json:"params"`
	Result    string         `json:"result"`
	Duration  time.Duration  `json:"duration_ns"`
}

// AuditLogger writes AuditEntry records as newline-delimited JSON to an
// io.Writer. It is safe for concurrent use.
type AuditLogger struct {
	mu sync.Mutex
	w  io.Writer
}

// NewAuditLogger returns an AuditLogger that writes to w. If w is nil the
// returned logger is also nil; callers must check for nil before use.
func NewAuditLogger(w io.Writer) *AuditLogger {
	if w == nil {
		return nil
	}
	return &AuditLogger{w: w}
}

// Log serialises entry as a single JSON line and writes it to the underlying
// writer. It returns an error if the writer is nil or if serialisation or
// writing fails. Log is safe for concurrent use.
func (l *AuditLogger) Log(entry AuditEntry) error {
	if l == nil || l.w == nil {
		return ErrNilWriter
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	data = append(data, '\n')

	l.mu.Lock()
	_, err = l.w.Write(data)
	l.mu.Unlock()

	return err
}

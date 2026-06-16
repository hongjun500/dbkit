package dbkit

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm/logger"
)

type logEntry struct {
	level  string
	msg    string
	fields []Field
}

type captureLogger struct {
	entries []logEntry
}

func (l *captureLogger) Debug(msg string, fields ...Field) {
	l.entries = append(l.entries, logEntry{level: "debug", msg: msg, fields: fields})
}
func (l *captureLogger) Info(msg string, fields ...Field) {
	l.entries = append(l.entries, logEntry{level: "info", msg: msg, fields: fields})
}
func (l *captureLogger) Warn(msg string, fields ...Field) {
	l.entries = append(l.entries, logEntry{level: "warn", msg: msg, fields: fields})
}
func (l *captureLogger) Error(msg string, fields ...Field) {
	l.entries = append(l.entries, logEntry{level: "error", msg: msg, fields: fields})
}

func (l *captureLogger) Debugw(_ context.Context, msg string, fields ...Field) {
	l.entries = append(l.entries, logEntry{level: "debug", msg: msg, fields: fields})
}
func (l *captureLogger) Infow(_ context.Context, msg string, fields ...Field) {
	l.entries = append(l.entries, logEntry{level: "info", msg: msg, fields: fields})
}
func (l *captureLogger) Warnw(_ context.Context, msg string, fields ...Field) {
	l.entries = append(l.entries, logEntry{level: "warn", msg: msg, fields: fields})
}
func (l *captureLogger) Errorw(_ context.Context, msg string, fields ...Field) {
	l.entries = append(l.entries, logEntry{level: "error", msg: msg, fields: fields})
}

func TestDBKitGORMLoggerTraceInfo(t *testing.T) {
	log := &captureLogger{}
	gl := newGORMLogger(log, "mysql", logger.Info)
	gl.Trace(context.Background(), time.Now().Add(-10*time.Millisecond), func() (string, int64) {
		return "SELECT 1", 1
	}, nil)

	if len(log.entries) != 1 {
		t.Fatalf("entries: got %d want 1", len(log.entries))
	}
	if log.entries[0].level != "info" {
		t.Fatalf("unexpected level: %s", log.entries[0].level)
	}
	if log.entries[0].msg != "gorm sql" {
		t.Fatalf("unexpected message: %s", log.entries[0].msg)
	}
}

func TestDBKitGORMLoggerTraceError(t *testing.T) {
	log := &captureLogger{}
	gl := newGORMLogger(log, "postgres", logger.Error)
	gl.Trace(context.Background(), time.Now().Add(-10*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM users", -1
	}, errors.New("db down"))

	if len(log.entries) != 1 {
		t.Fatalf("entries: got %d want 1", len(log.entries))
	}
	if log.entries[0].level != "error" {
		t.Fatalf("unexpected level: %s", log.entries[0].level)
	}
	if log.entries[0].msg != "gorm sql error" {
		t.Fatalf("unexpected message: %s", log.entries[0].msg)
	}
}

func TestDBKitGORMLoggerLogModeSilent(t *testing.T) {
	log := &captureLogger{}
	gl := newGORMLogger(log, "mysql", logger.Info).LogMode(logger.Silent)
	gl.Trace(context.Background(), time.Now().Add(-10*time.Millisecond), func() (string, int64) {
		return "SELECT 1", 1
	}, nil)

	if len(log.entries) != 0 {
		t.Fatalf("silent mode should suppress logs, got %d", len(log.entries))
	}
}

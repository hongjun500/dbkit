package dbkit

import (
	"strings"
	"testing"
	"time"

	"gorm.io/gorm/logger"
)

func TestParseGORMLogMode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    logger.LogLevel
		wantErr bool
	}{
		{name: "empty defaults silent", input: "", want: logger.Silent},
		{name: "silent", input: "silent", want: logger.Silent},
		{name: "uppercase info", input: "INFO", want: logger.Info},
		{name: "trimmed warn", input: "  warn  ", want: logger.Warn},
		{name: "debug alias", input: "debug", want: logger.Info},
		{name: "error", input: "error", want: logger.Error},
		{name: "invalid", input: "verbose", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGORMLogMode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("log mode mismatch: got %v want %v", got, tt.want)
			}
		})
	}
}

func TestEnsureMySQLDialTimeout(t *testing.T) {
	in := "user:pass@tcp(127.0.0.1:3306)/db?parseTime=true"
	out, err := ensureMySQLDialTimeout(in, 3*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == in {
		t.Fatal("expected dsn to be normalized with timeout fields")
	}
	if !containsAll(out, []string{"timeout=3s", "readTimeout=3s", "writeTimeout=3s"}) {
		t.Fatalf("normalized dsn missing timeout values: %s", out)
	}
}

func TestEnsureMySQLDialTimeout_KeepExisting(t *testing.T) {
	in := "user:pass@tcp(127.0.0.1:3306)/db?timeout=1s&readTimeout=2s&writeTimeout=4s"
	out, err := ensureMySQLDialTimeout(in, 3*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsAll(out, []string{"timeout=1s", "readTimeout=2s", "writeTimeout=4s"}) {
		t.Fatalf("existing timeout values should be kept: %s", out)
	}
}

func containsAll(s string, parts []string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

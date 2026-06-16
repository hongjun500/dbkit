package dbkit

import (
	"strings"
	"testing"
)

func TestRedactDSN_MySQLUserInfo(t *testing.T) {
	in := "root:secret@tcp(127.0.0.1:3306)/app?parseTime=true"
	got := redactDSN(in)
	if strings.Contains(got, "secret") {
		t.Fatalf("password should be redacted, got: %s", got)
	}
	if !strings.Contains(got, "root:***@") {
		t.Fatalf("unexpected redacted dsn: %s", got)
	}
}

func TestRedactDSN_PostgresKV(t *testing.T) {
	in := "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=test connect_timeout=2"
	got := redactDSN(in)
	if strings.Contains(got, "password=postgres") {
		t.Fatalf("password should be redacted, got: %s", got)
	}
	if !strings.Contains(strings.ToLower(got), "password=***") {
		t.Fatalf("redacted marker missing: %s", got)
	}
}

func TestRedactDSN_URL(t *testing.T) {
	in := "postgres://alice:secret@localhost:5432/mydb?sslmode=disable"
	got := redactDSN(in)
	if strings.Contains(got, "secret") {
		t.Fatalf("url password should be redacted, got: %s", got)
	}
	if !strings.Contains(got, "alice:") {
		t.Fatalf("unexpected redacted url dsn: %s", got)
	}
}

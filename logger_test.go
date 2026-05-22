package dbkit

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"testing"
)

type captureHandler struct {
	mu   sync.Mutex
	logs []map[string]any
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	m := map[string]any{
		"msg":   r.Message,
		"level": r.Level.String(),
	}
	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value.Any()
		return true
	})
	h.mu.Lock()
	h.logs = append(h.logs, m)
	h.mu.Unlock()
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(name string) slog.Handler       { return h }

func TestSlogLogger_StructuredFields(t *testing.T) {
	h := &captureHandler{}
	log := NewSlogLoggerFrom(slog.New(h))

	log.Infow(context.Background(), "database connected",
		String("db_type", "mysql"),
		String("component", "mysql"),
		Int("pool_max_open", 10),
	)

	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.logs) != 1 {
		t.Fatalf("logs: got %d want 1", len(h.logs))
	}
	if h.logs[0]["msg"] != "database connected" {
		t.Fatalf("msg: %v", h.logs[0]["msg"])
	}
	if h.logs[0]["db_type"] != "mysql" {
		t.Fatalf("db_type: %v", h.logs[0]["db_type"])
	}
}

func TestDefaultLogger_NotNil(t *testing.T) {
	if DefaultLogger() == nil {
		t.Fatal("DefaultLogger should not be nil")
	}
}

func TestRegistry_DefaultUsesSlog(t *testing.T) {
	reg := NewRegistry(Config{})
	if reg.logger == nil {
		t.Fatal("logger should not be nil")
	}
	if _, ok := reg.logger.(*SlogLogger); !ok {
		t.Fatalf("expected *SlogLogger, got %T", reg.logger)
	}
}

func TestFields_JSONRoundTrip(t *testing.T) {
	f := []Field{String("k", "v"), Int("n", 1), Bool("ok", true)}
	attrs := fieldsToAttrs(f)
	m := make(map[string]any)
	for _, a := range attrs {
		m[a.Key] = a.Value.Any()
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"k":"v"`)) {
		t.Fatalf("json: %s", b)
	}
}

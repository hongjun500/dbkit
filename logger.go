package dbkit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

// Logger 可插拔结构化日志接口，不绑定任何第三方日志库。
// 消息体仅承载事件描述；上下文一律通过 Field 键值对传递。
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
}

// Field 结构化日志字段（Key-Value）。
type Field struct {
	Key   string
	Value any
}

func String(key, val string) Field      { return Field{Key: key, Value: val} }
func Int(key string, val int) Field     { return Field{Key: key, Value: val} }
func Int64(key string, val int64) Field { return Field{Key: key, Value: val} }
func Bool(key string, val bool) Field   { return Field{Key: key, Value: val} }
func Duration(key string, d time.Duration) Field {
	return Field{Key: key, Value: d}
}
func Err(key string, err error) Field { return Field{Key: key, Value: err} }
func Any(key string, val any) Field   { return Field{Key: key, Value: val} }

// SlogLogger 基于标准库 slog 的默认实现（Go 1.21+）。
type SlogLogger struct {
	l *slog.Logger
}

// NewSlogLogger 使用 JSON 输出到 stderr 创建 SlogLogger（适合生产环境）。
func NewSlogLogger() *SlogLogger {
	return NewSlogLoggerFrom(slog.New(newSQLJSONHandler(os.Stderr, slog.LevelInfo)))
}

// NewSlogLoggerText 使用文本格式输出到 stderr（适合本地开发）。
func NewSlogLoggerText() *SlogLogger {
	return NewSlogLoggerFrom(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
}

// NewSlogLoggerFrom 包装已有 *slog.Logger。
func NewSlogLoggerFrom(l *slog.Logger) *SlogLogger {
	if l == nil {
		l = slog.New(newSQLJSONHandler(os.Stderr, slog.LevelInfo))
	}
	return &SlogLogger{l: l}
}

func (l *SlogLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelDebug, msg, fields)
}

func (l *SlogLogger) Info(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelInfo, msg, fields)
}

func (l *SlogLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelWarn, msg, fields)
}

func (l *SlogLogger) Error(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelError, msg, fields)
}

func (l *SlogLogger) log(ctx context.Context, level slog.Level, msg string, fields []Field) {
	attrs := fieldsToAttrs(append(contextFields(ctx), fields...))
	l.l.LogAttrs(ctx, level, msg, attrs...)
}

// DefaultLogger 返回库内全局默认 Logger（slog JSON → stderr）。
func DefaultLogger() Logger {
	return defaultLogger
}

var defaultLogger Logger = NewSlogLogger()

func fieldsToAttrs(fields []Field) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(fields))
	for _, f := range fields {
		attrs = append(attrs, fieldToAttr(f))
	}
	return attrs
}

func fieldToAttr(f Field) slog.Attr {
	switch v := f.Value.(type) {
	case string:
		return slog.String(f.Key, v)
	case int:
		return slog.Int(f.Key, v)
	case int64:
		return slog.Int64(f.Key, v)
	case bool:
		return slog.Bool(f.Key, v)
	case time.Duration:
		return slog.Duration(f.Key, v)
	case error:
		return slog.Any(f.Key, v)
	default:
		return slog.Any(f.Key, v)
	}
}

// ContextFields 从 context 提取可观测性字段（供适配器与 *w 方法复用）。
func ContextFields(ctx context.Context) []Field {
	if ctx == nil {
		return nil
	}
	// 预留：对接 OpenTelemetry trace/span 时在此追加字段
	return nil
}

func contextFields(ctx context.Context) []Field { return ContextFields(ctx) }

var _ Logger = (*SlogLogger)(nil)

// SQLJSONHandler 输出 JSON 日志，并将 SQL 原文附加在 JSON 记录后。
type SQLJSONHandler struct {
	out    io.Writer
	level  slog.Leveler
	mu     *sync.Mutex
	attrs  []slog.Attr
	groups []string
}

func newSQLJSONHandler(out io.Writer, level slog.Leveler) *SQLJSONHandler {
	return &SQLJSONHandler{
		out:   out,
		level: level,
		mu:    &sync.Mutex{},
	}
}

func (h *SQLJSONHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *SQLJSONHandler) Handle(_ context.Context, r slog.Record) error {
	data := make(map[string]any)

	data["time"] = r.Time.Format(time.RFC3339Nano)
	data["level"] = r.Level.String()
	data["msg"] = r.Message

	var sql string
	var addAttr func(a slog.Attr)
	addAttr = func(a slog.Attr) {
		a.Value = a.Value.Resolve()
		if a.Equal(slog.Attr{}) {
			return
		}
		if a.Value.Kind() == slog.KindGroup {
			for _, child := range a.Value.Group() {
				addAttr(child)
			}
			return
		}

		if a.Key == "sql" {
			sql = a.Value.String()
			return
		}

		data[a.Key] = a.Value.Any()
	}

	for _, a := range h.attrs {
		addAttr(a)
	}

	r.Attrs(func(a slog.Attr) bool {
		addAttr(a)
		return true
	})

	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}

	var output strings.Builder
	output.Write(encoded)
	output.WriteByte('\n')

	if sql != "" {
		output.WriteString("SQL:\n")
		output.WriteString(sql)
		output.WriteString("\n")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err = io.WriteString(h.out, output.String())
	return err
}

func (h *SQLJSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &clone
}

func (h *SQLJSONHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	clone := *h
	clone.groups = append(append([]string(nil), h.groups...), name)
	return &clone
}

var _ slog.Handler = (*SQLJSONHandler)(nil)

var _ = fmt.Sprint

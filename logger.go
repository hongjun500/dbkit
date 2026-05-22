package dbkit

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// Logger 可插拔结构化日志接口，不绑定任何第三方日志库。
// 消息体仅承载事件描述；上下文一律通过 Field 键值对传递。
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)

	Debugw(ctx context.Context, msg string, fields ...Field)
	Infow(ctx context.Context, msg string, fields ...Field)
	Warnw(ctx context.Context, msg string, fields ...Field)
	Errorw(ctx context.Context, msg string, fields ...Field)
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
	return NewSlogLoggerFrom(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
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
		l = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return &SlogLogger{l: l}
}

func (l *SlogLogger) Debug(msg string, fields ...Field) {
	l.log(context.Background(), slog.LevelDebug, msg, fields)
}
func (l *SlogLogger) Info(msg string, fields ...Field) {
	l.log(context.Background(), slog.LevelInfo, msg, fields)
}
func (l *SlogLogger) Warn(msg string, fields ...Field) {
	l.log(context.Background(), slog.LevelWarn, msg, fields)
}
func (l *SlogLogger) Error(msg string, fields ...Field) {
	l.log(context.Background(), slog.LevelError, msg, fields)
}

func (l *SlogLogger) Debugw(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelDebug, msg, fields)
}
func (l *SlogLogger) Infow(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelInfo, msg, fields)
}
func (l *SlogLogger) Warnw(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, slog.LevelWarn, msg, fields)
}
func (l *SlogLogger) Errorw(ctx context.Context, msg string, fields ...Field) {
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

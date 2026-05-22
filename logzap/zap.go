// Package logzap 提供基于 go.uber.org/zap 的 dbkit.Logger 适配器（可选依赖）。
package logzap

import (
	"context"

	"github.com/hongjun500/dbkit"
	"go.uber.org/zap"
)

// Logger 将 zap 适配为 dbkit.Logger。
type Logger struct {
	z *zap.Logger
}

// New 使用生产环境默认配置创建 Logger。
func New() (*Logger, error) {
	z, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return &Logger{z: z}, nil
}

// From 包装已有 *zap.Logger；nil 时使用 Nop。
func From(z *zap.Logger) *Logger {
	if z == nil {
		z = zap.NewNop()
	}
	return &Logger{z: z}
}

func (l *Logger) Debug(msg string, fields ...dbkit.Field) { l.z.Debug(msg, toZap(fields)...) }
func (l *Logger) Info(msg string, fields ...dbkit.Field)  { l.z.Info(msg, toZap(fields)...) }
func (l *Logger) Warn(msg string, fields ...dbkit.Field)  { l.z.Warn(msg, toZap(fields)...) }
func (l *Logger) Error(msg string, fields ...dbkit.Field) { l.z.Error(msg, toZap(fields)...) }

func (l *Logger) Debugw(ctx context.Context, msg string, fields ...dbkit.Field) {
	l.z.Debug(msg, toZap(append(dbkit.ContextFields(ctx), fields...))...)
}
func (l *Logger) Infow(ctx context.Context, msg string, fields ...dbkit.Field) {
	l.z.Info(msg, toZap(append(dbkit.ContextFields(ctx), fields...))...)
}
func (l *Logger) Warnw(ctx context.Context, msg string, fields ...dbkit.Field) {
	l.z.Warn(msg, toZap(append(dbkit.ContextFields(ctx), fields...))...)
}
func (l *Logger) Errorw(ctx context.Context, msg string, fields ...dbkit.Field) {
	l.z.Error(msg, toZap(append(dbkit.ContextFields(ctx), fields...))...)
}

// Sync 刷新缓冲，进程退出前调用。
func (l *Logger) Sync() error { return l.z.Sync() }

func toZap(fields []dbkit.Field) []zap.Field {
	out := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		switch v := f.Value.(type) {
		case string:
			out = append(out, zap.String(f.Key, v))
		case int:
			out = append(out, zap.Int(f.Key, v))
		case int64:
			out = append(out, zap.Int64(f.Key, v))
		case bool:
			out = append(out, zap.Bool(f.Key, v))
		case error:
			out = append(out, zap.NamedError(f.Key, v))
		default:
			out = append(out, zap.Any(f.Key, v))
		}
	}
	return out
}

var _ dbkit.Logger = (*Logger)(nil)

package dbkit

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 可插拔日志接口，业务方可注入自定义实现。
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
}

// Field 键值对日志字段。
type Field struct {
	Key   string
	Value any
}

func String(key, val string) Field  { return Field{Key: key, Value: val} }
func Int(key string, val int) Field { return Field{Key: key, Value: val} }
func Err(key string, err error) Field {
	return Field{Key: key, Value: err}
}

// ZapLogger 基于 zap 的默认实现。
type ZapLogger struct {
	z *zap.Logger
}

// NewZapLogger 使用生产环境默认配置创建 ZapLogger。
func NewZapLogger() (*ZapLogger, error) {
	z, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return &ZapLogger{z: z}, nil
}

// NewZapLoggerFrom 包装已有 *zap.Logger。
func NewZapLoggerFrom(z *zap.Logger) *ZapLogger {
	if z == nil {
		z = zap.NewNop()
	}
	return &ZapLogger{z: z}
}

func (l *ZapLogger) Debug(msg string, fields ...Field) { l.z.Debug(msg, toZap(fields)...) }
func (l *ZapLogger) Info(msg string, fields ...Field)  { l.z.Info(msg, toZap(fields)...) }
func (l *ZapLogger) Warn(msg string, fields ...Field)  { l.z.Warn(msg, toZap(fields)...) }
func (l *ZapLogger) Error(msg string, fields ...Field) { l.z.Error(msg, toZap(fields)...) }

// Sync 刷新缓冲，Close 前调用。
func (l *ZapLogger) Sync() error { return l.z.Sync() }

func toZap(fields []Field) []zap.Field {
	out := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		switch v := f.Value.(type) {
		case string:
			out = append(out, zap.String(f.Key, v))
		case int:
			out = append(out, zap.Int(f.Key, v))
		case int64:
			out = append(out, zap.Int64(f.Key, v))
		case error:
			out = append(out, zap.NamedError(f.Key, v))
		default:
			out = append(out, zap.Any(f.Key, v))
		}
	}
	return out
}

// nopLogger 未注入 logger 时的空实现。
type nopLogger struct{}

func (nopLogger) Debug(string, ...Field) {}
func (nopLogger) Info(string, ...Field)  {}
func (nopLogger) Warn(string, ...Field)  {}
func (nopLogger) Error(string, ...Field) {}

// 编译期断言
var _ Logger = (*ZapLogger)(nil)
var _ Logger = nopLogger{}

// zapGormLevel 将配置字符串映射为 zap level（供 GORM 日志适配，此处仅保留扩展点）。
func zapGormLevel(level string) zapcore.Level {
	switch level {
	case "silent":
		return zapcore.FatalLevel + 1
	case "error":
		return zapcore.ErrorLevel
	case "warn":
		return zapcore.WarnLevel
	default:
		return zapcore.InfoLevel
	}
}

package dbkit

import (
	"context"
	"database/sql"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// 统一结构化日志字段名，避免各驱动散落魔法字符串。
const (
	logKeyDBType    = "db_type"
	logKeyComponent = "component"
	logKeyEvent     = "event"
	logKeyAddr      = "addr"
	logKeyDSN       = "dsn"
	logKeyError     = "error"
	logKeyMaxOpen   = "pool_max_open"
	logKeyMaxIdle   = "pool_max_idle"
	logKeyPoolSize  = "pool_size"
	logKeyMinIdle   = "pool_min_idle"
	logKeyDial      = "dial_timeout"
)

func logConnectStart(ctx context.Context, log Logger, dbType string, fields ...Field) {
	base := []Field{
		String(logKeyDBType, dbType),
		String(logKeyComponent, dbType),
		String(logKeyEvent, "connect_start"),
	}
	log.Info(ctx, "opening database connection", append(base, fields...)...)
}

func logConnectOK(ctx context.Context, log Logger, dbType string, fields ...Field) {
	base := []Field{
		String(logKeyDBType, dbType),
		String(logKeyComponent, dbType),
		String(logKeyEvent, "connect_ok"),
	}
	log.Info(ctx, "database connected", append(base, fields...)...)
}

func logConnectFail(ctx context.Context, log Logger, dbType, stage string, err error, fields ...Field) {
	base := []Field{
		String(logKeyDBType, dbType),
		String(logKeyComponent, dbType),
		String(logKeyEvent, "connect_fail"),
		String("stage", stage),
		Err(logKeyError, err),
	}
	log.Error(ctx, "database connection failed", append(base, fields...)...)
}

func sqlPoolFields(pool PoolConfig) []Field {
	return []Field{
		Int(logKeyMaxOpen, pool.MaxOpenConns),
		Int(logKeyMaxIdle, pool.MaxIdleConns),
		Duration("pool_conn_max_lifetime", pool.ConnMaxLifetime),
		Duration("pool_conn_max_idle_time", pool.ConnMaxIdleTime),
	}
}

func statsPoolFields(db *sql.DB) []Field {
	if db == nil {
		return nil
	}
	st := db.Stats()
	return []Field{
		Int("pool_open", st.OpenConnections),
		Int("pool_in_use", st.InUse),
		Int("pool_idle", st.Idle),
	}
}

func redactDSN(dsn string) string {
	if dsn == "" {
		return ""
	}
	redacted := redactURLUserPassword(dsn)
	redacted = redactMySQLStyleUserPassword(redacted)
	redacted = redactPasswordKV(redacted)

	// 仅用于日志：截断过长 DSN，避免刷屏。
	const maxLen = 128
	if len(redacted) <= maxLen {
		return redacted
	}
	return redacted[:maxLen] + "..."
}

var passwordKVRe = regexp.MustCompile(`(?i)\b(password|passwd|pwd)\s*=\s*([^&\s;]+)`)

func redactPasswordKV(s string) string {
	return passwordKVRe.ReplaceAllString(s, "${1}=***")
}

func redactURLUserPassword(s string) string {
	u, err := url.Parse(s)
	if err != nil || u.User == nil {
		return s
	}
	username := u.User.Username()
	if username == "" {
		return s
	}
	u.User = url.UserPassword(username, "***")
	return u.String()
}

func redactMySQLStyleUserPassword(s string) string {
	if strings.Contains(s, "://") {
		return s
	}
	at := strings.Index(s, "@")
	if at <= 0 {
		return s
	}
	cred := s[:at]
	colon := strings.Index(cred, ":")
	if colon <= 0 {
		return s
	}
	return cred[:colon+1] + "***" + s[at:]
}

func dialField(d time.Duration) Field {
	if d <= 0 {
		return Duration(logKeyDial, 0)
	}
	return Duration(logKeyDial, d)
}

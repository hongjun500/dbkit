package dbkit

import (
	"context"
	"database/sql"
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
	log.Infow(ctx, "opening database connection", append(base, fields...)...)
}

func logConnectOK(ctx context.Context, log Logger, dbType string, fields ...Field) {
	base := []Field{
		String(logKeyDBType, dbType),
		String(logKeyComponent, dbType),
		String(logKeyEvent, "connect_ok"),
	}
	log.Infow(ctx, "database connected", append(base, fields...)...)
}

func logConnectFail(ctx context.Context, log Logger, dbType, stage string, err error, fields ...Field) {
	base := []Field{
		String(logKeyDBType, dbType),
		String(logKeyComponent, dbType),
		String(logKeyEvent, "connect_fail"),
		String("stage", stage),
		Err(logKeyError, err),
	}
	log.Errorw(ctx, "database connection failed", append(base, fields...)...)
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
	// 仅用于日志：截断过长 DSN，避免刷屏；密码仍可能存在于 DSN，生产环境建议业务方自定义 Logger 脱敏
	const maxLen = 128
	if len(dsn) <= maxLen {
		return dsn
	}
	return dsn[:maxLen] + "..."
}

func dialField(d time.Duration) Field {
	if d <= 0 {
		return Duration(logKeyDial, 0)
	}
	return Duration(logKeyDial, d)
}

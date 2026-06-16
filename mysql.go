package dbkit

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	stdmysql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// openMySQL 打开 MySQL GORM 连接并配置连接池。
func openMySQL(ctx context.Context, cfg MySQLConfig, log Logger) (*gorm.DB, error) {
	cfg = cfg.withDefaults()
	if cfg.DSN == "" {
		return nil, fmt.Errorf("dbkit mysql: dsn is required when enabled")
	}

	pool := cfg.Pool
	logConnectStart(ctx, log, "mysql",
		String(logKeyDSN, redactDSN(cfg.DSN)),
		dialField(cfg.Dial),
	)

	logMode, err := parseGORMLogMode(cfg.LogLevel)
	if err != nil {
		logConnectFail(ctx, log, "mysql", "parse_log_level", err, String("log_level", cfg.LogLevel))
		return nil, err
	}

	dsnForOpen, err := ensureMySQLDialTimeout(cfg.DSN, cfg.Dial)
	if err != nil {
		logConnectFail(ctx, log, "mysql", "normalize_dsn", err, String(logKeyDSN, redactDSN(cfg.DSN)))
		return nil, err
	}

	gormCfg := &gorm.Config{
		Logger: newGORMLogger(log, "mysql", logMode),
	}

	db, err := gorm.Open(mysql.Open(dsnForOpen), gormCfg)
	if err != nil {
		logConnectFail(ctx, log, "mysql", "open", err, String(logKeyDSN, redactDSN(cfg.DSN)))
		return nil, fmt.Errorf("dbkit mysql: open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		logConnectFail(ctx, log, "mysql", "get_sql_db", err)
		return nil, fmt.Errorf("dbkit mysql: get sql.DB: %w", err)
	}
	applyPool(sqlDB, pool)

	if cfg.Dial > 0 {
		pingCtx, cancel := context.WithTimeout(ctx, cfg.Dial)
		defer cancel()
		if err := sqlDB.PingContext(pingCtx); err != nil {
			_ = sqlDB.Close()
			logConnectFail(ctx, log, "mysql", "ping", err, dialField(cfg.Dial))
			return nil, fmt.Errorf("dbkit mysql: ping: %w", err)
		}
	}

	fields := append([]Field{
		String(logKeyDSN, redactDSN(cfg.DSN)),
		dialField(cfg.Dial),
	}, sqlPoolFields(pool)...)
	fields = append(fields, statsPoolFields(sqlDB)...)
	logConnectOK(ctx, log, "mysql", fields...)
	return db, nil
}

// pingMySQL 健康检查。
func pingMySQL(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// closeMySQL 关闭连接池。
func closeMySQL(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func applyPool(sqlDB *sql.DB, pool PoolConfig) {
	sqlDB.SetMaxOpenConns(pool.MaxOpenConns)
	sqlDB.SetMaxIdleConns(pool.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(pool.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(pool.ConnMaxIdleTime)
}

func parseGORMLogMode(level string) (gormlogger.LogLevel, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "silent":
		return gormlogger.Silent, nil
	case "error":
		return gormlogger.Error, nil
	case "warn":
		return gormlogger.Warn, nil
	case "info", "debug":
		return gormlogger.Info, nil
	default:
		return gormlogger.Silent, fmt.Errorf("dbkit: invalid gorm log level %q (allowed: silent,error,warn,info)", level)
	}
}

func ensureMySQLDialTimeout(dsn string, dial time.Duration) (string, error) {
	if dial <= 0 {
		return dsn, nil
	}

	cfg, err := stdmysql.ParseDSN(dsn)
	if err != nil {
		return "", fmt.Errorf("dbkit mysql: parse dsn: %w", err)
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = dial
	}
	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = dial
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = dial
	}
	return cfg.FormatDSN(), nil
}

// MySQLHealthCheck 对外的 Ping 辅助（带默认超时）。
func MySQLHealthCheck(ctx context.Context, db *gorm.DB, timeout time.Duration) error {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return pingMySQL(ctx, db)
}

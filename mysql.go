package dbkit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// openMySQL 打开 MySQL GORM 连接并配置连接池。
func openMySQL(ctx context.Context, cfg MySQLConfig, log Logger) (*gorm.DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("dbkit mysql: dsn is required when enabled")
	}

	pool := cfg.Pool.withDefaults()
	logConnectStart(ctx, log, "mysql",
		String(logKeyDSN, redactDSN(cfg.DSN)),
		dialField(cfg.Dial),
	)

	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}
	if cfg.LogLevel != "" && cfg.LogLevel != "silent" {
		gormCfg.Logger = logger.Default.LogMode(gormLogMode(cfg.LogLevel))
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN), gormCfg)
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

func gormLogMode(level string) logger.LogLevel {
	switch level {
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	case "debug":
		return logger.Info
	default:
		return logger.Silent
	}
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

package dbkit

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// openPostgres 打开 PostgreSQL GORM 连接并配置连接池。
func openPostgres(ctx context.Context, cfg PostgresConfig, log Logger) (*gorm.DB, error) {
	cfg = cfg.withDefaults()
	if cfg.DSN == "" {
		return nil, fmt.Errorf("dbkit postgres: dsn is required when enabled")
	}

	pool := cfg.Pool
	logConnectStart(ctx, log, "postgres",
		String(logKeyDSN, redactDSN(cfg.DSN)),
		dialField(cfg.Dial),
	)

	logMode, err := parseGORMLogMode(cfg.LogLevel)
	if err != nil {
		logConnectFail(ctx, log, "postgres", "parse_log_level", err, String("log_level", cfg.LogLevel))
		return nil, err
	}

	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logMode),
	}
	db, err := gorm.Open(postgres.Open(cfg.DSN), gormCfg)
	if err != nil {
		logConnectFail(ctx, log, "postgres", "open", err, String(logKeyDSN, redactDSN(cfg.DSN)))
		return nil, fmt.Errorf("dbkit postgres: open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		logConnectFail(ctx, log, "postgres", "get_sql_db", err)
		return nil, fmt.Errorf("dbkit postgres: get sql.DB: %w", err)
	}
	applyPool(sqlDB, pool)

	if cfg.Dial > 0 {
		pingCtx, cancel := context.WithTimeout(ctx, cfg.Dial)
		defer cancel()
		if err := sqlDB.PingContext(pingCtx); err != nil {
			_ = sqlDB.Close()
			logConnectFail(ctx, log, "postgres", "ping", err, dialField(cfg.Dial))
			return nil, fmt.Errorf("dbkit postgres: ping: %w", err)
		}
	}

	fields := append([]Field{
		String(logKeyDSN, redactDSN(cfg.DSN)),
		dialField(cfg.Dial),
	}, sqlPoolFields(pool)...)
	fields = append(fields, statsPoolFields(sqlDB)...)
	logConnectOK(ctx, log, "postgres", fields...)
	return db, nil
}

func pingPostgres(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func closePostgres(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

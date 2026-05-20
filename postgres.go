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
	if cfg.DSN == "" {
		return nil, fmt.Errorf("dbkit postgres: dsn is required when enabled")
	}

	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}
	if cfg.LogLevel != "" && cfg.LogLevel != "silent" {
		gormCfg.Logger = logger.Default.LogMode(gormLogMode(cfg.LogLevel))
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("dbkit postgres: open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("dbkit postgres: get sql.DB: %w", err)
	}
	applyPool(sqlDB, cfg.Pool.withDefaults())

	if cfg.Dial > 0 {
		pingCtx, cancel := context.WithTimeout(ctx, cfg.Dial)
		defer cancel()
		if err := sqlDB.PingContext(pingCtx); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("dbkit postgres: ping: %w", err)
		}
	}

	log.Info("postgres connected", String("component", "postgres"))
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

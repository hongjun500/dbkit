package dbkit

import (
	"testing"
	"time"
)

func TestPoolConfig_WithDefaultsClampIdle(t *testing.T) {
	p := PoolConfig{
		MaxOpenConns:    5,
		MaxIdleConns:    8,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: time.Minute,
	}.withDefaults()

	if p.MaxIdleConns != 5 {
		t.Fatalf("max idle should be clamped to max open: got %d", p.MaxIdleConns)
	}
}

func TestMySQLConfig_WithDefaultsDial(t *testing.T) {
	cfg := (MySQLConfig{}).withDefaults()
	if cfg.Dial != 5*time.Second {
		t.Fatalf("mysql dial default mismatch: got %s", cfg.Dial)
	}
}

func TestPostgresConfig_WithDefaultsDial(t *testing.T) {
	cfg := (PostgresConfig{}).withDefaults()
	if cfg.Dial != 5*time.Second {
		t.Fatalf("postgres dial default mismatch: got %s", cfg.Dial)
	}
}

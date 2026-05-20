// 演示在同一 SDK 下，系统 A（MySQL+Redis）与系统 C（PostgreSQL+ES）
// 如何通过不同配置实现按需加载与优雅关闭。
// 本示例使用不可达地址，无需真实中间件即可运行。
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/hongjun500/dbkit"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger, err := dbkit.NewZapLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	fmt.Println("=== 系统 A：MySQL + Redis ===")
	runSystemA(ctx, logger)

	fmt.Println("\n=== 系统 C：PostgreSQL + Elasticsearch ===")
	runSystemC(ctx, logger)
}

func runSystemA(ctx context.Context, log dbkit.Logger) {
	reg := dbkit.NewRegistry(dbkit.Config{
		MySQL: dbkit.MySQLConfig{
			Enabled: true,
			DSN:     "root:hongjun500@tcp(127.0.0.1:3306)/dianping?parseTime=true&timeout=2s",
			Dial:    2 * time.Second,
			Pool:    dbkit.PoolConfig{MaxOpenConns: 20, MaxIdleConns: 5},
		},
		Redis: dbkit.RedisConfig{
			Enabled:  true,
			Addr:     "127.0.0.1:6379",
			Password: "",
			DB:       0,
			Dial:     2 * time.Second,
		},
		// 系统 A 未启用 PG / ES
	}, dbkit.WithLogger(log))
	defer func() {
		if err := reg.Close(); err != nil {
			fmt.Printf("  close: %v\n", err)
		} else {
			fmt.Println("  closed gracefully")
		}
	}()

	tryDisabled(reg, "postgres", func() error {
		_, err := reg.Postgres(ctx)
		return err
	})
	tryDisabled(reg, "elasticsearch", func() error {
		_, err := reg.Elasticsearch(ctx)
		return err
	})

	tryEnabled(ctx, reg, "mysql", func() error {
		_, err := reg.MySQL(ctx)
		return err
	})
	tryEnabled(ctx, reg, "redis", func() error {
		_, err := reg.Redis(ctx)
		return err
	})
}

func runSystemC(ctx context.Context, log dbkit.Logger) {
	reg := dbkit.NewRegistry(dbkit.Config{
		Postgres: dbkit.PostgresConfig{
			Enabled: true,
			DSN:     "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=postgres connect_timeout=2",
			Dial:    2 * time.Second,
		},
		Elasticsearch: dbkit.ElasticsearchConfig{
			Enabled:   true,
			Addresses: []string{"http://127.0.0.1:9200"},
			Username: "123",
			Password: "123",
			Dial:      2 * time.Second,
		},
	}, dbkit.WithLogger(log))
	defer func() {
		if err := reg.Close(); err != nil {
			fmt.Printf("  close: %v\n", err)
		} else {
			fmt.Println("  closed gracefully")
		}
	}()

	tryDisabled(reg, "mysql", func() error {
		_, err := reg.MySQL(ctx)
		return err
	})
	tryDisabled(reg, "redis", func() error {
		_, err := reg.Redis(ctx)
		return err
	})

	tryEnabled(ctx, reg, "postgres", func() error {
		_, err := reg.Postgres(ctx)
		return err
	})
	tryEnabled(ctx, reg, "elasticsearch", func() error {
		_, err := reg.Elasticsearch(ctx)
		return err
	})
}

func tryDisabled(reg *dbkit.Registry, name string, fn func() error) {
	err := fn()
	if errors.Is(err, dbkit.ErrComponentNotEnabled) {
		fmt.Printf("  [%s] not configured -> ErrComponentNotEnabled (expected)\n", name)
		return
	}
	fmt.Printf("  [%s] unexpected: %v\n", name, err)
}

func tryEnabled(ctx context.Context, reg *dbkit.Registry, name string, fn func() error) {
	_ = ctx
	err := fn()
	if errors.Is(err, dbkit.ErrComponentNotEnabled) {
		fmt.Printf("  [%s] should be enabled but got ErrComponentNotEnabled\n", name)
		return
	}
	if err != nil {
		fmt.Printf("  [%s] init attempted, connection failed (expected in demo): %v\n", name, err)
		return
	}
	fmt.Printf("  [%s] connected\n", name)
}

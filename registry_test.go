package dbkit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestRegistry_DisabledComponentsReturnErrNotEnabled(t *testing.T) {
	// 模拟系统 B：仅 MySQL + Redis
	reg := NewRegistry(Config{
		MySQL:    MySQLConfig{Enabled: true, DSN: "invalid"},
		Postgres: PostgresConfig{Enabled: false},
		Redis:    RedisConfig{Enabled: true, Addr: "127.0.0.1:1"},
		Elasticsearch: ElasticsearchConfig{
			Enabled: false,
		},
	})

	ctx := context.Background()

	_, err := reg.Postgres(ctx)
	if !errors.Is(err, ErrComponentNotEnabled) {
		t.Fatalf("postgres: want ErrComponentNotEnabled, got %v", err)
	}

	_, err = reg.Elasticsearch(ctx)
	if !errors.Is(err, ErrComponentNotEnabled) {
		t.Fatalf("elasticsearch: want ErrComponentNotEnabled, got %v", err)
	}

	mysqlOn, pgOn, redisOn, esOn := reg.Enabled()
	if !mysqlOn || pgOn || !redisOn || esOn {
		t.Fatalf("enabled flags: mysql=%v postgres=%v redis=%v es=%v", mysqlOn, pgOn, redisOn, esOn)
	}
}

func TestRegistry_EnabledComponentInitFailsWithoutPanic(t *testing.T) {
	// 启用但未配置可达地址：应返回连接/超时错误，而非 Panic 或 ErrComponentNotEnabled
	short := 200 * time.Millisecond
	reg := NewRegistry(Config{
		MySQL: MySQLConfig{
			Enabled: true,
			DSN:     "user:pass@tcp(127.0.0.1:1)/db?timeout=200ms&readTimeout=200ms&writeTimeout=200ms",
			Dial:    short,
		},
		Redis: RedisConfig{
			Enabled: true,
			Addr:    "127.0.0.1:1",
			Dial:    short,
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := reg.MySQL(ctx)
	if err == nil {
		t.Fatal("mysql: expected connection error on unreachable host")
	}
	if errors.Is(err, ErrComponentNotEnabled) {
		t.Fatal("mysql: should not return ErrComponentNotEnabled when enabled")
	}

	_, err = reg.Redis(ctx)
	if err == nil {
		t.Fatal("redis: expected connection error on unreachable host")
	}
	if errors.Is(err, ErrComponentNotEnabled) {
		t.Fatal("redis: should not return ErrComponentNotEnabled when enabled")
	}
}

func TestRegistry_LazyInitOnce(t *testing.T) {
	reg := NewRegistry(Config{
		Postgres: PostgresConfig{
			Enabled: true,
			DSN:     "host=127.0.0.1 port=1 user=u password=p dbname=d connect_timeout=1",
			Dial:    200 * time.Millisecond,
		},
	})

	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make([]error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = reg.Postgres(ctx)
		}(i)
	}
	wg.Wait()

	first := errs[0]
	for _, e := range errs[1:] {
		if e != first && (e == nil) != (first == nil) {
			t.Fatalf("concurrent postgres init: inconsistent errors %v vs %v", first, e)
		}
	}
}

func TestRegistry_CloseIdempotent(t *testing.T) {
	reg := NewRegistry(Config{})
	if err := reg.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := reg.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}

	ctx := context.Background()
	_, err := reg.MySQL(ctx)
	if err == nil {
		t.Fatal("after close: expected error")
	}
	if err.Error() != "dbkit: registry is closed" {
		t.Fatalf("after close: want registry closed error, got %v", err)
	}
}

func TestRegistry_SystemC_OnlyPGAndES(t *testing.T) {
	reg := NewRegistry(Config{
		MySQL:    MySQLConfig{Enabled: false},
		Redis:    RedisConfig{Enabled: false},
		Postgres: PostgresConfig{Enabled: true, DSN: "host=127.0.0.1 port=1 user=u password=p dbname=d connect_timeout=1", Dial: 100 * time.Millisecond},
		Elasticsearch: ElasticsearchConfig{
			Enabled:   true,
			Addresses: []string{"http://127.0.0.1:1"},
			Dial:      100 * time.Millisecond,
		},
	})

	ctx := context.Background()

	_, err := reg.MySQL(ctx)
	if !errors.Is(err, ErrComponentNotEnabled) {
		t.Fatalf("mysql: %v", err)
	}
	_, err = reg.Redis(ctx)
	if !errors.Is(err, ErrComponentNotEnabled) {
		t.Fatalf("redis: %v", err)
	}

	_, err = reg.Postgres(ctx)
	if err == nil || errors.Is(err, ErrComponentNotEnabled) {
		t.Fatalf("postgres: want connection error, got %v", err)
	}

	_, err = reg.Elasticsearch(ctx)
	if err == nil || errors.Is(err, ErrComponentNotEnabled) {
		t.Fatalf("elasticsearch: want connection error, got %v", err)
	}
}

func TestRegistry_CustomLogger(t *testing.T) {
	log := NewZapLoggerFrom(nil)
	reg := NewRegistry(Config{}, WithLogger(log))
	if reg.logger == nil {
		t.Fatal("logger should be set")
	}
}

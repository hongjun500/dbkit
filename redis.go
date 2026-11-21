package dbkit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient 对 go-redis 的薄封装，提供常用操作与 Pub/Sub 入口。
type RedisClient struct {
	rdb *redis.Client
}

func openRedis(ctx context.Context, cfg RedisConfig, log Logger) (*RedisClient, error) {
	cfg = cfg.withDefaults()
	/* if cfg.Addr == "" {
		return nil, fmt.Errorf("dbkit redis: addr is required when enabled")
	} */
	opt, err := cfg.buildOptions()
	if err != nil {
		return nil, fmt.Errorf("dbkit redis: build options: %w", err)
	}
	useTLS := opt.TLSConfig != nil

	logConnectStart(ctx, log, "redis",
		String(logKeyAddr, opt.Addr),
		Int("db", opt.DB),
		Int(logKeyPoolSize, opt.PoolSize),
		Int(logKeyMinIdle, opt.MinIdleConns),
		Bool("tls", useTLS),
		dialField(opt.DialTimeout),
	)

	rdb := redis.NewClient(opt)

	pingCtx, cancel := context.WithTimeout(ctx, opt.DialTimeout+opt.ReadTimeout)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		_ = rdb.Close()
		logConnectFail(ctx, log, "redis", "ping", err,
			String(logKeyAddr, opt.Addr),
			Bool("tls", useTLS),
			dialField(cfg.Dial),
		)
		return nil, fmt.Errorf("dbkit redis: ping: %w", err)
	}

	st := rdb.PoolStats()
	logConnectOK(ctx, log, "redis",
		String(logKeyAddr, opt.Addr),
		Int("db", opt.DB),
		Int(logKeyPoolSize, opt.PoolSize),
		Int(logKeyMinIdle, opt.MinIdleConns),
		Bool("tls", useTLS),
		Int("pool_hits", int(st.Hits)),
		Int("pool_misses", int(st.Misses)),
		Int("pool_timeouts", int(st.Timeouts)),
		Int("pool_total_conns", int(st.TotalConns)),
		Int("pool_idle_conns", int(st.IdleConns)),
	)
	return &RedisClient{rdb: rdb}, nil
}

// buildOptions 以 URL 为基础，叠加显式字段，再补齐 dbkit 默认值。
func (c RedisConfig) buildOptions() (*redis.Options, error) {
	opt := &redis.Options{}
	if c.URL != "" {
		parsed, err := redis.ParseURL(c.URL)
		if err != nil {
			return nil, fmt.Errorf("parse url: %w", sanitizeURLErr(err))
		}
		opt = parsed
	}

	// 显式字段覆盖 URL
	if c.Addr != "" {
		opt.Addr = c.Addr
	}
	if c.Username != "" {
		opt.Username = c.Username
	}
	if c.Password != "" {
		opt.Password = c.Password
	}
	if c.DB != 0 {
		opt.DB = c.DB
	}
	if c.PoolSize > 0 {
		opt.PoolSize = c.PoolSize
	}
	if c.MinIdleConns > 0 {
		opt.MinIdleConns = c.MinIdleConns
	}
	if c.Dial > 0 {
		opt.DialTimeout = c.Dial
	}
	if c.Read > 0 {
		opt.ReadTimeout = c.Read
	}
	if c.Write > 0 {
		opt.WriteTimeout = c.Write
	}

	// 仍为零值的，补 dbkit 默认值
	if opt.Addr == "" {
		return nil, fmt.Errorf("addr or url is required")
	}
	
	// TLS：显式启用，或 URL 为 rediss:// 时 opt.TLSConfig 已非 nil
	if c.TLS.Enabled || opt.TLSConfig != nil {
		t := c.TLS
		t.Enabled = true
		if t.ServerName == "" && opt.TLSConfig != nil {
			t.ServerName = opt.TLSConfig.ServerName // 沿用 URL 解析出的 host
		}
		tc, err := t.build(hostOf(opt.Addr))
		if err != nil {
			return nil, err
		}
		opt.TLSConfig = tc
	}
	return opt, nil
}

func (c *RedisClient) Raw() *redis.Client { return c.rdb }

func (c *RedisClient) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *RedisClient) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

func (c *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

func (c *RedisClient) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

func (c *RedisClient) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.rdb.Exists(ctx, keys...).Result()
}

// Publish 发布消息到频道。
func (c *RedisClient) Publish(ctx context.Context, channel string, message any) error {
	return c.rdb.Publish(ctx, channel, message).Err()
}

// Subscribe 返回 Pub/Sub 订阅，调用方负责 Close。
func (c *RedisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.rdb.Subscribe(ctx, channels...)
}

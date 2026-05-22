package dbkit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// Registry 组件注册与生命周期管理：按需、延迟、并发安全加载。
type Registry struct {
	cfg    Config
	logger Logger

	mu     sync.RWMutex
	closed bool

	mysql struct {
		once sync.Once
		db   *gorm.DB
		err  error
	}
	postgres struct {
		once sync.Once
		db   *gorm.DB
		err  error
	}
	redis struct {
		once sync.Once
		cli  *RedisClient
		err  error
	}
	es struct {
		once sync.Once
		cli  *ESClient
		err  error
	}
}

// RegistryOption 可选配置。
type RegistryOption func(*Registry)

// WithLogger 注入自定义 Logger。
func WithLogger(l Logger) RegistryOption {
	return func(r *Registry) {
		if l != nil {
			r.logger = l
		}
	}
}

// NewRegistry 根据配置创建注册表；未启用的组件不会被初始化。
func NewRegistry(cfg Config, opts ...RegistryOption) *Registry {
	r := &Registry{cfg: cfg, logger: DefaultLogger()}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *Registry) ensureOpen() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return errors.New("dbkit: registry is closed")
	}
	return nil
}

// MySQL 获取 MySQL GORM 实例（延迟初始化）。
func (r *Registry) MySQL(ctx context.Context) (*gorm.DB, error) {
	if err := r.ensureOpen(); err != nil {
		return nil, err
	}
	if !r.cfg.MySQL.Enabled {
		return nil, ErrComponentNotEnabled
	}
	r.mysql.once.Do(func() {
		r.mysql.db, r.mysql.err = openMySQL(ctx, r.cfg.MySQL, r.logger)
	})
	if r.mysql.err != nil {
		return nil, r.mysql.err
	}
	return r.mysql.db, nil
}

// Postgres 获取 PostgreSQL GORM 实例（延迟初始化）。
func (r *Registry) Postgres(ctx context.Context) (*gorm.DB, error) {
	if err := r.ensureOpen(); err != nil {
		return nil, err
	}
	if !r.cfg.Postgres.Enabled {
		return nil, ErrComponentNotEnabled
	}
	r.postgres.once.Do(func() {
		r.postgres.db, r.postgres.err = openPostgres(ctx, r.cfg.Postgres, r.logger)
	})
	if r.postgres.err != nil {
		return nil, r.postgres.err
	}
	return r.postgres.db, nil
}

// Redis 获取 Redis 客户端（延迟初始化）。
func (r *Registry) Redis(ctx context.Context) (*RedisClient, error) {
	if err := r.ensureOpen(); err != nil {
		return nil, err
	}
	if !r.cfg.Redis.Enabled {
		return nil, ErrComponentNotEnabled
	}
	r.redis.once.Do(func() {
		r.redis.cli, r.redis.err = openRedis(ctx, r.cfg.Redis, r.logger)
	})
	if r.redis.err != nil {
		return nil, r.redis.err
	}
	return r.redis.cli, nil
}

// Elasticsearch 获取 ES 客户端（延迟初始化）。
func (r *Registry) Elasticsearch(ctx context.Context) (*ESClient, error) {
	if err := r.ensureOpen(); err != nil {
		return nil, err
	}
	if !r.cfg.Elasticsearch.Enabled {
		return nil, ErrComponentNotEnabled
	}
	r.es.once.Do(func() {
		r.es.cli, r.es.err = openElasticsearch(ctx, r.cfg.Elasticsearch, r.logger)
	})
	if r.es.err != nil {
		return nil, r.es.err
	}
	return r.es.cli, nil
}

// Ping 对已启用且已成功初始化的组件执行健康检查。
func (r *Registry) Ping(ctx context.Context) map[string]error {
	results := make(map[string]error)
	timeout := 3 * time.Second

	if r.cfg.MySQL.Enabled && r.mysql.db != nil {
		pingCtx, cancel := context.WithTimeout(ctx, timeout)
		results["mysql"] = pingMySQL(pingCtx, r.mysql.db)
		cancel()
	}
	if r.cfg.Postgres.Enabled && r.postgres.db != nil {
		pingCtx, cancel := context.WithTimeout(ctx, timeout)
		results["postgres"] = pingPostgres(pingCtx, r.postgres.db)
		cancel()
	}
	if r.cfg.Redis.Enabled && r.redis.cli != nil {
		pingCtx, cancel := context.WithTimeout(ctx, timeout)
		results["redis"] = r.redis.cli.Ping(pingCtx)
		cancel()
	}
	if r.cfg.Elasticsearch.Enabled && r.es.cli != nil {
		pingCtx, cancel := context.WithTimeout(ctx, timeout)
		results["elasticsearch"] = r.es.cli.Ping(pingCtx)
		cancel()
	}
	return results
}

// Close 优雅关闭所有已初始化的连接。
func (r *Registry) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	r.mu.Unlock()

	var errs []error
	if r.mysql.db != nil {
		if err := closeMySQL(r.mysql.db); err != nil {
			errs = append(errs, fmt.Errorf("mysql: %w", err))
		}
	}
	if r.postgres.db != nil {
		if err := closePostgres(r.postgres.db); err != nil {
			errs = append(errs, fmt.Errorf("postgres: %w", err))
		}
	}
	if r.redis.cli != nil {
		if err := r.redis.cli.Close(); err != nil {
			errs = append(errs, fmt.Errorf("redis: %w", err))
		}
	}
	if r.es.cli != nil {
		if err := r.es.cli.Close(); err != nil {
			errs = append(errs, fmt.Errorf("elasticsearch: %w", err))
		}
	}
	return errors.Join(errs...)
}

// Enabled 返回各组件是否在配置中启用（不触发初始化）。
func (r *Registry) Enabled() (mysql, postgres, redis, elasticsearch bool) {
	return r.cfg.MySQL.Enabled,
		r.cfg.Postgres.Enabled,
		r.cfg.Redis.Enabled,
		r.cfg.Elasticsearch.Enabled
}

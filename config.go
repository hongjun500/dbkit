package dbkit

import "time"

// Config 统一入口配置，仅 Enabled=true 的组件会被注册表按需加载。
type Config struct {
	MySQL         MySQLConfig         `mapstructure:"mysql" json:"mysql" yaml:"mysql"`
	Postgres      PostgresConfig      `mapstructure:"postgres" json:"postgres" yaml:"postgres"`
	Redis         RedisConfig         `mapstructure:"redis" json:"redis" yaml:"redis"`
	Elasticsearch ElasticsearchConfig `mapstructure:"elasticsearch" json:"elasticsearch" yaml:"elasticsearch"`
}

// PoolConfig 关系型数据库连接池参数（GORM 底层 *sql.DB）。
type PoolConfig struct {
	MaxOpenConns    int           `mapstructure:"max_open_conns" json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" json:"conn_max_idle_time" yaml:"conn_max_idle_time"`
}

// MySQLConfig MySQL / GORM 配置。
type MySQLConfig struct {
	Enabled  bool          `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	DSN      string        `mapstructure:"dsn" json:"dsn" yaml:"dsn"`
	Pool     PoolConfig    `mapstructure:"pool" json:"pool" yaml:"pool"`
	Dial     time.Duration `mapstructure:"dial_timeout" json:"dial_timeout" yaml:"dial_timeout"`
	LogLevel string        `mapstructure:"log_level" json:"log_level" yaml:"log_level"`
}

// PostgresConfig PostgreSQL / GORM 配置。
type PostgresConfig struct {
	Enabled  bool          `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	DSN      string        `mapstructure:"dsn" json:"dsn" yaml:"dsn"`
	Pool     PoolConfig    `mapstructure:"pool" json:"pool" yaml:"pool"`
	Dial     time.Duration `mapstructure:"dial_timeout" json:"dial_timeout" yaml:"dial_timeout"`
	LogLevel string        `mapstructure:"log_level" json:"log_level" yaml:"log_level"`
}

// RedisConfig go-redis 客户端配置。
type RedisConfig struct {
	Enabled      bool          `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Addr         string        `mapstructure:"addr" json:"addr" yaml:"addr"`
	Username     string        `mapstructure:"username" json:"username" yaml:"username"`
	Password     string        `mapstructure:"password" json:"password" yaml:"password"`
	DB           int           `mapstructure:"db" json:"db" yaml:"db"`
	PoolSize     int           `mapstructure:"pool_size" json:"pool_size" yaml:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns" json:"min_idle_conns" yaml:"min_idle_conns"`
	Dial         time.Duration `mapstructure:"dial_timeout" json:"dial_timeout" yaml:"dial_timeout"`
	Read         time.Duration `mapstructure:"read_timeout" json:"read_timeout" yaml:"read_timeout"`
	Write        time.Duration `mapstructure:"write_timeout" json:"write_timeout" yaml:"write_timeout"`
}

// ElasticsearchConfig 官方 ES 客户端配置。
type ElasticsearchConfig struct {
	Enabled   bool          `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Addresses []string      `mapstructure:"addresses" json:"addresses" yaml:"addresses"`
	Username  string        `mapstructure:"username" json:"username" yaml:"username"`
	Password  string        `mapstructure:"password" json:"password" yaml:"password"`
	CloudID   string        `mapstructure:"cloud_id" json:"cloud_id" yaml:"cloud_id"`
	APIKey    string        `mapstructure:"api_key" json:"api_key" yaml:"api_key"`
	Dial      time.Duration `mapstructure:"dial_timeout" json:"dial_timeout" yaml:"dial_timeout"`
}


func (p PoolConfig) withDefaults() PoolConfig {
	if p.MaxOpenConns <= 0 {
		p.MaxOpenConns = 25
	}
	if p.MaxIdleConns <= 0 {
		p.MaxIdleConns = 10
	}
	if p.ConnMaxLifetime <= 0 {
		p.ConnMaxLifetime = time.Hour
	}
	if p.ConnMaxIdleTime <= 0 {
		p.ConnMaxIdleTime = 10 * time.Minute
	}
	return p
}


func (c RedisConfig) withDefaults() RedisConfig {
	if c.PoolSize <= 0 {
		c.PoolSize = 10
	}
	if c.MinIdleConns <= 0 {
		c.MinIdleConns = 2
	}
	if c.Dial <= 0 {
		c.Dial = 5 * time.Second
	}
	if c.Read <= 0 {
		c.Read = 3 * time.Second
	}
	if c.Write <= 0 {
		c.Write = 3 * time.Second
	}
	return c
}

func (c ElasticsearchConfig) withDefaults() ElasticsearchConfig {
	if c.Dial <= 0 {
		c.Dial = 5 * time.Second
	}
	return c
}

package dbkit

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// TLSConfig 是 Redis / Postgres / MySQL 共用的 TLS 配置。
type TLSConfig struct {
	Enabled            bool   // 是否启用 TLS（Redis 的 rediss:// 等 URL 会自动启用）
	ServerName         string // 证书校验用的 SNI；留空则取连接地址里的 host
	InsecureSkipVerify bool   // 跳过证书校验，仅限测试环境
	CAFile             string // 自定义 CA 根证书（私有 CA 才需要；设置后会替代系统根证书）
	CertFile           string // 客户端证书（双向 TLS）
	KeyFile            string // 客户端私钥（双向 TLS）
	MinVersion         uint16 // 0 表示 tls.VersionTLS12
}

// build 生成 *tls.Config。defaultServerName 通常传入连接地址里的 host。
func (t TLSConfig) build(defaultServerName string) (*tls.Config, error) {
	minVer := t.MinVersion
	if minVer == 0 {
		minVer = tls.VersionTLS12
	}
	tc := &tls.Config{
		MinVersion:         minVer,
		ServerName:         t.ServerName,
		InsecureSkipVerify: t.InsecureSkipVerify, //nolint:gosec // 由配置显式开启
	}
	if tc.ServerName == "" {
		tc.ServerName = defaultServerName
	}

	if t.CAFile != "" {
		pem, err := os.ReadFile(t.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read tls ca file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("tls ca file: no valid certificates found")
		}
		tc.RootCAs = pool
	}

	if t.CertFile != "" || t.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load tls client keypair: %w", err)
		}
		tc.Certificates = []tls.Certificate{cert}
	}
	return tc, nil
}
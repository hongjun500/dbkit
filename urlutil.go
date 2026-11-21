package dbkit

import (
	"errors"
	"net"
	"net/url"
)

// sanitizeURLErr 去掉 *url.Error 中携带的原始 URL（里面可能有密码），只保留底层原因。
func sanitizeURLErr(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return ue.Err
	}
	return err
}

// hostOf 从 "host:port" 中取出 host；解析失败则原样返回。
func hostOf(addr string) string {
	h, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return h
}
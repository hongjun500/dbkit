package dbkit

import "errors"

// ErrComponentNotEnabled 表示该中间件未在配置中启用，调用方应视为可选依赖未装配。
var ErrComponentNotEnabled = errors.New("dbkit: component not enabled")

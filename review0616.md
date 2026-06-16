# dbkit 审查与整改记录（2026-06-16）

本文档汇总本次审查发现与已落地整改，目标是以最小改动完成规范化收敛：默认使用标准库 `slog`，可选注入 `zap`，并修复连接与日志中的高风险问题。

## 一、问题汇总（按优先级）

### P0（高优先）
- `gorm` 日志级别解析大小写敏感，非法值会静默降级到 `Silent`，容易导致排障时“以为开了日志，实际没有”。
- DSN 脱敏不完整，日志中可能出现明文密码。
- MySQL `dial_timeout` 未覆盖 `gorm.Open` 阶段，连接初始化超时语义不完整。

### P1（中优先）
- 连接生命周期日志与 SQL 日志通道不同（`dbkit.Logger` vs GORM 默认 logger），输出风格不一致。
- 连接池参数边界未显式约束（`MaxIdleConns > MaxOpenConns`）。

### P2（低优先）
- `debug -> info` 的兼容行为缺少注释说明。
- 缺少针对日志级别解析、DSN 脱敏、超时注入与池参数边界的表驱动测试。

## 二、已完成整改

### 1) GORM 日志级别解析规范化（已完成）
- 新增 `parseGORMLogMode(level)`：
  - 输入统一 `TrimSpace + ToLower`。
  - 严格支持 `silent/error/warn/info`，并兼容 `debug -> info`。
  - 非法值直接返回错误（fail-fast），不再静默降级。
- MySQL/Postgres 初始化均改为使用该统一解析逻辑。

### 2) DSN 脱敏加强（已完成）
- `redactDSN` 现在同时处理：
  - URL 场景（如 `postgres://user:pass@...`）的 `userinfo` 密码掩码。
  - MySQL 风格 `user:pass@tcp(...)` 的密码掩码。
  - `password=` / `passwd=` / `pwd=` 键值对（大小写不敏感）掩码。
- 保留长度截断逻辑，避免超长日志刷屏。

### 3) MySQL 超时语义修正（已完成）
- 增加 `MySQLConfig.withDefaults()`，默认 `Dial=5s`。
- 新增 `ensureMySQLDialTimeout(dsn, dial)`，在 DSN 未显式设置时自动补齐：
  - `timeout`
  - `readTimeout`
  - `writeTimeout`
- 保障 `gorm.Open` 阶段具备可预期的超时边界。

### 4) 配置边界与注释（已完成）
- `PoolConfig.withDefaults()` 增加边界约束：`MaxIdleConns` 不允许超过 `MaxOpenConns`。
- `MySQLConfig.LogLevel`、`PostgresConfig.LogLevel` 注释明确支持值与 `debug` 兼容映射。
- 补充 `PostgresConfig.withDefaults()`，默认 `Dial=5s`（与其它组件统一）。

### 5) 日志实现一致性小修（已完成）
- `logzap` 适配器补齐 `time.Duration` 的类型映射，减少与 `slog` 的字段行为差异。
- `logzap` 的非 `*w` 方法也统一走上下文字段拼接路径（与 `*w` 方法行为保持一致）。

## 三、测试覆盖（已完成）

新增测试：
- `mysql_test.go`
  - `parseGORMLogMode`：覆盖大小写、空白、非法值、`debug` 兼容。
  - `ensureMySQLDialTimeout`：覆盖注入与“已有值不覆盖”。
- `logging_test.go`
  - MySQL 风格 DSN 密码脱敏。
  - Postgres 键值 DSN 密码脱敏。
  - URL DSN 密码脱敏。
- `config_test.go`
  - 连接池 `MaxIdleConns` 边界约束。
  - MySQL/Postgres 的默认 `Dial` 值。

## 四、当前日志策略（统一口径）

- dbkit 默认日志实现：标准库 `slog`（结构化）。
- `zap` 为可选注入实现（通过 `logzap` 适配器）。
- 连接生命周期日志统一走 `dbkit.Logger`。
- SQL 日志级别遵循 GORM 四级语义，`debug` 作为兼容别名映射到 `info`。

## 五、后续可选优化（未在本次最小改造内）

- 将 GORM SQL logger 适配到 `dbkit.Logger`，彻底收敛为单一日志通道。
- 将 `logzap` 拆分为独立 module，实现“默认用户零 zap 依赖”的发布形态。

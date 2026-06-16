package dbkit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const defaultGORMSlowThreshold = 200 * time.Millisecond

type dbkitGORMLogger struct {
	base                 Logger
	level                gormlogger.LogLevel
	dbType               string
	slowThreshold        time.Duration
	ignoreRecordNotFound bool
}

func newGORMLogger(base Logger, dbType string, level gormlogger.LogLevel) gormlogger.Interface {
	if base == nil {
		base = DefaultLogger()
	}
	return &dbkitGORMLogger{
		base:                 base,
		level:                level,
		dbType:               dbType,
		slowThreshold:        defaultGORMSlowThreshold,
		ignoreRecordNotFound: false,
	}
}

func (l *dbkitGORMLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	cp := *l
	cp.level = level
	return &cp
}

func (l *dbkitGORMLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Info {
		return
	}
	l.base.Infow(ctx, "gorm info", l.gormBaseFields("gorm_info", msg, data...)...)
}

func (l *dbkitGORMLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Warn {
		return
	}
	l.base.Warnw(ctx, "gorm warn", l.gormBaseFields("gorm_warn", msg, data...)...)
}

func (l *dbkitGORMLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Error {
		return
	}
	l.base.Errorw(ctx, "gorm error", l.gormBaseFields("gorm_error", msg, data...)...)
}

func (l *dbkitGORMLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level == gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	if err != nil && l.level >= gormlogger.Error && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.ignoreRecordNotFound) {
		sql, rows := fc()
		fields := l.gormTraceFields("gorm_sql_error", elapsed, sql, rows)
		fields = append(fields, Err(logKeyError, err))
		l.base.Errorw(ctx, "gorm sql error", fields...)
		return
	}

	if l.slowThreshold > 0 && elapsed > l.slowThreshold && l.level >= gormlogger.Warn {
		sql, rows := fc()
		fields := l.gormTraceFields("gorm_slow_sql", elapsed, sql, rows)
		fields = append(fields, Duration("slow_threshold", l.slowThreshold))
		l.base.Warnw(ctx, "gorm slow sql", fields...)
		return
	}

	if l.level >= gormlogger.Info {
		sql, rows := fc()
		l.base.Infow(ctx, "gorm sql", l.gormTraceFields("gorm_sql", elapsed, sql, rows)...)
	}
}

func (l *dbkitGORMLogger) gormBaseFields(event, msg string, data ...interface{}) []Field {
	fields := []Field{
		String(logKeyDBType, l.dbType),
		String(logKeyComponent, "gorm"),
		String(logKeyEvent, event),
		String("message", formatGORMMessage(msg, data...)),
	}
	return fields
}

func (l *dbkitGORMLogger) gormTraceFields(event string, elapsed time.Duration, sql string, rows int64) []Field {
	return []Field{
		String(logKeyDBType, l.dbType),
		String(logKeyComponent, "gorm"),
		String(logKeyEvent, event),
		String("sql", sql),
		Duration("elapsed", elapsed),
		Int64("rows", rows),
	}
}

func formatGORMMessage(msg string, data ...interface{}) string {
	if len(data) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, data...)
}

var _ gormlogger.Interface = (*dbkitGORMLogger)(nil)

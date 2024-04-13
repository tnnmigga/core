package mysql

import (
	"context"
	"time"

	"github.com/tnnmigga/nett/zlog"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm/logger"
)

type gormLogger struct{}

func (l gormLogger) LogMode(logger.LogLevel) logger.Interface {
	return l
}

func (l gormLogger) Info(ctx context.Context, f string, s ...interface{}) {
	if zlog.Logger().Level() > zapcore.InfoLevel {
		return
	}
	zlog.Infof(f, s...)
}

func (l gormLogger) Warn(ctx context.Context, f string, s ...interface{}) {
	if zlog.Logger().Level() > zapcore.WarnLevel {
		return
	}
	zlog.Warnf(f, s...)
}

func (l gormLogger) Error(ctx context.Context, f string, s ...interface{}) {
	if zlog.Logger().Level() > zapcore.ErrorLevel {
		return
	}
	zlog.Errorf(f, s...)
}

func (l gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if err != nil {
		sql, ra := fc()
		zlog.Errorf("exec sql error %v, SQL: %s, rows affected: %d", err, sql, ra)
	}
}

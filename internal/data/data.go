package data

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cigc/app/internal/conf"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type txCtxKey struct{}

// Data 持有 GORM 连接。表结构只来自 scripts/schema.sql，禁止 AutoMigrate。
type Data struct {
	db *gorm.DB
}

// DB 暴露给 Repo 使用。
func (d *Data) DB() *gorm.DB {
	return d.db
}

// Session 优先使用事务里的连接。
func (d *Data) Session(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txCtxKey{}).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return d.db.WithContext(ctx)
}

// InTx 开启事务并把连接放入 ctx。已在事务中则复用同一条连接（GORM savepoint），避免外键锁死。
func (d *Data) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return d.Session(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txCtxKey{}, tx))
	})
}

// Ping 用于 /health 探活。
func (d *Data) Ping(ctx context.Context) error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// NewData 打开 MySQL；启动时短暂重试，避免 Compose 里 API 早于 MySQL ready。
func NewData(c *conf.Data) (*Data, func(), error) {
	cfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}
	var db *gorm.DB
	var err error
	for i := 0; i < 30; i++ {
		db, err = gorm.Open(mysql.Open(c.Database.Source), cfg)
		if err == nil {
			sqlDB, pingErr := db.DB()
			if pingErr == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				pingErr = sqlDB.PingContext(ctx)
				cancel()
			}
			if pingErr == nil {
				break
			}
			err = pingErr
		}
		slog.Warn("mysql not ready, retrying", "attempt", i+1, "err", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("connect mysql: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	d := &Data{db: db}
	cleanup := func() {
		if cerr := sqlDB.Close(); cerr != nil {
			slog.Error("close db", "err", cerr)
		}
	}
	return d, cleanup, nil
}

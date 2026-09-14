package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/cigc/app/internal/biz"
	"github.com/cigc/app/internal/conf"
	"github.com/robfig/cron/v3"
)

// DepositCron 按 cron 扫 BSC USDT 收款。
type DepositCron struct {
	cronExpr string
	loc      *time.Location
	c        *cron.Cron
	deposit  *biz.DepositUseCase
}

// NewDepositCron 从 app 配置构造；表达式或收款未配则空操作。
func NewDepositCron(app *conf.App, deposit *biz.DepositUseCase) *DepositCron {
	tz := "Asia/Shanghai"
	if app != nil && app.SettleTimezone != "" {
		tz = app.SettleTimezone
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	expr := ""
	if app != nil {
		expr = app.DepositCron
	}
	return &DepositCron{cronExpr: expr, loc: loc, deposit: deposit}
}

// Start 启动调度。
func (d *DepositCron) Start() {
	if d == nil || d.cronExpr == "" || d.deposit == nil || !d.deposit.Enabled() {
		return
	}
	d.c = cron.New(cron.WithLocation(d.loc))
	_, err := d.c.AddFunc(d.cronExpr, func() {
		res, err := d.deposit.Scan(context.Background())
		if err != nil {
			slog.Error("deposit cron", "err", err)
			return
		}
		if res.Skipped {
			slog.Info("deposit cron skipped")
			return
		}
		slog.Info("deposit cron done",
			"from", res.FromBlock,
			"to", res.ToBlock,
			"matched", res.Matched,
			"abnormal", res.Abnormal,
		)
	})
	if err != nil {
		slog.Error("deposit cron schedule", "err", err)
		return
	}
	d.c.Start()
}

// Stop 停止调度。
func (d *DepositCron) Stop() {
	if d == nil || d.c == nil {
		return
	}
	ctx := d.c.Stop()
	<-ctx.Done()
}

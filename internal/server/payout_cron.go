package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/cigc/app/internal/biz"
	"github.com/cigc/app/internal/conf"
	"github.com/robfig/cron/v3"
)

// PayoutCron 按 cron 扫已通过的 USDT/ISPAY 提现并打款。
type PayoutCron struct {
	cronExpr string
	loc      *time.Location
	c        *cron.Cron
	withdraw *biz.WithdrawUseCase
}

// NewPayoutCron 从 app 配置构造；表达式为空或未开打款则空操作。
func NewPayoutCron(app *conf.App, withdraw *biz.WithdrawUseCase) *PayoutCron {
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
		expr = app.PayoutCron
	}
	return &PayoutCron{cronExpr: expr, loc: loc, withdraw: withdraw}
}

// Start 启动调度。
func (p *PayoutCron) Start() {
	if p == nil || p.cronExpr == "" || p.withdraw == nil || !p.withdraw.PayoutEnabled() {
		return
	}
	p.c = cron.New(cron.WithLocation(p.loc))
	_, err := p.c.AddFunc(p.cronExpr, func() {
		res, err := p.withdraw.RunPayout(context.Background(), 0)
		if err != nil {
			slog.Error("payout cron", "err", err)
			return
		}
		slog.Info("payout cron done",
			"scanned", res.Scanned,
			"sent", res.Sent,
			"passed", res.Passed,
			"failed", res.Failed,
			"skipped", res.Skipped,
		)
	})
	if err != nil {
		slog.Error("payout cron schedule", "err", err)
		return
	}
	p.c.Start()
}

// Stop 停止调度。
func (p *PayoutCron) Stop() {
	if p == nil || p.c == nil {
		return
	}
	ctx := p.c.Stop()
	<-ctx.Done()
}

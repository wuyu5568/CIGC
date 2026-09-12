package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/cigc/app/internal/biz"
	"github.com/cigc/app/internal/conf"
	"github.com/robfig/cron/v3"
)

// SettleCron 在配置时区按 cron 表达式触发日结。
type SettleCron struct {
	cronExpr string
	loc      *time.Location
	c        *cron.Cron
	settle   *biz.SettleUseCase
}

// NewSettleCron 从 app 配置构造调度器。
func NewSettleCron(app *conf.App, settle *biz.SettleUseCase) *SettleCron {
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
		expr = app.SettleCron
	}
	return &SettleCron{cronExpr: expr, loc: loc, settle: settle}
}

// Start 启动调度；表达式为空则空操作。
func (s *SettleCron) Start() {
	if s == nil || s.cronExpr == "" || s.settle == nil {
		return
	}
	s.c = cron.New(cron.WithLocation(s.loc))
	_, err := s.c.AddFunc(s.cronExpr, func() {
		res, err := s.settle.Run(context.Background(), false)
		if err != nil {
			slog.Error("settle cron", "err", err)
			return
		}
		if res.Skipped {
			slog.Info("settle cron skipped", "date", res.SettleDate)
			return
		}
		slog.Info("settle cron done",
			"date", res.SettleDate,
			"users", res.UserCount,
			"cap_updated", res.CapUpdated,
			"direct", res.DirectCount,
		)
	})
	if err != nil {
		slog.Error("settle cron schedule", "err", err)
		return
	}
	s.c.Start()
}

// Stop 停止调度。
func (s *SettleCron) Stop() {
	if s == nil || s.c == nil {
		return
	}
	ctx := s.c.Stop()
	<-ctx.Done()
}

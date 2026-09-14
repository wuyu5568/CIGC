package main

// 一次性把 BuySomething 合约上的待处理充值写入 recharge_balance。
// 这不是每日奖励日结（日结在 cmd/app 的 settle_cron）。

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cigc/app/internal/biz"
	"github.com/cigc/app/internal/conf"
	"github.com/cigc/app/internal/data"
)

var confPath string

func init() {
	flag.StringVar(&confPath, "conf", "configs/config.yaml", "config path")
}

func main() {
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("buy settle", "err", err)
		os.Exit(1)
	}
}

func run() error {
	unlock, err := lockSettle()
	if err != nil {
		slog.Info("buy settle skipped", "reason", "already running")
		return nil
	}
	defer unlock()

	cfg, err := conf.Load(confPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	d, cleanupData, err := data.NewData(&cfg.Data)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer cleanupData()

	deposit := biz.NewDepositUseCase(
		data.NewUserRepo(d),
		data.NewOrderRepo(d),
		data.NewChainDepositRepo(d),
		data.NewChainCursorRepo(d),
		data.NewChainReader(&cfg.App),
		d,
		&cfg.App,
		data.NewLedgerRepo(d),
	)
	if !deposit.Runnable() {
		slog.Info("buy settle skipped", "reason", "not configured")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	res, err := deposit.Run(ctx)
	if err != nil {
		return err
	}
	if res.Skipped {
		slog.Info("buy settle skipped", "mode", res.Mode, "head", res.HeadBlock)
		return nil
	}
	slog.Info("buy settle done",
		"mode", res.Mode,
		"head", res.HeadBlock,
		"at_block", res.ToBlock,
		"from_index", res.FromIndex,
		"to_index", res.ToIndex,
		"length", res.Length,
		"seen", res.Seen,
		"matched", res.Matched,
		"abnormal", res.Abnormal,
		"already_seen", res.AlreadySeen,
	)
	return nil
}

func lockSettle() (func(), error) {
	path := filepath.Join(os.TempDir(), "cigc-buy-settle.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

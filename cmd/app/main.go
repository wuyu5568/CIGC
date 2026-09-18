package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/cigc/app/internal/conf"
	"github.com/cigc/app/internal/data"

	_ "github.com/go-kratos/kratos/v3/encoding/json"
)

var confPath string

func init() {
	flag.StringVar(&confPath, "conf", "configs/config.yaml", "config path")
}

func main() {
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := conf.Load(confPath)
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	if cfg.App.AllowForceSettle {
		slog.Warn("allow_force_settle=true; disable in production")
	}
	if cfg.App.PayoutEnabled {
		slog.Info("payout enabled", "max_usdt", cfg.App.PayoutMaxUSDT, "max_ispay_fallback", cfg.App.PayoutMaxIspay, "hot_key_set", cfg.App.HotWalletKey != "")
	}

	d, cleanupData, err := data.NewData(&cfg.Data)
	if err != nil {
		slog.Error("connect database", "err", err)
		os.Exit(1)
	}
	defer cleanupData()

	kapp, cleanupApp, err := newApp(cfg, d, logger)
	if err != nil {
		slog.Error("wire app", "err", err)
		os.Exit(1)
	}
	defer cleanupApp()

	if err := kapp.Run(); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}

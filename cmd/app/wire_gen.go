//go:build !wireinject
// +build !wireinject

package main

import (
	"log/slog"

	"github.com/cigc/app/internal/biz"
	"github.com/cigc/app/internal/conf"
	"github.com/cigc/app/internal/data"
	"github.com/cigc/app/internal/pkg/middleware/auth"
	"github.com/cigc/app/internal/server"
	"github.com/cigc/app/internal/service"
	"github.com/go-kratos/kratos/v3"
)

func newApp(cfg *conf.Bootstrap, d *data.Data, logger *slog.Logger) (*kratos.App, func(), error) {
	place := biz.NewPlacementUseCase(data.NewUserRepo(d), data.NewPlacementRepo(d))
	users := biz.NewUserUseCase(
		data.NewUserRepo(d),
		data.NewRecommendRepo(d),
		auth.NewSignatureVerifier(),
		auth.NewTokenIssuer(&cfg.Auth),
		&cfg.Auth,
		cfg.App.GenesisAddress,
		place,
	)
	orders := biz.NewOrderUseCase(
		data.NewPackageRepo(d),
		data.NewOrderRepo(d),
		data.NewUserRepo(d),
	)
	settle := biz.NewSettleUseCase(
		data.NewUserRepo(d),
		data.NewPackageRepo(d),
		data.NewSettleRunRepo(d),
		data.NewOrderRepo(d),
		data.NewUserBalanceRepo(d),
		data.NewLedgerRepo(d),
		data.NewConfigRepo(d),
		data.NewPlacementRepo(d),
		data.NewMatchRepo(d),
		d,
		&cfg.App,
	)
	ledger := biz.NewLedgerUseCase(data.NewLedgerRepo(d))
	withdraw := biz.NewWithdrawUseCase(
		data.NewUserRepo(d),
		data.NewUserBalanceRepo(d),
		data.NewLedgerRepo(d),
		data.NewWithdrawRepo(d),
		data.NewConfigRepo(d),
		d,
	)
	svc := service.NewAppService(users, orders, settle, ledger, withdraw, place, d, &cfg.Auth)
	hs := server.NewHTTPServer(cfg, svc)
	settleCron := server.NewSettleCron(&cfg.App, settle)
	return server.NewApp(cfg, hs, settleCron, logger)
}

func provideAuthConf(cfg *conf.Bootstrap) *conf.Auth { return &cfg.Auth }

func provideGenesis(cfg *conf.Bootstrap) string { return cfg.App.GenesisAddress }

func provideAppConf(cfg *conf.Bootstrap) *conf.App { return &cfg.App }

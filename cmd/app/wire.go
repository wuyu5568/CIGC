//go:build wireinject
// +build wireinject

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
	"github.com/google/wire"
)

func newApp(*conf.Bootstrap, *data.Data, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		provideAuthConf,
		provideGenesis,
		auth.NewSignatureVerifier,
		auth.NewTokenIssuer,
		provideAppConf,
		data.NewUserRepo,
		data.NewRecommendRepo,
		data.NewPackageRepo,
		data.NewOrderRepo,
		data.NewSettleRunRepo,
		data.NewLedgerRepo,
		data.NewWithdrawRepo,
		data.NewConfigRepo,
		biz.NewConfigUseCase,
		data.NewUserBalanceRepo,
		data.NewPlacementRepo,
		data.NewMatchRepo,
		data.NewChainCursorRepo,
		data.NewChainDepositRepo,
		data.NewChainReader,
		biz.ProviderSet,
		service.ProviderSet,
		server.ProviderSet,
	))
}

func provideAuthConf(cfg *conf.Bootstrap) *conf.Auth { return &cfg.Auth }

func provideGenesis(cfg *conf.Bootstrap) string { return cfg.App.GenesisAddress }

func provideAppConf(cfg *conf.Bootstrap) *conf.App { return &cfg.App }

package server

import (
	"log/slog"

	"github.com/cigc/app/internal/conf"
	"github.com/go-kratos/kratos/v3"
	khttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/wire"
)

// ProviderSet 是 server 层 Wire 集合。
var ProviderSet = wire.NewSet(NewHTTPServer, NewSettleCron, NewApp)

// NewApp 组装 Kratos 应用并挂载日结 cron。
func NewApp(cfg *conf.Bootstrap, hs *khttp.Server, settle *SettleCron, logger *slog.Logger) (*kratos.App, func(), error) {
	opts := []kratos.Option{
		kratos.Name("cigc-api"),
		kratos.Server(hs),
	}
	if logger != nil {
		opts = append(opts, kratos.Logger(logger))
	}
	app := kratos.New(opts...)
	_ = cfg
	if settle != nil {
		settle.Start()
	}
	cleanup := func() {
		if settle != nil {
			settle.Stop()
		}
	}
	return app, cleanup, nil
}

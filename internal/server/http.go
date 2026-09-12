package server

import (
	stdhttp "net/http"

	"github.com/cigc/app/internal/conf"
	"github.com/cigc/app/internal/pkg/middleware/auth"
	"github.com/cigc/app/internal/service"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	khttp "github.com/go-kratos/kratos/v3/transport/http"
)

// NewHTTPServer 创建 Kratos v3 HTTP 服务并注册手写路由。
func NewHTTPServer(cfg *conf.Bootstrap, svc *service.AppService) *khttp.Server {
	var opts []khttp.ServerOption
	opts = append(opts, khttp.Middleware(recovery.Recovery()))
	opts = append(opts, khttp.Filter(corsFilter()))
	if cfg.Server.HTTP.Addr != "" {
		opts = append(opts, khttp.Address(cfg.Server.HTTP.Addr))
	}
	if cfg.Server.HTTP.Timeout > 0 {
		opts = append(opts, khttp.Timeout(cfg.Server.HTTP.Timeout))
	}
	srv := khttp.NewServer(opts...)
	registerHTTPRoutes(srv, cfg, svc)
	return srv
}

func corsFilter() func(stdhttp.Handler) stdhttp.Handler {
	return func(next stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = "*"
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == stdhttp.MethodOptions {
				w.WriteHeader(stdhttp.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func registerHTTPRoutes(srv *khttp.Server, cfg *conf.Bootstrap, svc *service.AppService) {
	jwt := cfg.Auth.JWTKey
	srv.Handle("/health", stdhttp.HandlerFunc(svc.Health))
	srv.Handle("/api/health", stdhttp.HandlerFunc(svc.Health))
	srv.Handle("/api/app_server/eth_authorize", stdhttp.HandlerFunc(svc.CompatEthAuthorize))
	srv.Handle("/api/app_server/user_info", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatUserInfo)))
	srv.Handle("/api/app_server/package_list", stdhttp.HandlerFunc(svc.CompatPackageList))
	srv.Handle("/api/app_server/buy", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatBuy)))
	srv.Handle("/api/app_server/order_list", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatOrderList)))
	srv.Handle("/api/app_server/reward_list", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatRewardList)))
	srv.Handle("/api/app_server/withdraw", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatWithdraw)))
	srv.Handle("/api/app_server/withdraw_list", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatWithdrawList)))
	registerAdminRoutes(srv, "/api/admin_cigc", jwt, svc)
	registerAdminRoutes(srv, "/api/admin_dhb", jwt, svc)
}

func registerAdminRoutes(srv *khttp.Server, prefix, jwt string, svc *service.AppService) {
	srv.Handle(prefix+"/login", stdhttp.HandlerFunc(svc.CompatAdminLogin))
	srv.Handle(prefix+"/order_pay", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminMarkPaid)))
	srv.Handle(prefix+"/settle", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminSettle)))
	srv.Handle(prefix+"/settle_status", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminSettleStatus)))
	srv.Handle(prefix+"/reward_list", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminRewardList)))
	srv.Handle(prefix+"/buy_list", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminBuyList)))
	srv.Handle(prefix+"/user_list", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminUserList)))
	srv.Handle(prefix+"/lock_user", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminLockUser)))
	srv.Handle(prefix+"/unlock_user", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminUnlockUser)))
	srv.Handle(prefix+"/sub_money", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminSubMoney)))
	srv.Handle(prefix+"/withdraw_list", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminWithdrawList)))
	srv.Handle(prefix+"/withdraw_pass", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminWithdrawPass)))
	srv.Handle(prefix+"/withdraw_reject", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminWithdrawReject)))
	srv.Handle(prefix+"/placement", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.Method == stdhttp.MethodGet {
			svc.CompatAdminPlacementGet(w, r)
			return
		}
		svc.CompatAdminPlacement(w, r)
	})))
}

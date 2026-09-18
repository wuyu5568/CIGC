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
	srv.Handle("/api/app_server/ispay_price", stdhttp.HandlerFunc(svc.CompatIspayPrice))
	srv.Handle("/api/app_server/buy", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatBuy)))
	srv.Handle("/api/app_server/deposit_list", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatDepositList)))
	srv.Handle("/api/app_server/order_list", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatOrderList)))
	srv.Handle("/api/app_server/reward_list", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatRewardList)))
	srv.Handle("/api/app_server/withdraw", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatWithdraw)))
	srv.Handle("/api/app_server/withdraw_cancel", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatWithdrawCancel)))
	srv.Handle("/api/app_server/withdraw_list", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatWithdrawList)))
	srv.Handle("/api/app_server/recommend_list", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatRecommendList)))
	srv.Handle("/api/app_server/downline", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatUserDownline)))
	srv.Handle("/api/app_server/shipping_address", stdhttp.HandlerFunc(auth.RequireJWT(jwt, svc.CompatShippingAddress)))
	registerAdminRoutes(srv, "/api/admin", jwt, svc)
	registerAdminRoutes(srv, "/api/admin_cigc", jwt, svc)
	registerAdminRoutes(srv, "/api/admin_dhb", jwt, svc)
}

func registerAdminRoutes(srv *khttp.Server, prefix, jwt string, svc *service.AppService) {
	srv.Handle(prefix+"/login", stdhttp.HandlerFunc(svc.CompatAdminLogin))
	srv.Handle(prefix+"/order_pay", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminMarkPaid)))
	srv.Handle(prefix+"/settle", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminSettle)))
	srv.Handle(prefix+"/settle_status", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminSettleStatus)))
	srv.Handle(prefix+"/settle_reset", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminSettleReset)))
	srv.Handle(prefix+"/test_data_clear", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminTestDataClear)))
	srv.Handle(prefix+"/deposit_scan", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminDepositScan)))
	srv.Handle(prefix+"/reward_list", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminRewardList)))
	srv.Handle(prefix+"/freeze_assets", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminFreezeAssets)))
	srv.Handle(prefix+"/freeze_release", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminFreezeRelease)))
	srv.Handle(prefix+"/buy_list", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminBuyList)))
	srv.Handle(prefix+"/user_list", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminUserList)))
	srv.Handle(prefix+"/lock_user", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminLockUser)))
	srv.Handle(prefix+"/unlock_user", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminUnlockUser)))
	srv.Handle(prefix+"/sub_money", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminSubMoney)))
	srv.Handle(prefix+"/add_money_two", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminAddMoneyTwo)))
	srv.Handle(prefix+"/add_money_three", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminAddMoneyThree)))
	srv.Handle(prefix+"/record_list", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminRecordList)))
	srv.Handle(prefix+"/set_ispay", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminSetIspay)))
	srv.Handle(prefix+"/add_lock", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminAddLock)))
	srv.Handle(prefix+"/add_lock_ispay", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminAddLockIspay)))
	srv.Handle(prefix+"/adjust_balance", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminAdjustBalance)))
	srv.Handle(prefix+"/all", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminAll)))
	srv.Handle(prefix+"/withdraw_list", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminWithdrawList)))
	srv.Handle(prefix+"/withdraw_pass", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminWithdrawPass)))
	srv.Handle(prefix+"/withdraw_reject", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminWithdrawReject)))
	srv.Handle(prefix+"/withdraw_payout", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminWithdrawPayout)))
	srv.Handle(prefix+"/placement", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.Method == stdhttp.MethodGet {
			svc.CompatAdminPlacementGet(w, r)
			return
		}
		svc.CompatAdminPlacement(w, r)
	})))
	srv.Handle(prefix+"/recommend_list", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminRecommendList)))
	srv.Handle(prefix+"/downline", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminDownline)))
	srv.Handle(prefix+"/config", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminConfig)))
	srv.Handle(prefix+"/config_update", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminConfigUpdate)))
	srv.Handle(prefix+"/daily_cap_tiers", stdhttp.HandlerFunc(auth.RequireAdminOrUserGET(jwt, svc.CompatDailyCapTiers)))
	srv.Handle(prefix+"/daily_cap_tiers_update", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminDailyCapTiersUpdate)))
	srv.Handle(prefix+"/package_list", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminPackageList)))
	srv.Handle(prefix+"/package_create", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminPackageCreate)))
	srv.Handle(prefix+"/package_update", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminPackageUpdate)))
	srv.Handle(prefix+"/package_delete", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminPackageDelete)))
	srv.Handle(prefix+"/web3_goods", stdhttp.HandlerFunc(auth.RequireAdminOrUserGET(jwt, svc.AdminWeb3GoodsList)))
	srv.Handle(prefix+"/web3_goods_detail", stdhttp.HandlerFunc(auth.RequireAdminOrUserGET(jwt, svc.AdminWeb3GoodsDetail)))
	srv.Handle(prefix+"/web3_goods_create", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.AdminWeb3GoodsCreate)))
	srv.Handle(prefix+"/web3_goods_update", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.AdminWeb3GoodsUpdate)))
	srv.Handle(prefix+"/web3_goods_status", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.AdminWeb3GoodsStatus)))
	srv.Handle(prefix+"/web3_goods_delete", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.AdminWeb3GoodsDelete)))
	srv.Handle(prefix+"/web3_goods_sort", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.AdminWeb3GoodsSort)))
	srv.Handle(prefix+"/web3_goods_image_upload", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.AdminWeb3GoodsImageUpload)))
	srv.Handle(prefix+"/my_auth_list", stdhttp.HandlerFunc(auth.RequireAdminJWT(jwt, svc.CompatAdminMyAuthList)))
}

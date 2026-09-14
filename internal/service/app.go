package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/cigc/app/internal/biz"
	"github.com/cigc/app/internal/conf"
	"github.com/cigc/app/internal/data"
	"github.com/cigc/app/internal/pkg/middleware/auth"
	"github.com/cigc/app/internal/pkg/money"
	"github.com/cigc/app/internal/pkg/wallet"
	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/wire"
	"github.com/shopspring/decimal"
)

// ProviderSet 是 service 层 Wire 集合。
var ProviderSet = wire.NewSet(NewAppService)

// AppService 把 HTTP 转成 biz 调用，不含业务规则。
type AppService struct {
	users    *biz.UserUseCase
	orders   *biz.OrderUseCase
	settle   *biz.SettleUseCase
	ledger   *biz.LedgerUseCase
	withdraw *biz.WithdrawUseCase
	place    *biz.PlacementUseCase
	deposit  *biz.DepositUseCase
	configs  *biz.ConfigUseCase
	adjust   *biz.AdjustUseCase
	stats    *biz.StatsUseCase
	ping     *data.Data
	auth     *conf.Auth
	app      *conf.App
}

// NewAppService 构造 HTTP 适配器。
func NewAppService(users *biz.UserUseCase, orders *biz.OrderUseCase, settle *biz.SettleUseCase, ledger *biz.LedgerUseCase, withdraw *biz.WithdrawUseCase, place *biz.PlacementUseCase, deposit *biz.DepositUseCase, configs *biz.ConfigUseCase, adjust *biz.AdjustUseCase, stats *biz.StatsUseCase, ping *data.Data, auth *conf.Auth, app *conf.App) *AppService {
	return &AppService{users: users, orders: orders, settle: settle, ledger: ledger, withdraw: withdraw, place: place, deposit: deposit, configs: configs, adjust: adjust, stats: stats, ping: ping, auth: auth, app: app}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return
	}
}

func writeBizError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	var ke *kerrors.Error
	if errors.As(err, &ke) {
		writeJSON(w, int(ke.Code), map[string]string{"message": ke.Message, "reason": ke.Reason})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func decStr(d decimal.Decimal) string {
	return money.Display(d)
}

func parseAmount(s string) (decimal.Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero, biz.ErrInvalidAmount
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, biz.ErrInvalidAmount
	}
	return d, nil
}

func parseUint(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, biz.ErrInvalidAmount
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (s *AppService) Health(w http.ResponseWriter, r *http.Request) {
	if err := s.ping.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status":  "degraded",
			"db":      "fail",
			"message": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"db":     "ok",
	})
}

func (s *AppService) CompatEthAuthorize(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Address   string `json:"address"`
		Code      string `json:"code"`
		Sign      string `json:"sign"`
		Signature string `json:"signature"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "fail"})
		return
	}
	sign := body.Sign
	if sign == "" {
		sign = body.Signature
	}
	res, err := s.users.EthAuthorize(r.Context(), body.Address, sign, body.Code)
	if err != nil {
		switch {
		case errors.Is(err, biz.ErrUserDisabled):
			writeJSON(w, http.StatusOK, map[string]string{"status": "用户已锁定"})
		case errors.Is(err, biz.ErrInviteRequired):
			writeJSON(w, http.StatusOK, map[string]string{"status": "请输入推荐码"})
		case errors.Is(err, biz.ErrInviteInvalid):
			writeJSON(w, http.StatusOK, map[string]string{"status": "无效的推荐码"})
		case errors.Is(err, biz.ErrInvalidSignature):
			writeJSON(w, http.StatusOK, map[string]string{"status": "签名失败"})
		default:
			writeJSON(w, http.StatusOK, map[string]string{"status": "fail"})
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "token": res.Token})
}

func (s *AppService) CompatUserInfo(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeBizError(w, biz.ErrUnauthorized)
		return
	}
	user, err := s.users.GetProfile(r.Context(), uid)
	if err != nil {
		writeBizError(w, err)
		return
	}
	inviteAddr, err := s.users.InviterAddress(r.Context(), user)
	if err != nil {
		writeBizError(w, err)
		return
	}
	pkgs, err := s.orders.ListPackages(r.Context())
	if err != nil {
		writeBizError(w, err)
		return
	}
	goods := make([]map[string]any, 0, len(pkgs))
	for _, p := range pkgs {
		goods = append(goods, packageJSON(p))
	}
	avail := decStr(user.AvailableBalance)
	recharge := decStr(user.RechargeBalance)
	vol, err := s.place.TeamVolumeOf(r.Context(), user.ID)
	if err != nil {
		writeBizError(w, err)
		return
	}
	unlock := biz.ComputeLockUnlock(user, false)
	if s.settle != nil {
		unlock, err = s.settle.PreviewUserLockUnlock(r.Context(), user)
		if err != nil {
			writeBizError(w, err)
			return
		}
	}
	var limits biz.WithdrawLimits
	if s.withdraw != nil {
		limits = s.withdraw.UserLimits(r.Context(), user.ID)
	}
	stats, err := s.userAssetStats(r.Context(), user.ID)
	if err != nil {
		writeBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":            "ok",
		"address":           user.Address,
		"level":             "0",
		"usdt":              avail,
		"raw":               avail,
		"amountGet":         stats["amountGet"],
		"amountUsdt":        recharge,
		"rechargeBalance":   recharge,
		"inviteUserAddress": inviteAddr,
		"withdrawRate":      decStr(limits.Rate),
		"withdrawMin":       decStr(limits.Min),
		"withdrawRateTwo":   "0",
		"withdrawMinTwo":    "0",
		"withdrawDaily":     decStr(limits.Daily),
		"withdrawToday":     decStr(limits.Today),
		"withdrawRemain":    decStr(limits.Remain),
		"withdrawDailyTwo":  decStr(limits.DailyTwo),
		"withdrawTodayTwo":  decStr(limits.TodayTwo),
		"withdrawRemainTwo": decStr(limits.RemainTwo),
		"locationNum":       "0",
		"LocationList":      []any{},
		"total":             decStr(vol.Total),
		"max":               decStr(vol.Max),
		"min":               decStr(vol.Min),
		"buy":               decStr(user.PaidAmount),
		"amountGetSub":      stats["amountGetSub"],
		"outNum":            stats["outNum"],
		"pendingStatic":     stats["pendingStatic"],
		"releasedStatic":    stats["releasedStatic"],
		"releaseCount":      stats["releaseCount"],
		"location":          stats["location"],
		"recommend":         stats["recommend"],
		"recommendTwo":      stats["recommendTwo"],
		"team":              stats["team"],
		"staticTotal":       stats["staticTotal"],
		"directTotal":       stats["directTotal"],
		"matchTotal":        stats["matchTotal"],
		"manageTotal":       stats["manageTotal"],
		"teamTwo":           "0",
		"all":               avail,
		"notice":            "",
		"goods":             goods,
		"capEffective":      decStr(user.CapEffective),
		"frozen":            decStr(user.FrozenBalance),
		"frozenIspay":       decStr(user.FrozenIspay),
		"lock":              decStr(user.LockBalance),
		"lockBalance":       decStr(user.LockBalance),
		"lockIspay":         decStr(user.LockIspay),
		"unlockToday":       decStr(unlock.UnlockTodayUSDT),
		"unlockTodayIspay":  decStr(unlock.UnlockTodayIspay),
		"todayLockReleased": unlock.TodayReleased,
		"activated":         user.IsActivated(),
		"ispay":             decStr(user.IspayBalance),
		"ispayAmount":       decStr(user.IspayBalance),
		"ispayPrice":        decStr(s.ispaySpot(r.Context())),
		"receive_addresses": s.receiveJSON(decimal.Zero),
		"release_tiers":     releaseTiersJSON(),
		"buy_contract":      s.buyContract(),
	})
}

func (s *AppService) userAssetStats(ctx context.Context, userID uint64) (map[string]any, error) {
	pending := decimal.Zero
	released := decimal.Zero
	if s.orders != nil && s.settle != nil {
		rows, err := s.orders.ListOrders(ctx, userID)
		if err != nil {
			return nil, err
		}
		rel, err := s.settle.OrderStaticReleases(ctx, rows)
		if err != nil {
			return nil, err
		}
		for _, o := range rows {
			if o == nil {
				continue
			}
			it := rel[o.ID]
			pending = pending.Add(it.PendingUSDT)
			released = released.Add(it.ReleasedUSDT)
		}
	}
	staticT, direct, match, manage := decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	times := 0
	if s.ledger != nil {
		var err error
		staticT, direct, match, manage, times, err = s.ledger.UserRewardTotals(ctx, userID)
		if err != nil {
			return nil, err
		}
	}
	return map[string]any{
		"amountGetSub":   decStr(money.Round(pending)),
		"amountGet":      decStr(money.Round(released)),
		"outNum":         strconv.Itoa(times),
		"pendingStatic":  decStr(money.Round(pending)),
		"releasedStatic": decStr(money.Round(released)),
		"releaseCount":   strconv.Itoa(times),
		"location":       decStr(staticT),
		"recommend":      decStr(direct),
		"recommendTwo":   decStr(match),
		"team":           decStr(manage),
		"staticTotal":    decStr(staticT),
		"directTotal":    decStr(direct),
		"matchTotal":     decStr(match),
		"manageTotal":    decStr(manage),
	}, nil
}

func (s *AppService) CompatPackageList(w http.ResponseWriter, r *http.Request) {
	pkgs, err := s.orders.ListPackages(r.Context())
	if err != nil {
		writeBizError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(pkgs))
	for _, p := range pkgs {
		items = append(items, packageJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "items": items, "release_tiers": releaseTiersJSON(), "ispay_price": decStr(s.ispaySpot(r.Context())), "buy_contract": s.buyContract()})
}

func packageJSON(p *biz.Package) map[string]any {
	if p == nil {
		return map[string]any{}
	}
	days := p.ReleaseDays
	if !biz.ValidReleaseDays(days) {
		days = biz.ReleaseDays300
	}
	return map[string]any{
		"id":           p.ID,
		"amount":       decStr(p.Amount),
		"title":        p.Title,
		"goods":        p.GoodsDesc,
		"name":         p.Title,
		"one":          p.GoodsDesc,
		"dailyCap":     decStr(p.DailyCap),
		"daily_cap":    decStr(p.DailyCap),
		"release_days": days,
		"days":         days,
		"sort_order":   p.SortOrder,
		"sortOrder":    p.SortOrder,
		"enabled":      p.Enabled,
		"status":       enabledStatus(p.Enabled),
	}
}

func enabledStatus(on bool) string {
	if on {
		return "1"
	}
	return "0"
}

func (s *AppService) CompatBuy(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeBizError(w, biz.ErrUnauthorized)
		return
	}
	var body struct {
		Amount      json.RawMessage `json:"amount"`
		Days        json.RawMessage `json:"days"`
		ReleaseDays json.RawMessage `json:"release_days"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "fail"})
		return
	}
	amountStr := strings.Trim(string(body.Amount), `"`)
	amount, err := parseAmount(amountStr)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "金额错误"})
		return
	}
	_ = r.ParseForm()
	days, err := parseReleaseDays(firstNonEmpty(strings.Trim(string(body.ReleaseDays), `"`), strings.Trim(string(body.Days), `"`), r.Form.Get("release_days"), r.Form.Get("days")))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "请选择释放天数"})
		return
	}
	o, err := s.orders.BuyWithRecharge(r.Context(), uid, amount, days)
	if err != nil {
		switch {
		case errors.Is(err, biz.ErrPackageNotFound), errors.Is(err, biz.ErrPackageDisabled):
			writeJSON(w, http.StatusOK, map[string]string{"status": "套餐不存在"})
		case errors.Is(err, biz.ErrInvalidAmount):
			writeJSON(w, http.StatusOK, map[string]string{"status": "金额错误"})
		case errors.Is(err, biz.ErrInvalidReleaseDays):
			writeJSON(w, http.StatusOK, map[string]string{"status": "请选择释放天数"})
		case errors.Is(err, biz.ErrInsufficientBalance):
			writeJSON(w, http.StatusOK, map[string]string{"status": "充值余额不足"})
		default:
			writeJSON(w, http.StatusOK, map[string]string{"status": "fail"})
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "ok",
		"order_id":        o.ID,
		"amount":          decStr(o.Amount),
		"release_days":    o.ReleaseDays,
		"order_status":    o.Status,
		"rechargeBalance": decStr(s.userRecharge(r.Context(), uid)),
		"buy_contract":    s.buyContract(),
	})
}

func (s *AppService) userRecharge(ctx context.Context, uid uint64) decimal.Decimal {
	if s == nil || s.users == nil {
		return decimal.Zero
	}
	u, err := s.users.GetProfile(ctx, uid)
	if err != nil || u == nil {
		return decimal.Zero
	}
	return u.RechargeBalance
}

func (s *AppService) CompatDepositList(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeBizError(w, biz.ErrUnauthorized)
		return
	}
	page, err := s.deposit.ListUser(r.Context(), uid, parsePage(r))
	if err != nil {
		writeBizError(w, err)
		return
	}
	list := make([]map[string]any, 0, len(page.Items))
	for _, d := range page.Items {
		list = append(list, depositJSON(d))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"list":   list,
		"count":  strconv.Itoa(page.Total),
	})
}

func depositJSON(d *biz.ChainDeposit) map[string]any {
	if d == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":          d.ID,
		"address":     d.FromAddr,
		"amount":      decStr(d.Amount),
		"status":      d.Status,
		"txHash":      d.TxHash,
		"toAddr":      d.ToAddr,
		"one":         d.Remark,
		"remark":      d.Remark,
		"blockNumber": d.BlockNumber,
		"createdAt":   d.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func parseReleaseDays(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		return 0, biz.ErrInvalidReleaseDays
	}
	n, err := strconv.Atoi(s)
	if err != nil || !biz.ValidReleaseDays(n) {
		return 0, biz.ErrInvalidReleaseDays
	}
	return n, nil
}

func releaseTiersJSON() []map[string]any {
	return []map[string]any{
		{"days": 300, "price": "1200"},
		{"days": 600, "price": "1000"},
		{"days": 750, "price": "800"},
	}
}

func (s *AppService) CompatIspayPrice(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"price":  decStr(s.ispaySpot(r.Context())),
		"source": "business_configs",
	})
}

func (s *AppService) ispaySpot(ctx context.Context) decimal.Decimal {
	if s != nil && s.configs != nil {
		return s.configs.Spot(ctx)
	}
	return biz.IspaySpotFallback()
}

func (s *AppService) buyContract() string {
	if s == nil || s.app == nil {
		return ""
	}
	return wallet.NormalizeOrEmpty(s.app.BuyContract)
}

func (s *AppService) receiveJSON(amount decimal.Decimal) []map[string]any {
	if s == nil || s.deposit == nil {
		return []map[string]any{}
	}
	var items []ReceivePayView
	if amount.IsPositive() {
		items = payItems(s.deposit.PayPlan(amount))
	} else {
		items = shareItems(s.deposit.Shares())
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		row := map[string]any{
			"address": it.Address,
			"percent": it.Percent,
		}
		if it.HasAmount {
			row["amount"] = it.Amount
		}
		out = append(out, row)
	}
	return out
}

type ReceivePayView struct {
	Address   string
	Percent   string
	Amount    string
	HasAmount bool
}

func shareItems(shares []biz.ReceiveShare) []ReceivePayView {
	out := make([]ReceivePayView, 0, len(shares))
	for _, s := range shares {
		out = append(out, ReceivePayView{
			Address: s.Address,
			Percent: s.Percent.String(),
		})
	}
	return out
}

func payItems(plan []biz.ReceivePayItem) []ReceivePayView {
	out := make([]ReceivePayView, 0, len(plan))
	for _, it := range plan {
		out = append(out, ReceivePayView{
			Address:   it.Address,
			Percent:   it.Percent.String(),
			Amount:    decStr(it.Amount),
			HasAmount: true,
		})
	}
	return out
}

func (s *AppService) CompatOrderList(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeBizError(w, biz.ErrUnauthorized)
		return
	}
	rows, err := s.orders.ListOrders(r.Context(), uid)
	if err != nil {
		writeBizError(w, err)
		return
	}
	releases := map[uint64]biz.OrderStaticRelease{}
	if s.settle != nil {
		releases, err = s.settle.OrderStaticReleases(r.Context(), rows)
		if err != nil {
			writeBizError(w, err)
			return
		}
	}
	items := make([]map[string]any, 0, len(rows))
	for _, o := range rows {
		rel := releases[o.ID]
		item := map[string]any{
			"id":             o.ID,
			"order_no":       o.DisplayNo(),
			"amount":         decStr(o.Amount),
			"title":          o.TitleSnapshot,
			"goods":          o.GoodsSnapshot,
			"status":         o.Status,
			"release_days":   o.ReleaseDays,
			"buy_contract":   s.buyContract(),
			"today_usdt":     decStr(rel.TodayUSDT),
			"today_ispay":    decStr(rel.TodayIspay),
			"released_usdt":  decStr(rel.ReleasedUSDT),
			"released_ispay": decStr(rel.ReleasedIspay),
			"pending_usdt":   decStr(rel.PendingUSDT),
			"pending_ispay":  decStr(rel.PendingIspay),
			"settle_date":    rel.SettleDate,
		}
		item["created_at"] = o.CreatedAt.Format("2006-01-02 15:04:05")
		item["createdAt"] = o.CreatedAt.Format("2006-01-02 15:04:05")
		if o.PaidAt != nil {
			paid := o.PaidAt.Format("2006-01-02 15:04:05")
			item["paid_at"] = paid
			item["purchase_date"] = paid
		} else {
			item["purchase_date"] = o.CreatedAt.Format("2006-01-02 15:04:05")
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *AppService) CompatAdminLogin(w http.ResponseWriter, r *http.Request) {
	username, password, err := readAdminLogin(r)
	if err != nil {
		writeBizError(w, biz.ErrUnauthorized)
		return
	}
	token, err := s.users.AdminLogin(r.Context(), username, password)
	if err != nil {
		writeBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "status": "ok"})
}

func readAdminLogin(r *http.Request) (username, password string, err error) {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Account  string `json:"account"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return "", "", err
		}
		return firstNonEmpty(body.Username, body.Account), body.Password, nil
	}
	if err := r.ParseForm(); err != nil {
		return "", "", err
	}
	return firstNonEmpty(r.Form.Get("username"), r.Form.Get("account")), r.Form.Get("password"), nil
}

func (s *AppService) CompatAdminMarkPaid(w http.ResponseWriter, r *http.Request) {
	id, err := readOrderID(r)
	if err != nil || id == 0 {
		writeBizError(w, biz.ErrInvalidAmount)
		return
	}
	o, err := s.orders.MarkPaid(r.Context(), id)
	if err != nil {
		writeBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"order_id": o.ID,
		"paid":     decStr(o.Amount),
	})
}

func readOrderID(r *http.Request) (uint64, error) {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		var body struct {
			ID json.RawMessage `json:"id"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return 0, err
		}
		return parseUint(strings.Trim(string(body.ID), `"`))
	}
	if err := r.ParseForm(); err != nil {
		return 0, err
	}
	return parseUint(r.Form.Get("id"))
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func (s *AppService) CompatAdminSettle(w http.ResponseWriter, r *http.Request) {
	force := readForce(r)
	res, err := s.settle.Run(r.Context(), force)
	if err != nil {
		writeBizError(w, err)
		return
	}
	status := "ok"
	if res.Skipped {
		status = "already_settled"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       status,
		"forced":       res.Forced,
		"settle_date":  res.SettleDate,
		"user_count":   res.UserCount,
		"cap_updated":  res.CapUpdated,
		"direct_count": res.DirectCount,
		"match_count":  res.MatchCount,
		"manage_count": res.ManageCount,
		"warning":      settleForceWarning(res),
	})
}

func (s *AppService) CompatAdminDepositScan(w http.ResponseWriter, r *http.Request) {
	if s.deposit == nil || !s.deposit.Enabled() {
		writeJSON(w, http.StatusOK, map[string]any{"status": "disabled", "enabled": false})
		return
	}
	res, err := s.deposit.Scan(r.Context())
	if err != nil {
		writeBizError(w, err)
		return
	}
	status := "ok"
	if res.Skipped {
		status = "skipped"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       status,
		"enabled":      true,
		"from_block":   res.FromBlock,
		"to_block":     res.ToBlock,
		"head_block":   res.HeadBlock,
		"seen":         res.Seen,
		"matched":      res.Matched,
		"abnormal":     res.Abnormal,
		"already_seen": res.AlreadySeen,
	})
}

func (s *AppService) CompatAdminSettleStatus(w http.ResponseWriter, r *http.Request) {
	st, err := s.settle.Status(r.Context())
	if err != nil {
		writeBizError(w, err)
		return
	}
	out := map[string]any{
		"status":             "ok",
		"settle_today":       st.TodayDate,
		"settle_today_done":  st.TodaySettled,
		"settle_next_test":   st.NextTestDate,
		"allow_force_settle": st.AllowForce,
	}
	if st.Today != nil {
		out["today"] = settleRunJSON(st.Today)
	}
	if st.Latest != nil {
		out["latest"] = settleRunJSON(st.Latest)
	}
	writeJSON(w, http.StatusOK, out)
}

func settleRunJSON(run *biz.SettleRun) map[string]any {
	return map[string]any{
		"settle_date":  run.SettleDate.Format("2006-01-02"),
		"forced":       run.Forced,
		"user_count":   run.UserCount,
		"cap_updated":  run.CapUpdated,
		"direct_count": run.DirectCount,
		"match_count":  run.MatchCount,
		"manage_count": run.ManageCount,
		"remark":       run.Remark,
	}
}

func settleForceWarning(res *biz.SettleResult) string {
	if res != nil && res.Forced {
		return "force=1 settles the next day; test only"
	}
	return ""
}

func readForce(r *http.Request) bool {
	if q := r.URL.Query().Get("force"); isTruthyFlag(q) {
		return true
	}
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		var body struct {
			Force json.RawMessage `json:"force"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return false
		}
		return isTruthyFlag(strings.Trim(string(body.Force), `"`))
	}
	if err := r.ParseForm(); err != nil {
		return false
	}
	return isTruthyFlag(r.Form.Get("force"))
}

func isTruthyFlag(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "1" || s == "true" || s == "yes"
}

func parsePage(r *http.Request) int {
	raw := strings.TrimSpace(r.URL.Query().Get("page"))
	if raw == "" {
		return 1
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

func (s *AppService) CompatRewardList(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeBizError(w, biz.ErrUnauthorized)
		return
	}
	page, err := s.ledger.ListUserRewards(r.Context(), uid, r.URL.Query().Get("reqType"), parsePage(r))
	if err != nil {
		writeBizError(w, err)
		return
	}
	list := make([]map[string]any, 0, len(page.Items))
	for _, it := range page.Items {
		list = append(list, map[string]any{
			"id":          it.ID,
			"amount":      it.Amount,
			"amountTwo":   it.AmountTwo,
			"reward":      it.Amount,
			"name":        it.Name,
			"address":     it.Address,
			"num":         it.Num,
			"reason":      it.Reason,
			"orderNo":     it.OrderNo,
			"orderTitle":  it.OrderTitle,
			"orderAmount": it.OrderAmount,
			"settleDate":  it.SettleDate,
			"detail":      it.Detail,
			"createdAt":   it.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count": page.Total,
		"list":  list,
	})
}

func (s *AppService) CompatAdminRewardList(w http.ResponseWriter, r *http.Request) {
	page, err := s.ledger.ListAdminRewards(
		r.Context(),
		strings.TrimSpace(r.URL.Query().Get("address")),
		strings.TrimSpace(r.URL.Query().Get("reason")),
		parsePage(r),
	)
	if err != nil {
		writeBizError(w, err)
		return
	}
	rewards := make([]map[string]any, 0, len(page.Items))
	for _, it := range page.Items {
		rewards = append(rewards, map[string]any{
			"id":            it.ID,
			"amount":        it.Amount,
			"amountTwo":     it.AmountTwo,
			"reward":        it.Amount,
			"name":          it.Name,
			"category":      it.Category,
			"address":       it.Address,
			"num":           it.Num,
			"reason":        it.Reason,
			"remark":        it.Remark,
			"detail":        it.Detail,
			"balance":       it.BalanceKind,
			"balanceName":   it.BalanceName,
			"orderId":       it.OrderID,
			"orderNo":       it.OrderNo,
			"orderAmount":   it.OrderAmount,
			"orderTitle":    it.OrderTitle,
			"orderSource":   it.OrderSource,
			"sourceAddress": it.SourceAddress,
			"settleDate":    it.SettleDate,
			"createdAt":     it.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"rewards": rewards,
		"count":   strconv.Itoa(page.Total),
	})
}

func zeroStr() string { return "0" }

func (s *AppService) CompatAdminBuyList(w http.ResponseWriter, r *http.Request) {
	page, err := s.orders.ListAdminOrders(
		r.Context(),
		strings.TrimSpace(r.URL.Query().Get("address")),
		strings.TrimSpace(r.URL.Query().Get("status")),
		parsePage(r),
	)
	if err != nil {
		writeBizError(w, err)
		return
	}
	rewards := make([]map[string]any, 0, len(page.Items))
	for _, o := range page.Items {
		rewards = append(rewards, map[string]any{
			"id":           o.ID,
			"orderNo":      o.DisplayNo(),
			"amount":       decStr(o.Amount),
			"address":      o.Address,
			"createdAt":    o.CreatedAt.Format("2006-01-02 15:04:05"),
			"status":       o.Status,
			"one":          o.TitleSnapshot,
			"title":        o.TitleSnapshot,
			"goods":        o.GoodsSnapshot,
			"release_days": o.ReleaseDays,
			"two":          "",
			"three":        "",
			"four":         "",
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"rewards": rewards,
		"count":   strconv.Itoa(page.Total),
	})
}

func (s *AppService) CompatAdminUserList(w http.ResponseWriter, r *http.Request) {
	page, err := s.users.ListAdminUsers(r.Context(), strings.TrimSpace(r.URL.Query().Get("address")), parsePage(r))
	if err != nil {
		writeBizError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(page.Items))
	for _, u := range page.Items {
		lock := "0"
		if u.IsDisabled() {
			lock = "1"
		}
		avail := decStr(u.AvailableBalance)
		paid := decStr(u.PaidAmount)
		unlock := biz.ComputeLockUnlock(&u.User, false)
		if s.settle != nil {
			if preview, err := s.settle.PreviewUserLockUnlock(r.Context(), &u.User); err == nil {
				unlock = preview
			}
		}
		out = append(out, map[string]any{
			"userId":             strconv.FormatUint(u.ID, 10),
			"createdAt":          u.CreatedAt.Format("2006-01-02 15:04:05"),
			"address":            u.Address,
			"amountUsdtCurrent":  paid,
			"bAmount":            zeroStr(),
			"amountUsdtGet":      avail,
			"amountUsdtTwo":      decStr(u.RechargeBalance),
			"rechargeBalance":    decStr(u.RechargeBalance),
			"balanceUsdt":        avail,
			"balanceDhb":         zeroStr(),
			"bAmountTwo":         zeroStr(),
			"perDayAmount":       zeroStr(),
			"out":                "0",
			"areaTeam":           zeroStr(),
			"areaMax":            zeroStr(),
			"areaTotal":          zeroStr(),
			"areaMin":            zeroStr(),
			"vip":                "0",
			"vipLocked":          "0",
			"historyRecommend":   "0",
			"lock":               lock,
			"lockReward":         "0",
			"lockBalance":        decStr(u.LockBalance),
			"lockIspay":          decStr(u.LockIspay),
			"activated":          u.IsActivated(),
			"myRecommendAddress": u.InviterAddress,
			"capEffective":       decStr(u.CapEffective),
			"unlockToday":        decStr(unlock.UnlockTodayUSDT),
			"unlockTodayIspay":   decStr(unlock.UnlockTodayIspay),
			"todayLockReleased":  unlock.TodayReleased,
			"frozen":             decStr(u.FrozenBalance),
			"frozenIspay":        decStr(u.FrozenIspay),
			"ispay":              decStr(u.IspayBalance),
			"paidAmount":         paid,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"users": out,
		"count": strconv.Itoa(page.Total),
	})
}

func (s *AppService) CompatAdminLockUser(w http.ResponseWriter, r *http.Request) {
	userID, lockStr, line, err := readLockParams(r)
	if err != nil || userID == 0 || lockStr == "" {
		writeBizError(w, biz.ErrInvalidAmount)
		return
	}
	if line {
		writeBizError(w, biz.ErrLineLockUnsupported)
		return
	}
	locked := isTruthyFlag(lockStr)
	if err := s.users.SetUserLock(r.Context(), userID, locked); err != nil {
		writeBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *AppService) CompatAdminUnlockUser(w http.ResponseWriter, r *http.Request) {
	userID, err := readUserIDParam(r)
	if err != nil || userID == 0 {
		writeBizError(w, biz.ErrInvalidAmount)
		return
	}
	if err := s.users.UnlockUser(r.Context(), userID); err != nil {
		writeBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *AppService) CompatAdminSubMoney(w http.ResponseWriter, r *http.Request) {
	_ = r
	writeBizError(w, biz.ErrUnsupportedOperation)
}

func (s *AppService) CompatAdminAddMoneyTwo(w http.ResponseWriter, r *http.Request) {
	s.writeAdminAdjust(w, r, biz.AdjustAvailable)
}

func (s *AppService) CompatAdminSetIspay(w http.ResponseWriter, r *http.Request) {
	s.writeAdminAdjust(w, r, biz.AdjustIspay)
}

func (s *AppService) CompatAdminAddLock(w http.ResponseWriter, r *http.Request) {
	s.writeAdminAdjust(w, r, biz.AdjustLock)
}

func (s *AppService) CompatAdminAddLockIspay(w http.ResponseWriter, r *http.Request) {
	s.writeAdminAdjust(w, r, biz.AdjustLockIspay)
}

func (s *AppService) CompatAdminAdjustBalance(w http.ResponseWriter, r *http.Request) {
	address, kind, amount, err := readAdjustParams(r, true)
	if err != nil {
		writeBizError(w, biz.ErrInvalidAmount)
		return
	}
	s.applyAdminAdjust(w, r, address, kind, amount)
}

func (s *AppService) writeAdminAdjust(w http.ResponseWriter, r *http.Request, kind string) {
	address, _, amount, err := readAdjustParams(r, false)
	if err != nil {
		writeBizError(w, biz.ErrInvalidAmount)
		return
	}
	s.applyAdminAdjust(w, r, address, kind, amount)
}

func (s *AppService) applyAdminAdjust(w http.ResponseWriter, r *http.Request, address, kind, amount string) {
	delta, err := parseAmount(amount)
	if err != nil {
		writeBizError(w, biz.ErrInvalidAmount)
		return
	}
	u, err := s.adjust.Adjust(r.Context(), address, kind, delta)
	if err != nil {
		writeBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "ok",
		"address":         u.Address,
		"kind":            kind,
		"delta":           decStr(delta),
		"available":       decStr(u.AvailableBalance),
		"lock":            decStr(u.LockBalance),
		"lockBalance":     decStr(u.LockBalance),
		"ispay":           decStr(u.IspayBalance),
		"lockIspay":       decStr(u.LockIspay),
		"recharge":        decStr(u.RechargeBalance),
		"rechargeBalance": decStr(u.RechargeBalance),
	})
}

func (s *AppService) CompatAdminAddMoneyThree(w http.ResponseWriter, r *http.Request) {
	s.writeAdminAdjust(w, r, biz.AdjustRecharge)
}

func (s *AppService) CompatAdminRecordList(w http.ResponseWriter, r *http.Request) {
	page, err := s.deposit.ListAdmin(r.Context(), strings.TrimSpace(r.URL.Query().Get("address")), parsePage(r))
	if err != nil {
		writeBizError(w, err)
		return
	}
	locations := make([]map[string]any, 0, len(page.Items))
	for _, d := range page.Items {
		locations = append(locations, depositJSON(d))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"locations": locations,
		"count":     strconv.Itoa(page.Total),
	})
}

func (s *AppService) CompatAdminPackageList(w http.ResponseWriter, r *http.Request) {
	pkgs, err := s.orders.ListAllPackages(r.Context())
	if err != nil {
		writeBizError(w, err)
		return
	}
	goods := make([]map[string]any, 0, len(pkgs))
	for _, p := range pkgs {
		goods = append(goods, packageJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"goods":  goods,
		"items":  goods,
		"count":  strconv.Itoa(len(goods)),
	})
}

func writePackageBiz(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, biz.ErrPackageNotFound):
		writeJSON(w, http.StatusOK, map[string]string{"status": "套餐不存在"})
	case errors.Is(err, biz.ErrPackageAmountTaken):
		writeJSON(w, http.StatusOK, map[string]string{"status": "金额已存在"})
	case errors.Is(err, biz.ErrPackageTitle):
		writeJSON(w, http.StatusOK, map[string]string{"status": "请填写名称"})
	case errors.Is(err, biz.ErrPackageInUse):
		writeJSON(w, http.StatusOK, map[string]string{"status": "已有订单不能删除，请先下架"})
	case errors.Is(err, biz.ErrInvalidAmount):
		writeJSON(w, http.StatusOK, map[string]string{"status": "金额错误"})
	case errors.Is(err, biz.ErrInvalidReleaseDays):
		writeJSON(w, http.StatusOK, map[string]string{"status": "请选择释放天数"})
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "fail"})
	}
}

func (s *AppService) CompatAdminPackageCreate(w http.ResponseWriter, r *http.Request) {
	in, err := readAdminPackage(r)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "金额错误"})
		return
	}
	p, err := s.orders.CreatePackage(r.Context(), &in.Package)
	if err != nil {
		writePackageBiz(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "item": packageJSON(p)})
}

func (s *AppService) CompatAdminPackageUpdate(w http.ResponseWriter, r *http.Request) {
	in, err := readAdminPackage(r)
	if err != nil || in.ID == 0 {
		writeBizError(w, biz.ErrInvalidAmount)
		return
	}
	cur, err := s.orders.GetPackage(r.Context(), in.ID)
	if err != nil {
		writePackageBiz(w, err)
		return
	}
	if strings.TrimSpace(in.Title) == "" {
		in.Title = cur.Title
	}
	if strings.TrimSpace(in.GoodsDesc) == "" {
		in.GoodsDesc = cur.GoodsDesc
	}
	if !in.Amount.IsPositive() {
		in.Amount = cur.Amount
	}
	if in.ReleaseDays == 0 {
		in.ReleaseDays = cur.ReleaseDays
	}
	if !in.hasDailyCap {
		in.DailyCap = cur.DailyCap
	}
	if !in.hasSort {
		in.SortOrder = cur.SortOrder
	}
	if !in.hasEnabled {
		in.Enabled = cur.Enabled
	}
	p, err := s.orders.SavePackage(r.Context(), &in.Package)
	if err != nil {
		writePackageBiz(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "item": packageJSON(p)})
}

func (s *AppService) CompatAdminPackageDelete(w http.ResponseWriter, r *http.Request) {
	in, err := readAdminPackage(r)
	if err != nil || in.ID == 0 {
		writeBizError(w, biz.ErrInvalidAmount)
		return
	}
	if err := s.orders.DeletePackage(r.Context(), in.ID); err != nil {
		writePackageBiz(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type adminPackageIn struct {
	biz.Package
	hasDailyCap bool
	hasSort     bool
	hasEnabled  bool
}

func readAdminPackage(r *http.Request) (*adminPackageIn, error) {
	in := &adminPackageIn{}
	in.Enabled = true
	if strings.Contains(r.Header.Get("Content-Type"), "json") {
		var body struct {
			ID      json.RawMessage `json:"id"`
			Amount  json.RawMessage `json:"amount"`
			Days    json.RawMessage `json:"days"`
			Rel     json.RawMessage `json:"release_days"`
			Title   string          `json:"title"`
			Name    string          `json:"name"`
			Goods   string          `json:"goods"`
			One     string          `json:"one"`
			Desc    string          `json:"goods_desc"`
			Cap     json.RawMessage `json:"daily_cap"`
			Cap2    json.RawMessage `json:"dailyCap"`
			Sort    json.RawMessage `json:"sort_order"`
			Sort2   json.RawMessage `json:"sortOrder"`
			Enabled json.RawMessage `json:"enabled"`
			Status  json.RawMessage `json:"status"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return nil, err
		}
		in.ID, _ = strconv.ParseUint(strings.Trim(string(body.ID), `"`), 10, 64)
		in.Title = firstNonEmpty(body.Title, body.Name)
		in.GoodsDesc = firstNonEmpty(body.Goods, body.One, body.Desc)
		if amt := strings.Trim(string(body.Amount), `"`); amt != "" && amt != "null" {
			d, err := parseAmount(amt)
			if err != nil {
				return nil, err
			}
			in.Amount = d
		}
		if days := firstNonEmpty(strings.Trim(string(body.Rel), `"`), strings.Trim(string(body.Days), `"`)); days != "" && days != "null" {
			n, err := parseReleaseDays(days)
			if err != nil {
				in.ReleaseDays = 0
			} else {
				in.ReleaseDays = n
			}
		}
		if cap := firstNonEmpty(strings.Trim(string(body.Cap), `"`), strings.Trim(string(body.Cap2), `"`)); cap != "" && cap != "null" {
			d, err := parseAmount(cap)
			if err != nil {
				return nil, err
			}
			in.DailyCap = d
			in.hasDailyCap = true
		}
		if sortRaw := firstNonEmpty(strings.Trim(string(body.Sort), `"`), strings.Trim(string(body.Sort2), `"`)); sortRaw != "" && sortRaw != "null" {
			n, _ := strconv.Atoi(sortRaw)
			in.SortOrder = n
			in.hasSort = true
		}
		if en := firstNonEmpty(strings.Trim(string(body.Enabled), `"`), strings.Trim(string(body.Status), `"`)); en != "" && en != "null" {
			in.Enabled = isTruthyFlag(en)
			in.hasEnabled = true
		}
		return in, nil
	}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	in.ID, _ = strconv.ParseUint(r.Form.Get("id"), 10, 64)
	in.Title = firstNonEmpty(r.Form.Get("title"), r.Form.Get("name"))
	in.GoodsDesc = firstNonEmpty(r.Form.Get("goods"), r.Form.Get("one"), r.Form.Get("goods_desc"))
	if amt := r.Form.Get("amount"); amt != "" {
		d, err := parseAmount(amt)
		if err != nil {
			return nil, err
		}
		in.Amount = d
	}
	if days := firstNonEmpty(r.Form.Get("release_days"), r.Form.Get("days")); days != "" {
		if n, err := parseReleaseDays(days); err == nil {
			in.ReleaseDays = n
		}
	}
	if cap := firstNonEmpty(r.Form.Get("daily_cap"), r.Form.Get("dailyCap")); cap != "" {
		d, err := parseAmount(cap)
		if err != nil {
			return nil, err
		}
		in.DailyCap = d
		in.hasDailyCap = true
	}
	if sortRaw := firstNonEmpty(r.Form.Get("sort_order"), r.Form.Get("sortOrder")); sortRaw != "" {
		n, _ := strconv.Atoi(sortRaw)
		in.SortOrder = n
		in.hasSort = true
	}
	if en := firstNonEmpty(r.Form.Get("enabled"), r.Form.Get("status")); en != "" {
		in.Enabled = isTruthyFlag(en)
		in.hasEnabled = true
	}
	return in, nil
}

func readPackageUpdate(r *http.Request) (id uint64, amount, days string, err error) {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		var body struct {
			ID     json.RawMessage `json:"id"`
			Amount json.RawMessage `json:"amount"`
			Days   json.RawMessage `json:"days"`
			Rel    json.RawMessage `json:"release_days"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return 0, "", "", err
		}
		id, _ = strconv.ParseUint(strings.Trim(string(body.ID), `"`), 10, 64)
		amount = strings.Trim(string(body.Amount), `"`)
		days = firstNonEmpty(strings.Trim(string(body.Rel), `"`), strings.Trim(string(body.Days), `"`))
		return id, amount, days, nil
	}
	if err := r.ParseForm(); err != nil {
		return 0, "", "", err
	}
	id, _ = strconv.ParseUint(r.Form.Get("id"), 10, 64)
	return id, r.Form.Get("amount"), firstNonEmpty(r.Form.Get("release_days"), r.Form.Get("days")), nil
}

func readAdjustParams(r *http.Request, needKind bool) (address, kind, amount string, err error) {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		var body struct {
			Address json.RawMessage `json:"address"`
			Kind    json.RawMessage `json:"kind"`
			Amount  json.RawMessage `json:"amount"`
			Usdt    json.RawMessage `json:"usdt"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return "", "", "", err
		}
		address = strings.Trim(string(body.Address), `"`)
		kind = strings.Trim(string(body.Kind), `"`)
		amount = firstNonEmpty(strings.Trim(string(body.Usdt), `"`), strings.Trim(string(body.Amount), `"`))
		if needKind && strings.TrimSpace(kind) == "" {
			return "", "", "", biz.ErrAdjustKind
		}
		return address, kind, amount, nil
	}
	if err := r.ParseForm(); err != nil {
		return "", "", "", err
	}
	address = firstNonEmpty(r.Form.Get("address"))
	kind = r.Form.Get("kind")
	amount = firstNonEmpty(r.Form.Get("usdt"), r.Form.Get("amount"))
	if needKind && strings.TrimSpace(kind) == "" {
		return "", "", "", biz.ErrAdjustKind
	}
	return address, kind, amount, nil
}

func readLockParams(r *http.Request) (userID uint64, lockStr string, line bool, err error) {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		var body struct {
			UserID json.RawMessage `json:"user_id"`
			Lock   json.RawMessage `json:"lock"`
			One    json.RawMessage `json:"one"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return 0, "", false, err
		}
		userID, err = parseUint(strings.Trim(string(body.UserID), `"`))
		if err != nil {
			return 0, "", false, err
		}
		lockStr = strings.Trim(string(body.Lock), `"`)
		one := strings.Trim(string(body.One), `"`)
		return userID, lockStr, isTruthyFlag(one), nil
	}
	if err := r.ParseForm(); err != nil {
		return 0, "", false, err
	}
	userID, err = parseUint(r.Form.Get("user_id"))
	if err != nil {
		return 0, "", false, err
	}
	return userID, r.Form.Get("lock"), isTruthyFlag(r.Form.Get("one")), nil
}

func readUserIDParam(r *http.Request) (uint64, error) {
	if q := strings.TrimSpace(r.URL.Query().Get("user_id")); q != "" {
		return parseUint(q)
	}
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		var body struct {
			UserID json.RawMessage `json:"user_id"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return 0, err
		}
		return parseUint(strings.Trim(string(body.UserID), `"`))
	}
	if err := r.ParseForm(); err != nil {
		return 0, err
	}
	return parseUint(r.Form.Get("user_id"))
}

func (s *AppService) CompatWithdraw(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeBizError(w, biz.ErrUnauthorized)
		return
	}
	var body struct {
		Amount   json.RawMessage `json:"amount"`
		CoinType json.RawMessage `json:"coinType"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "金额无效"})
		return
	}
	amount, err := parseAmount(strings.Trim(string(body.Amount), `"`))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "金额无效"})
		return
	}
	asset, err := biz.ParseWithdrawAsset(jsonScalar(body.CoinType))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": withdrawUserMessage(err)})
		return
	}
	wd, err := s.withdraw.CreateAsset(r.Context(), uid, amount, asset)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": withdrawUserMessage(err)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": wd.ID})
}

func (s *AppService) CompatWithdrawCancel(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeBizError(w, biz.ErrUnauthorized)
		return
	}
	id, err := readOrderID(r)
	if err != nil || id == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "金额无效"})
		return
	}
	wd, err := s.withdraw.Cancel(r.Context(), uid, id)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": withdrawUserMessage(err)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": wd.ID, "withdraw_status": wd.Status})
}

func jsonScalar(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	return strings.Trim(s, `"`)
}

func withdrawAssetJSON(asset string) string {
	if strings.TrimSpace(asset) == "" {
		return biz.WithdrawAssetUSDT
	}
	return asset
}

func withdrawUserMessage(err error) string {
	switch {
	case errors.Is(err, biz.ErrInsufficientBalance):
		return "可提余额不足"
	case errors.Is(err, biz.ErrWithdrawBelowMin):
		return "低于最低提现金额"
	case errors.Is(err, biz.ErrWithdrawFeeExceeds):
		return "手续费后到账必须大于0"
	case errors.Is(err, biz.ErrWithdrawDailyCap):
		return "超过每日提现上限"
	case errors.Is(err, biz.ErrWithdrawConflict):
		return "该提现不能取消"
	case errors.Is(err, biz.ErrForbidden):
		return "不能取消他人提现"
	case errors.Is(err, biz.ErrWithdrawNotFound):
		return "提现单不存在"
	case errors.Is(err, biz.ErrInvalidAmount):
		return "金额无效"
	case errors.Is(err, biz.ErrUserDisabled):
		return "用户已锁定"
	case errors.Is(err, biz.ErrUserInactive):
		return "未激活用户不能提现，请先购买订单激活"
	case errors.Is(err, biz.ErrIspayWithdrawClosed):
		return "暂不支持ISPAY提现"
	case errors.Is(err, biz.ErrInvalidWithdrawAsset):
		return "提现类型无效"
	default:
		return "提现失败"
	}
}

func (s *AppService) CompatWithdrawList(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeBizError(w, biz.ErrUnauthorized)
		return
	}
	page, err := s.withdraw.ListUser(r.Context(), uid, parsePage(r))
	if err != nil {
		writeBizError(w, err)
		return
	}
	list := make([]map[string]any, 0, len(page.Items))
	for _, it := range page.Items {
		list = append(list, map[string]any{
			"id":        it.ID,
			"amount":    decStr(it.Amount),
			"feeAmount": decStr(it.FeeAmount),
			"relAmount": decStr(it.CreditedAmount),
			"asset":     withdrawAssetJSON(it.Asset),
			"type":      withdrawAssetJSON(it.Asset),
			"status":    it.Status,
			"createdAt": it.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count": page.Total,
		"list":  list,
	})
}

func (s *AppService) CompatAdminWithdrawList(w http.ResponseWriter, r *http.Request) {
	rawAsset := strings.TrimSpace(r.URL.Query().Get("withDrawType"))
	if rawAsset == "" {
		rawAsset = strings.TrimSpace(r.URL.Query().Get("asset"))
	}
	asset, err := biz.ParseWithdrawListAsset(rawAsset)
	if err != nil {
		writeBizError(w, err)
		return
	}
	page, err := s.withdraw.ListAdmin(
		r.Context(),
		strings.TrimSpace(r.URL.Query().Get("address")),
		strings.TrimSpace(r.URL.Query().Get("status")),
		asset,
		parsePage(r),
	)
	if err != nil {
		writeBizError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(page.Items))
	for _, it := range page.Items {
		out = append(out, map[string]any{
			"id":          it.ID,
			"address":     it.Address,
			"amount":      decStr(it.Amount),
			"relAmount":   decStr(it.CreditedAmount),
			"feeAmount":   decStr(it.FeeAmount),
			"asset":       withdrawAssetJSON(it.Asset),
			"type":        withdrawAssetJSON(it.Asset),
			"status":      it.Status,
			"txHash":      it.TxHash,
			"payoutError": it.PayoutError,
			"createdAt":   it.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"withdraw": out,
		"count":    strconv.Itoa(page.Total),
	})
}

func (s *AppService) CompatAdminWithdrawPass(w http.ResponseWriter, r *http.Request) {
	id, err := readOrderID(r)
	if err != nil || id == 0 {
		writeBizError(w, biz.ErrInvalidAmount)
		return
	}
	wd, err := s.withdraw.Pass(r.Context(), id)
	if err != nil {
		writeBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": wd.ID, "withdraw_status": wd.Status})
}

func (s *AppService) CompatAdminWithdrawPayout(w http.ResponseWriter, r *http.Request) {
	id, err := readOptionalPayoutID(r)
	if err != nil {
		writeBizError(w, biz.ErrInvalidAmount)
		return
	}
	res, err := s.withdraw.RunPayout(r.Context(), id)
	if err != nil {
		writeBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"enabled": res.Enabled,
		"scanned": res.Scanned,
		"sent":    res.Sent,
		"passed":  res.Passed,
		"failed":  res.Failed,
		"skipped": res.Skipped,
	})
}

func readOptionalPayoutID(r *http.Request) (uint64, error) {
	ct := r.Header.Get("Content-Type")
	raw := ""
	if strings.Contains(ct, "json") {
		var body struct {
			ID json.RawMessage `json:"id"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return 0, err
		}
		raw = strings.Trim(string(body.ID), `"`)
	} else if err := r.ParseForm(); err != nil {
		return 0, err
	} else {
		raw = r.Form.Get("id")
	}
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return 0, nil
	}
	return parseUint(raw)
}

func (s *AppService) CompatAdminWithdrawReject(w http.ResponseWriter, r *http.Request) {
	id, err := readOrderID(r)
	if err != nil || id == 0 {
		writeBizError(w, biz.ErrInvalidAmount)
		return
	}
	wd, err := s.withdraw.Reject(r.Context(), id)
	if err != nil {
		writeBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": wd.ID, "withdraw_status": wd.Status})
}

func placementJSON(v *biz.PlacementView) map[string]any {
	return map[string]any{
		"id":              v.ID,
		"user_id":         v.UserID,
		"address":         v.UserAddress,
		"sponsor_id":      v.SponsorID,
		"sponsor_address": v.SponsorAddress,
		"side":            v.Side,
		"createdAt":       v.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func (s *AppService) CompatAdminPlacement(w http.ResponseWriter, r *http.Request) {
	userID, sponsorID, side, userAddr, sponsorAddr, err := readPlacementParams(r)
	if err != nil {
		writeBizError(w, biz.ErrInvalidAmount)
		return
	}
	uid, err := s.place.ResolveUserID(r.Context(), userID, userAddr)
	if err != nil {
		writeBizError(w, err)
		return
	}
	sid, err := s.place.ResolveUserID(r.Context(), sponsorID, sponsorAddr)
	if err != nil {
		writeBizError(w, err)
		return
	}
	p, err := s.place.Place(r.Context(), uid, sid, side)
	if err != nil {
		writeBizError(w, err)
		return
	}
	view, err := s.place.GetByUser(r.Context(), p.UserID)
	if err != nil {
		writeBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "placement": placementJSON(view)})
}

func (s *AppService) CompatAdminPlacementGet(w http.ResponseWriter, r *http.Request) {
	userID, _ := parseUint(strings.TrimSpace(r.URL.Query().Get("user_id")))
	sponsorID, _ := parseUint(strings.TrimSpace(r.URL.Query().Get("sponsor_id")))
	address := strings.TrimSpace(r.URL.Query().Get("address"))
	sponsorAddr := strings.TrimSpace(r.URL.Query().Get("sponsor_address"))
	if sponsorAddr == "" {
		sponsorAddr = strings.TrimSpace(r.URL.Query().Get("sponsor"))
	}

	if sponsorID > 0 || sponsorAddr != "" {
		sid, err := s.place.ResolveUserID(r.Context(), sponsorID, sponsorAddr)
		if err != nil {
			writeBizError(w, err)
			return
		}
		children, err := s.place.ListChildren(r.Context(), sid)
		if err != nil {
			writeBizError(w, err)
			return
		}
		list := make([]map[string]any, 0, len(children))
		for _, c := range children {
			list = append(list, placementJSON(c))
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "children": list})
		return
	}

	if userID > 0 || address != "" {
		uid, err := s.place.ResolveUserID(r.Context(), userID, address)
		if err != nil {
			writeBizError(w, err)
			return
		}
		view, err := s.place.GetByUser(r.Context(), uid)
		if err != nil {
			writeBizError(w, err)
			return
		}
		if view == nil {
			writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "placement": nil})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "placement": placementJSON(view)})
		return
	}

	page, total, err := s.place.ListAdmin(r.Context(), address, parsePage(r))
	if err != nil {
		writeBizError(w, err)
		return
	}
	list := make([]map[string]any, 0, len(page))
	for _, v := range page {
		list = append(list, placementJSON(v))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"list":   list,
		"count":  strconv.Itoa(total),
	})
}

func recommendNodeJSON(n *biz.RecommendNode) map[string]any {
	if n == nil {
		return nil
	}
	item := map[string]any{
		"user_id": n.UserID,
		"address": n.Address,
		"amount":  decStr(n.Amount),
		"side":    n.Side,
	}
	kids := make([]map[string]any, 0, 2)
	if n.Left != nil {
		kids = append(kids, recommendNodeJSON(n.Left))
	}
	if n.Right != nil {
		kids = append(kids, recommendNodeJSON(n.Right))
	}
	if len(kids) > 0 {
		item["children"] = kids
	}
	return item
}

func recommendJSON(nodes []*biz.RecommendNode) []map[string]any {
	list := make([]map[string]any, 0, len(nodes))
	for _, n := range nodes {
		if n == nil {
			continue
		}
		list = append(list, recommendNodeJSON(n))
	}
	return list
}

func (s *AppService) fullDownline() bool {
	return s.app != nil && s.app.FullDownline
}

func downlinePersonJSON(p *biz.DownlinePerson) map[string]any {
	if p == nil {
		return nil
	}
	created := ""
	if !p.CreatedAt.IsZero() {
		created = p.CreatedAt.Format("2006-01-02 15:04:05")
	}
	item := map[string]any{
		"user_id":   p.UserID,
		"address":   p.Address,
		"paid":      decStr(p.Paid),
		"createdAt": created,
	}
	if len(p.Children) > 0 {
		kids := make([]map[string]any, 0, len(p.Children))
		for _, c := range p.Children {
			kids = append(kids, downlinePersonJSON(c))
		}
		item["children"] = kids
	}
	return item
}

func downlineSlotJSON(n *biz.RecommendNode) map[string]any {
	if n == nil {
		return nil
	}
	item := map[string]any{
		"user_id": n.UserID,
		"address": n.Address,
		"amount":  decStr(n.Amount),
		"side":    n.Side,
	}
	if n.Left != nil {
		item["left"] = downlineSlotJSON(n.Left)
	}
	if n.Right != nil {
		item["right"] = downlineSlotJSON(n.Right)
	}
	return item
}

func (s *AppService) CompatAdminDownline(w http.ResponseWriter, r *http.Request) {
	userID, _ := parseUint(strings.TrimSpace(r.URL.Query().Get("user_id")))
	if userID == 0 {
		userID, _ = parseUint(strings.TrimSpace(r.URL.Query().Get("userId")))
	}
	address := strings.TrimSpace(r.URL.Query().Get("address"))
	var (
		view *biz.DownlineView
		err  error
	)
	if s.fullDownline() {
		view, err = s.place.AdminDownlineFull(r.Context(), userID, address)
	} else {
		view, err = s.place.AdminDownline(r.Context(), userID, address)
	}
	if err != nil {
		writeBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, downlineViewJSON(view))
}

func (s *AppService) CompatUserDownline(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeBizError(w, biz.ErrUnauthorized)
		return
	}
	if s.place == nil {
		writeJSON(w, http.StatusOK, downlineViewJSON(&biz.DownlineView{}))
		return
	}
	view, err := s.place.UserDownline(r.Context(), uid, strings.TrimSpace(r.URL.Query().Get("address")))
	if err != nil {
		writeBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, downlineViewJSON(view))
}

func downlineViewJSON(view *biz.DownlineView) map[string]any {
	if view == nil {
		view = &biz.DownlineView{}
	}
	invites := make([]map[string]any, 0, len(view.Invites))
	for _, it := range view.Invites {
		invites = append(invites, downlinePersonJSON(it))
	}
	return map[string]any{
		"status":       "ok",
		"full":         view.Full,
		"current":      downlinePersonJSON(view.Current),
		"inviter":      downlinePersonJSON(view.Inviter),
		"sponsor":      downlinePersonJSON(view.Sponsor),
		"sponsor_side": view.SponsorSide,
		"invites":      invites,
		"left":         downlineSlotJSON(view.Left),
		"right":        downlineSlotJSON(view.Right),
	}
}

func (s *AppService) CompatRecommendList(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeBizError(w, biz.ErrUnauthorized)
		return
	}
	nodes, err := s.place.ListRecommend(r.Context(), uid, strings.TrimSpace(r.URL.Query().Get("address")))
	if err != nil {
		writeBizError(w, err)
		return
	}
	if s.fullDownline() {
		if err := s.place.ExpandRecommendTrees(r.Context(), nodes); err != nil {
			writeBizError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "full": s.fullDownline(), "recommends": recommendJSON(nodes)})
}

func (s *AppService) CompatAdminMyAuthList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"super": "1", "auth": []any{}})
}

func (s *AppService) CompatAdminAll(w http.ResponseWriter, r *http.Request) {
	if s.stats == nil {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	st, err := s.stats.Dashboard(r.Context())
	if err != nil {
		writeBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"totalUserR":    st.TotalUserR,
		"totalUser":     st.TotalUser,
		"todayUserR":    st.TodayUserR,
		"todayUser":     st.TodayUser,
		"buyTotal":      decStr(st.BuyTotal),
		"todayBuy":      decStr(st.TodayBuy),
		"balanceUsdt":   decStr(st.BalanceUSDT),
		"todayOne":      decStr(st.TodayOne),
		"todayTwo":      decStr(st.TodayTwo),
		"todayThree":    decStr(st.TodayThree),
		"totalReward":   decStr(st.TotalReward),
		"todayWithdraw": decStr(st.TodayWithdraw),
		"totalWithdraw": decStr(st.TotalWithdraw),
		"totalIspay":    decStr(st.TotalIspay),
	})
}

func (s *AppService) CompatAdminConfig(w http.ResponseWriter, r *http.Request) {
	if s.configs == nil {
		writeJSON(w, http.StatusOK, map[string]any{"config": []any{}})
		return
	}
	rows, err := s.configs.List(r.Context())
	if err != nil {
		writeBizError(w, err)
		return
	}
	list := make([]map[string]any, 0, len(rows))
	for _, c := range rows {
		list = append(list, map[string]any{
			"id":    strconv.FormatUint(c.ID, 10),
			"key":   c.Key,
			"name":  c.Name,
			"value": c.Value,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"config": list})
}

func (s *AppService) CompatAdminConfigUpdate(w http.ResponseWriter, r *http.Request) {
	id, value, err := readConfigUpdate(r)
	if err != nil || id == 0 {
		writeBizError(w, biz.ErrConfigInvalid)
		return
	}
	if s.configs == nil {
		writeBizError(w, biz.ErrConfigNotFound)
		return
	}
	row, err := s.configs.Update(r.Context(), id, value)
	if err != nil {
		writeBizError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"id":     strconv.FormatUint(row.ID, 10),
		"key":    row.Key,
		"value":  row.Value,
	})
}

func readConfigUpdate(r *http.Request) (uint64, string, error) {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		var body struct {
			ID    json.RawMessage `json:"id"`
			Value string          `json:"value"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return 0, "", err
		}
		id, err := parseUint(strings.Trim(string(body.ID), `"`))
		return id, body.Value, err
	}
	if err := r.ParseForm(); err != nil {
		return 0, "", err
	}
	id, err := parseUint(firstNonEmpty(r.Form.Get("id"), r.Form.Get("config_id")))
	return id, r.Form.Get("value"), err
}

func (s *AppService) CompatAdminRecommendList(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.place.ListRecommendAdmin(r.Context(), strings.TrimSpace(r.URL.Query().Get("address")))
	if err != nil {
		writeBizError(w, err)
		return
	}
	if s.fullDownline() {
		if err := s.place.ExpandRecommendTrees(r.Context(), nodes); err != nil {
			writeBizError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "full": s.fullDownline(), "recommends": recommendJSON(nodes)})
}

func readPlacementParams(r *http.Request) (userID, sponsorID uint64, side, userAddr, sponsorAddr string, err error) {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		var body struct {
			UserID         json.RawMessage `json:"user_id"`
			SponsorID      json.RawMessage `json:"sponsor_id"`
			Side           string          `json:"side"`
			Address        string          `json:"address"`
			SponsorAddress string          `json:"sponsor_address"`
			Sponsor        string          `json:"sponsor"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return 0, 0, "", "", "", err
		}
		userID, _ = parseUint(strings.Trim(string(body.UserID), `"`))
		sponsorID, _ = parseUint(strings.Trim(string(body.SponsorID), `"`))
		side = body.Side
		userAddr = body.Address
		sponsorAddr = firstNonEmpty(body.SponsorAddress, body.Sponsor)
		return userID, sponsorID, side, userAddr, sponsorAddr, nil
	}
	if err := r.ParseForm(); err != nil {
		return 0, 0, "", "", "", err
	}
	userID, _ = parseUint(r.Form.Get("user_id"))
	sponsorID, _ = parseUint(r.Form.Get("sponsor_id"))
	return userID, sponsorID, r.Form.Get("side"), r.Form.Get("address"), firstNonEmpty(r.Form.Get("sponsor_address"), r.Form.Get("sponsor")), nil
}

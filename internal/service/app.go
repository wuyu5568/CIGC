package service

import (
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
	ping     *data.Data
	auth     *conf.Auth
}

// NewAppService 构造 HTTP 适配器。
func NewAppService(users *biz.UserUseCase, orders *biz.OrderUseCase, settle *biz.SettleUseCase, ledger *biz.LedgerUseCase, withdraw *biz.WithdrawUseCase, place *biz.PlacementUseCase, ping *data.Data, auth *conf.Auth) *AppService {
	return &AppService{users: users, orders: orders, settle: settle, ledger: ledger, withdraw: withdraw, place: place, ping: ping, auth: auth}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
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
	return d.Round(8).StringFixed(8)
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
		goods = append(goods, map[string]any{
			"id":       p.ID,
			"amount":   decStr(p.Amount),
			"title":    p.Title,
			"goods":    p.GoodsDesc,
			"dailyCap": decStr(p.DailyCap),
		})
	}
	avail := decStr(user.AvailableBalance)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":            "ok",
		"address":           user.Address,
		"level":             "0",
		"usdt":              avail,
		"raw":               avail,
		"amountGet":         avail,
		"amountUsdt":        avail,
		"inviteUserAddress": inviteAddr,
		"withdrawRate":      "0",
		"withdrawMin":       "0",
		"withdrawRateTwo":   "0",
		"withdrawMinTwo":    "0",
		"locationNum":       "0",
		"LocationList":      []any{},
		"total":             "0.00000000",
		"max":               "0.00000000",
		"min":               "0.00000000",
		"buy":               decStr(user.PaidAmount),
		"amountGetSub":      "0.00000000",
		"outNum":            "0",
		"location":          "0.00000000",
		"recommend":         "0.00000000",
		"recommendTwo":      "0.00000000",
		"team":              "0.00000000",
		"teamTwo":           "0.00000000",
		"all":               avail,
		"notice":            "",
		"goods":             goods,
		"capEffective":      decStr(user.CapEffective),
		"frozen":            decStr(user.FrozenBalance),
	})
}

func (s *AppService) CompatPackageList(w http.ResponseWriter, r *http.Request) {
	pkgs, err := s.orders.ListPackages(r.Context())
	if err != nil {
		writeBizError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(pkgs))
	for _, p := range pkgs {
		items = append(items, map[string]any{
			"id":        p.ID,
			"amount":    decStr(p.Amount),
			"title":     p.Title,
			"goods":     p.GoodsDesc,
			"daily_cap": decStr(p.DailyCap),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "items": items})
}

func (s *AppService) CompatBuy(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeBizError(w, biz.ErrUnauthorized)
		return
	}
	var body struct {
		Amount json.RawMessage `json:"amount"`
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
	o, err := s.orders.CreateOrder(r.Context(), uid, amount)
	if err != nil {
		switch {
		case errors.Is(err, biz.ErrPackageNotFound), errors.Is(err, biz.ErrPackageDisabled):
			writeJSON(w, http.StatusOK, map[string]string{"status": "套餐不存在"})
		case errors.Is(err, biz.ErrInvalidAmount):
			writeJSON(w, http.StatusOK, map[string]string{"status": "金额错误"})
		default:
			writeJSON(w, http.StatusOK, map[string]string{"status": "fail"})
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"order_id": o.ID,
		"amount":   decStr(o.Amount),
	})
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
	items := make([]map[string]any, 0, len(rows))
	for _, o := range rows {
		item := map[string]any{
			"id":     o.ID,
			"amount": decStr(o.Amount),
			"title":  o.TitleSnapshot,
			"goods":  o.GoodsSnapshot,
			"status": o.Status,
		}
		if o.PaidAt != nil {
			item["paid_at"] = o.PaidAt.Format("2006-01-02 15:04:05")
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
		return "force=1 may rewrite caps again; test only"
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
			"id":        it.ID,
			"amount":    it.Amount,
			"amountTwo": it.Amount,
			"reward":    it.Amount,
			"name":      it.Name,
			"address":   it.Address,
			"num":       it.Num,
			"reason":    it.Reason,
			"createdAt": it.CreatedAt.Format("2006-01-02 15:04:05"),
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
			"id":        it.ID,
			"amount":    it.Amount,
			"amountTwo": it.Amount,
			"reward":    it.Amount,
			"name":      it.Name,
			"address":   it.Address,
			"num":       it.Num,
			"reason":    it.Reason,
			"remark":    it.Remark,
			"createdAt": it.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"rewards": rewards,
		"count":   strconv.Itoa(page.Total),
	})
}

func zeroStr() string { return "0.00000000" }

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
			"id":        o.ID,
			"amount":    decStr(o.Amount),
			"address":   o.Address,
			"createdAt": o.CreatedAt.Format("2006-01-02 15:04:05"),
			"status":    o.Status,
			"one":       o.TitleSnapshot,
			"two":       "",
			"three":     "",
			"four":      "",
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
		out = append(out, map[string]any{
			"userId":             strconv.FormatUint(u.ID, 10),
			"createdAt":          u.CreatedAt.Format("2006-01-02 15:04:05"),
			"address":            u.Address,
			"amountUsdtCurrent":  paid,
			"bAmount":            zeroStr(),
			"amountUsdtGet":      avail,
			"amountUsdtTwo":      zeroStr(),
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
			"myRecommendAddress": u.InviterAddress,
			"capEffective":       decStr(u.CapEffective),
			"frozen":             decStr(u.FrozenBalance),
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
		Amount json.RawMessage `json:"amount"`
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
	wd, err := s.withdraw.Create(r.Context(), uid, amount)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": withdrawUserMessage(err)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": wd.ID})
}

func withdrawUserMessage(err error) string {
	switch {
	case errors.Is(err, biz.ErrInsufficientBalance):
		return "可提余额不足"
	case errors.Is(err, biz.ErrWithdrawBelowMin):
		return "低于最低提现金额"
	case errors.Is(err, biz.ErrInvalidAmount):
		return "金额无效"
	case errors.Is(err, biz.ErrUserDisabled):
		return "用户已锁定"
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
	page, err := s.withdraw.ListAdmin(
		r.Context(),
		strings.TrimSpace(r.URL.Query().Get("address")),
		strings.TrimSpace(r.URL.Query().Get("status")),
		parsePage(r),
	)
	if err != nil {
		writeBizError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(page.Items))
	for _, it := range page.Items {
		out = append(out, map[string]any{
			"id":        it.ID,
			"address":   it.Address,
			"amount":    decStr(it.Amount),
			"relAmount": decStr(it.CreditedAmount),
			"feeAmount": decStr(it.FeeAmount),
			"status":    it.Status,
			"createdAt": it.CreatedAt.Format("2006-01-02 15:04:05"),
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

package biz

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

const (
	LedgerDirect        = "direct"
	LedgerMatch         = "match"
	LedgerManage        = "manage"
	LedgerAdminAdjust   = "admin_adjust"
	LedgerRecharge      = "recharge"
	LedgerRechargeBuy   = "recharge_buy"
	LedgerWithdraw      = "withdraw"
	LedgerFreeze        = "withdraw_freeze"
	LedgerUnfreeze      = "withdraw_unfreeze"
	LedgerActivate      = "activate_unfreeze"
	LedgerActivateIspay = "activate_unfreeze_ispay"

	BalanceAvailable   = "available"
	BalanceRecharge    = "recharge"
	BalanceFrozen      = "frozen"
	BalanceFrozenIspay = "frozen_ispay"
	BalanceLock        = "lock"
	BalanceLockIspay   = "lock_ispay"

	// DefaultRewardPageSize 对齐 dapp 分页（每页 10）。
	DefaultRewardPageSize = 10
)

// NormalizeAmount 将业务金额对齐到金牛口径：Round(8)。
func NormalizeAmount(d decimal.Decimal) decimal.Decimal {
	return money.Round(d)
}

// LedgerEntry 是一条余额变动流水，禁止只改余额不写流水。
type LedgerEntry struct {
	ID          uint64
	UserID      uint64
	OrderID     *uint64
	EntryType   string
	Amount      decimal.Decimal
	BalanceKind string
	SettleDate  *time.Time
	Remark      string
	Address     string
	CreatedAt   time.Time
}

// LedgerRepo 持久化账本。
type LedgerRepo interface {
	Create(ctx context.Context, e *LedgerEntry) error
	ListByUser(ctx context.Context, userID uint64, from, to time.Time) ([]*LedgerEntry, error)
	ListPaged(ctx context.Context, address string, entryTypes []string, page, pageSize int) ([]*LedgerEntry, int, error)
	ListByTypes(ctx context.Context, address string, entryTypes []string) ([]*LedgerEntry, error)
	FindMatching(ctx context.Context, userID uint64, entryType string, orderID *uint64, settleDate *time.Time, remark string) (*LedgerEntry, error)
	ExistsByOrderAndType(ctx context.Context, orderID uint64, entryType string) (bool, error)
	ExistsByOrderTypeAndDate(ctx context.Context, orderID uint64, entryType string, settleDate time.Time) (bool, error)
	CountByOrderAndType(ctx context.Context, orderID uint64, entryType string) (int, error)
	ListByOrderIDsAndTypes(ctx context.Context, orderIDs []uint64, entryTypes []string) ([]*LedgerEntry, error)
	ExistsByUserTypeAndDate(ctx context.Context, userID uint64, entryType string, settleDate time.Time) (bool, error)
	ExistsByUserTypeDateRemark(ctx context.Context, userID uint64, entryType string, settleDate time.Time, remark string) (bool, error)
	ListByTypeAndDate(ctx context.Context, entryType string, settleDate time.Time) ([]*LedgerEntry, error)
}

// RewardItem 是用户/管理端流水展示行。
type RewardItem struct {
	ID            uint64
	Amount        string
	AmountTwo     string
	Name          string
	Reason        string
	Category      string
	Address       string
	Num           string
	Remark        string
	Detail        string
	BalanceKind   string
	BalanceName   string
	OrderID       string
	OrderNo       string
	OrderAmount   string
	OrderTitle    string
	OrderSource   string
	SourceAddress string
	SettleDate    string
	CreatedAt     time.Time
}

// RewardPage 分页结果。
type RewardPage struct {
	Items []*RewardItem
	Total int
}

// LedgerUseCase 账本查询（写入由奖励/提现模块负责）。
type LedgerUseCase struct {
	ledger LedgerRepo
	orders OrderRepo
	users  UserRepo
	now    func() time.Time
}

// NewLedgerUseCase 构造账本查询用例。orders / users 可为 nil。
func NewLedgerUseCase(ledger LedgerRepo, orders OrderRepo, users UserRepo) *LedgerUseCase {
	return &LedgerUseCase{ledger: ledger, orders: orders, users: users, now: time.Now}
}

// IsRewardEntry 三类收益流水；默认列表只含这些类型。
func IsRewardEntry(entryType string) bool {
	switch entryType {
	case LedgerDirect, LedgerMatch, LedgerManage, LedgerStatic:
		return true
	default:
		return false
	}
}

// ReqTypeToEntryType 将 dapp reqType 映射为 entry_type。
// 空或 "1"：默认三类收益（返回 ok=true 且 entryType="" 表示全部收益类）。
// "7"：冻结释放（activate_unfreeze*，由 filter 去重配对）。
// "6" 是冻结资产快照，不走账本（由 CompatRewardList 拦截）。
// 未知码：ok=false，调用方应返回空列表。
func ReqTypeToEntryType(reqType string) (entryType string, ok bool) {
	switch reqType {
	case "", "1":
		return "", true
	case "2":
		return LedgerStatic, true
	case "3":
		return LedgerDirect, true
	case "4":
		return LedgerMatch, true
	case "5":
		return LedgerManage, true
	case "7":
		return LedgerActivate, true
	default:
		return "", false
	}
}

// ReasonToEntryType 管理端 reason 筛选 → 单个 entry_type；空表示不按类型筛。
func ReasonToEntryType(reason string) string {
	types := ReasonToEntryTypes(reason)
	if len(types) == 1 {
		return types[0]
	}
	return ""
}

// ReasonToEntryTypes 管理端筛选。空/all 不限类型；reward 为静态+动态 U 入账。
func ReasonToEntryTypes(reason string) []string {
	switch strings.TrimSpace(reason) {
	case "", "all":
		return nil
	case "reward", "income":
		return []string{LedgerStatic, LedgerDirect, LedgerMatch, LedgerManage}
	case "dynamic":
		return []string{LedgerDirect, LedgerMatch, LedgerManage}
	case "static", "2", "location":
		return []string{LedgerStatic}
	case "direct", "recommend", "3":
		return []string{LedgerDirect}
	case "match", "4", "area":
		return []string{LedgerMatch}
	case "manage", "community_base", "5", "area_two":
		return []string{LedgerManage}
	case "unfreeze", "activate", "activate_unfreeze", "7":
		return []string{LedgerActivate, LedgerActivateIspay}
	case "admin_adjust":
		return []string{LedgerAdminAdjust}
	case "withdraw", "extract":
		return []string{LedgerWithdraw}
	case "withdraw_freeze":
		return []string{LedgerFreeze}
	case "withdraw_unfreeze":
		return []string{LedgerUnfreeze}
	default:
		return []string{reason}
	}
}

// RewardFamily 把 U / ISPAY 成对类型归到同一收益族。
func RewardFamily(entryType string) string {
	switch entryType {
	case LedgerStatic, LedgerStaticIspay:
		return LedgerStatic
	case LedgerDirect, LedgerDirectIspay:
		return LedgerDirect
	case LedgerMatch, LedgerMatchIspay:
		return LedgerMatch
	case LedgerManage, LedgerManageIspay:
		return LedgerManage
	default:
		return entryType
	}
}

// RewardCategoryName 静态 / 动态。
func RewardCategoryName(entryType string) string {
	switch RewardFamily(entryType) {
	case LedgerStatic:
		return "静态收益"
	case LedgerDirect, LedgerMatch, LedgerManage:
		return "动态收益"
	default:
		return ""
	}
}

// IspayEntryType 对应的 ISPAY 半边流水类型。
func IspayEntryType(entryType string) string {
	switch entryType {
	case LedgerStatic:
		return LedgerStaticIspay
	case LedgerDirect:
		return LedgerDirectIspay
	case LedgerMatch:
		return LedgerMatchIspay
	case LedgerManage:
		return LedgerManageIspay
	default:
		return ""
	}
}

// BalanceKindName 入账账户展示。
func BalanceKindName(kind string) string {
	switch kind {
	case BalanceAvailable:
		return "可提U"
	case BalanceLock:
		return "冻结U"
	case BalanceIspay:
		return "可提ISPAY"
	case BalanceLockIspay:
		return "冻结ISPAY"
	case BalanceRecharge:
		return "充值余额"
	case BalanceFrozen:
		return "提现冻结U"
	case BalanceFrozenIspay:
		return "提现冻结ISPAY"
	default:
		return kind
	}
}

// LedgerName 展示名。
func LedgerName(entryType string) string {
	switch entryType {
	case LedgerStatic, LedgerStaticIspay:
		return "静态释放"
	case LedgerDirect, LedgerDirectIspay:
		return "直推奖励"
	case LedgerMatch, LedgerMatchIspay:
		return "对碰奖励"
	case LedgerManage, LedgerManageIspay:
		return "管理奖"
	case LedgerAdminAdjust:
		return "管理端调整"
	case LedgerWithdraw:
		return "提现"
	case LedgerFreeze:
		return "提现冻结"
	case LedgerUnfreeze:
		return "提现解冻"
	case LedgerActivate, LedgerActivateIspay:
		return "解冻"
	case LedgerCapOverflowRelease:
		return "封顶超额释放"
	case LedgerCapOverflowBurn:
		return "封顶超额清除"
	default:
		return entryType
	}
}

// LedgerReason 对外 reason 字段。
func LedgerReason(entryType string) string {
	switch entryType {
	case LedgerStatic:
		return "static"
	case LedgerDirect:
		return "direct"
	case LedgerMatch:
		return "match"
	case LedgerManage:
		return "manage"
	case LedgerAdminAdjust:
		return "admin_adjust"
	case LedgerWithdraw:
		return "withdraw"
	case LedgerFreeze:
		return "withdraw_freeze"
	case LedgerUnfreeze:
		return "withdraw_unfreeze"
	case LedgerActivate:
		return "unfreeze"
	case LedgerActivateIspay:
		return "unfreeze_ispay"
	default:
		return entryType
	}
}

func amountStr(d decimal.Decimal) string {
	return money.Display(d)
}

func toRewardItem(e *LedgerEntry) *RewardItem {
	usdt := e.Amount
	ispay := decimal.Zero
	return &RewardItem{
		ID:          e.ID,
		Amount:      amountStr(usdt),
		AmountTwo:   amountStr(ispay),
		Name:        LedgerName(e.EntryType),
		Reason:      LedgerReason(e.EntryType),
		Category:    RewardCategoryName(e.EntryType),
		Address:     e.Address,
		Num:         "",
		Remark:      e.Remark,
		Detail:      RewardDetail(e),
		BalanceKind: e.BalanceKind,
		BalanceName: BalanceKindName(e.BalanceKind),
		OrderID:     orderIDStr(e.OrderID),
		SettleDate:  settleDateStr(e.SettleDate),
		CreatedAt:   e.CreatedAt,
	}
}

func toAdminRewardItem(e *LedgerEntry, ispay *LedgerEntry) *RewardItem {
	item := toRewardItem(e)
	item.AmountTwo = "0"
	if ispay != nil {
		item.AmountTwo = amountStr(ispay.Amount)
	}
	kv := parseRemarkKV(e.Remark)
	if g := kv["gen"]; g != "" {
		item.Num = g
	}
	return item
}

func orderIDStr(id *uint64) string {
	if id == nil || *id == 0 {
		return ""
	}
	return strconv.FormatUint(*id, 10)
}

func settleDateStr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func parseRemarkKV(remark string) map[string]string {
	out := map[string]string{}
	for _, p := range strings.Fields(remark) {
		k, v, ok := strings.Cut(p, "=")
		if ok {
			out[k] = v
		}
	}
	return out
}

// RewardDetail 把日结 remark 拆成可读说明。
func RewardDetail(e *LedgerEntry) string {
	if e == nil {
		return ""
	}
	kv := parseRemarkKV(e.Remark)
	switch RewardFamily(e.EntryType) {
	case LedgerStatic:
		if kv["day"] != "" {
			return "第" + kv["day"] + "天"
		}
		if kv["days"] != "" {
			return kv["days"] + "天档"
		}
		return ""
	case LedgerDirect:
		parts := []string{"直推到账"}
		if kv["rate"] != "" {
			parts = append(parts, "费率 "+formatRatePct(kv["rate"]))
		}
		return strings.Join(parts, " · ")
	case LedgerMatch:
		parts := []string{}
		if kv["pair"] != "" {
			parts = append(parts, "对碰额 "+kv["pair"])
		}
		if kv["rate"] != "" {
			parts = append(parts, "费率 "+formatRatePct(kv["rate"]))
		}
		return strings.Join(parts, " · ")
	case LedgerManage:
		if kv["gen"] != "" {
			return "第" + kv["gen"] + "代"
		}
		return ""
	case LedgerActivate:
		return freezeReleaseDetail(e.Remark)
	default:
		return e.Remark
	}
}

func formatRatePct(raw string) string {
	d, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil {
		return raw
	}
	if d.GreaterThan(decimal.NewFromInt(1)) {
		return money.Display(d) + "%"
	}
	return money.Display(d.Mul(decimal.NewFromInt(100))) + "%"
}

func freezeReleaseDetail(remark string) string {
	switch strings.TrimSpace(remark) {
	case remarkActivateUSDT, remarkActivateIspay:
		return "激活解冻至可提现"
	case remarkDailyLockUSDT, remarkDailyLockIspay:
		return "日结按封顶解冻"
	case remarkOverflowOrderUnlock:
		return "买单解冻超额冻结"
	case remarkOverflowClear72h:
		return "到期清除（不转入可提现）"
	default:
		if strings.Contains(remark, "overflow clear") {
			return "到期清除（不转入可提现）"
		}
		return remark
	}
}

func freezeReleaseName(remark string) string {
	switch strings.TrimSpace(remark) {
	case remarkOverflowClear72h:
		return "冻结清除"
	case remarkOverflowOrderUnlock:
		return "超额解冻"
	case remarkDailyLockUSDT, remarkDailyLockIspay:
		return "日结解冻"
	case remarkActivateUSDT, remarkActivateIspay:
		return "激活解冻"
	default:
		if strings.Contains(remark, "overflow clear") {
			return "冻结清除"
		}
		return "冻结释放"
	}
}

type freezePairKey struct {
	userID uint64
	remark string
	settle string
	order  string
}

func freezePairOf(e *LedgerEntry) freezePairKey {
	if e == nil {
		return freezePairKey{}
	}
	return freezePairKey{
		userID: e.UserID,
		remark: freezeRemarkFamily(e.Remark),
		settle: settleDateStr(e.SettleDate),
		order:  orderIDStr(e.OrderID),
	}
}

func freezeRemarkFamily(remark string) string {
	switch strings.TrimSpace(remark) {
	case remarkActivateUSDT, remarkActivateIspay:
		return "activate"
	case remarkDailyLockUSDT, remarkDailyLockIspay:
		return "daily"
	case remarkOverflowOrderUnlock:
		return "order_unlock"
	case remarkOverflowClear72h:
		return "clear"
	default:
		if strings.Contains(remark, "overflow clear") {
			return "clear"
		}
		return strings.TrimSpace(remark)
	}
}

func isOverflowClearRemark(remark string) bool {
	r := strings.TrimSpace(remark)
	return r == remarkOverflowClear72h || strings.Contains(r, "overflow clear")
}

// filterFreezeRelease 冻结释放明细：只展示解冻入账与到期清除，U/ISPAY 配对，去掉 lock 侧对账行。
func filterFreezeRelease(items []*LedgerEntry) []*RewardItem {
	ispayCredit := map[freezePairKey]decimal.Decimal{}
	ispayBurn := map[freezePairKey]decimal.Decimal{}
	for _, e := range items {
		if e == nil || e.EntryType != LedgerActivateIspay {
			continue
		}
		k := freezePairOf(e)
		if e.BalanceKind == BalanceIspay && e.Amount.IsPositive() {
			ispayCredit[k] = money.Round(e.Amount)
		}
		if e.BalanceKind == BalanceLockIspay && e.Amount.IsNegative() && isOverflowClearRemark(e.Remark) {
			ispayBurn[k] = money.Round(e.Amount.Abs())
		}
	}
	pairedCredit := map[freezePairKey]bool{}
	pairedBurn := map[freezePairKey]bool{}
	out := make([]*RewardItem, 0)
	for _, e := range items {
		if e == nil || e.EntryType != LedgerActivate {
			continue
		}
		k := freezePairOf(e)
		if e.BalanceKind == BalanceAvailable && e.Amount.IsPositive() {
			item := toRewardItem(e)
			item.Name = freezeReleaseName(e.Remark)
			item.Detail = freezeReleaseDetail(e.Remark)
			item.Reason = "unfreeze"
			if v, ok := ispayCredit[k]; ok {
				item.AmountTwo = amountStr(v)
				pairedCredit[k] = true
			} else {
				item.AmountTwo = "0"
			}
			out = append(out, item)
			continue
		}
		if e.BalanceKind == BalanceLock && e.Amount.IsNegative() && isOverflowClearRemark(e.Remark) {
			item := toRewardItem(e)
			item.Amount = amountStr(e.Amount.Abs())
			item.Name = freezeReleaseName(e.Remark)
			item.Detail = freezeReleaseDetail(e.Remark)
			item.Reason = "unfreeze_clear"
			if v, ok := ispayBurn[k]; ok {
				item.AmountTwo = amountStr(v)
				pairedBurn[k] = true
			} else {
				item.AmountTwo = "0"
			}
			out = append(out, item)
		}
	}
	for _, e := range items {
		if e == nil || e.EntryType != LedgerActivateIspay {
			continue
		}
		k := freezePairOf(e)
		if e.BalanceKind == BalanceIspay && e.Amount.IsPositive() && !pairedCredit[k] {
			item := toRewardItem(e)
			item.Amount = "0"
			item.AmountTwo = amountStr(e.Amount)
			item.Name = freezeReleaseName(e.Remark)
			item.Detail = freezeReleaseDetail(e.Remark)
			item.Reason = "unfreeze_ispay"
			out = append(out, item)
			continue
		}
		if e.BalanceKind == BalanceLockIspay && e.Amount.IsNegative() && isOverflowClearRemark(e.Remark) && !pairedBurn[k] {
			item := toRewardItem(e)
			item.Amount = "0"
			item.AmountTwo = amountStr(e.Amount.Abs())
			item.Name = freezeReleaseName(e.Remark)
			item.Detail = freezeReleaseDetail(e.Remark)
			item.Reason = "unfreeze_clear"
			out = append(out, item)
		}
	}
	return out
}

func paginateRewards(items []*RewardItem, page, pageSize int) []*RewardItem {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultRewardPageSize
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []*RewardItem{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func rewardPairOf(e *LedgerEntry) freezePairKey {
	if e == nil {
		return freezePairKey{}
	}
	return freezePairKey{
		userID: e.UserID,
		remark: RewardFamily(e.EntryType),
		settle: settleDateStr(e.SettleDate),
		order:  orderIDStr(e.OrderID),
	}
}

func filterUserRewardPair(items []*LedgerEntry, usdtType string) []*RewardItem {
	ispayType := IspayEntryType(usdtType)
	credits := map[freezePairKey]decimal.Decimal{}
	for _, e := range items {
		if e == nil || e.EntryType != ispayType || !e.Amount.IsPositive() {
			continue
		}
		k := rewardPairOf(e)
		credits[k] = money.Round(credits[k].Add(e.Amount))
	}
	out := make([]*RewardItem, 0)
	for _, e := range items {
		if e == nil || e.EntryType != usdtType || !e.Amount.IsPositive() {
			continue
		}
		item := toRewardItem(e)
		if v, ok := credits[rewardPairOf(e)]; ok {
			item.AmountTwo = amountStr(v)
		}
		out = append(out, item)
	}
	return out
}

func filterUserRewards(items []*LedgerEntry, want string) []*RewardItem {
	if want == LedgerActivate {
		return filterFreezeRelease(items)
	}
	switch want {
	case LedgerStatic, LedgerDirect, LedgerMatch, LedgerManage:
		return filterUserRewardPair(items, want)
	}
	out := make([]*RewardItem, 0, len(items))
	for _, e := range items {
		if want != "" {
			if e.EntryType != want {
				continue
			}
		} else if !IsRewardEntry(e.EntryType) {
			continue
		}
		out = append(out, toRewardItem(e))
	}
	return out
}

// ListUserRewards 用户端 reward_list：默认三类收益；reqType 映射单类；未知码空列表。
func (uc *LedgerUseCase) ListUserRewards(ctx context.Context, userID uint64, reqType string, page int) (*RewardPage, error) {
	want, ok := ReqTypeToEntryType(reqType)
	if !ok {
		return &RewardPage{Items: []*RewardItem{}, Total: 0}, nil
	}
	now := uc.now()
	from := now.AddDate(-20, 0, 0)
	to := now.AddDate(0, 0, 1)
	rows, err := uc.ledger.ListByUser(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	filtered := filterUserRewards(rows, want)
	if err := uc.attachOrderSources(ctx, filtered); err != nil {
		return nil, err
	}
	return &RewardPage{
		Items: paginateRewards(filtered, page, DefaultRewardPageSize),
		Total: len(filtered),
	}, nil
}

// UserRewardTotals 四类账本半边 U 合计（含冻结）+ 静态释放次数。用户端展示再 ×2 还原产值。
func (uc *LedgerUseCase) UserRewardTotals(ctx context.Context, userID uint64) (static, direct, match, manage decimal.Decimal, staticTimes int, err error) {
	if uc == nil || uc.ledger == nil || userID == 0 {
		return
	}
	now := uc.now()
	rows, err := uc.ledger.ListByUser(ctx, userID, now.AddDate(-20, 0, 0), now.AddDate(0, 0, 1))
	if err != nil {
		return
	}
	for _, e := range rows {
		if e == nil {
			continue
		}
		amt := money.Round(e.Amount)
		switch e.EntryType {
		case LedgerStatic:
			static = money.Round(static.Add(amt))
			staticTimes++
		case LedgerDirect:
			direct = money.Round(direct.Add(amt))
		case LedgerMatch:
			match = money.Round(match.Add(amt))
		case LedgerManage:
			manage = money.Round(manage.Add(amt))
		}
	}
	return
}

// ListAdminRewards 管理端流水：分页 + address + reason；reason 空则全类型。
func (uc *LedgerUseCase) ListAdminRewards(ctx context.Context, address, reason string, page int) (*RewardPage, error) {
	rows, total, err := uc.ledger.ListPaged(ctx, address, ReasonToEntryTypes(reason), page, DefaultRewardPageSize)
	if err != nil {
		return nil, err
	}
	items := make([]*RewardItem, 0, len(rows))
	for _, e := range rows {
		var ispay *LedgerEntry
		if pair := IspayEntryType(e.EntryType); pair != "" && uc.ledger != nil {
			ispay, err = uc.ledger.FindMatching(ctx, e.UserID, pair, e.OrderID, e.SettleDate, e.Remark)
			if err != nil {
				return nil, err
			}
		}
		items = append(items, toAdminRewardItem(e, ispay))
	}
	if err := uc.attachOrderSources(ctx, items); err != nil {
		return nil, err
	}
	return &RewardPage{Items: items, Total: total}, nil
}

// ListAdminFreezeRelease 管理端冻结释放明细：解冻入账与到期清除，U/ISPAY 配对。
func (uc *LedgerUseCase) ListAdminFreezeRelease(ctx context.Context, address string, page int) (*RewardPage, error) {
	if uc == nil || uc.ledger == nil {
		return &RewardPage{Items: []*RewardItem{}}, nil
	}
	rows, err := uc.ledger.ListByTypes(ctx, strings.TrimSpace(address), []string{LedgerActivate, LedgerActivateIspay})
	if err != nil {
		return nil, err
	}
	items := filterFreezeRelease(rows)
	for _, it := range items {
		if it == nil {
			continue
		}
		it.Category = "冻结释放"
	}
	if err := uc.attachOrderSources(ctx, items); err != nil {
		return nil, err
	}
	return &RewardPage{
		Items: paginateRewards(items, page, DefaultRewardPageSize),
		Total: len(items),
	}, nil
}

func (uc *LedgerUseCase) attachOrderSources(ctx context.Context, items []*RewardItem) error {
	if uc == nil {
		return nil
	}
	orderCache := map[uint64]*Order{}
	addrCache := map[uint64]string{}
	lookupOrder := func(id uint64) (*Order, error) {
		if uc.orders == nil || id == 0 {
			return nil, nil
		}
		if o, ok := orderCache[id]; ok {
			return o, nil
		}
		o, err := uc.orders.FindByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrOrderNotFound) {
				orderCache[id] = nil
				return nil, nil
			}
			return nil, err
		}
		orderCache[id] = o
		return o, nil
	}
	lookupAddr := func(userID uint64) (string, error) {
		if uc.users == nil || userID == 0 {
			return "", nil
		}
		if a, ok := addrCache[userID]; ok {
			return a, nil
		}
		u, err := uc.users.FindByID(ctx, userID)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				addrCache[userID] = ""
				return "", nil
			}
			return "", err
		}
		addr := ""
		if u != nil {
			addr = u.Address
		}
		addrCache[userID] = addr
		return addr, nil
	}
	for _, it := range items {
		if it == nil {
			continue
		}
		kv := parseRemarkKV(it.Remark)
		oid := uint64(0)
		if it.OrderID != "" {
			id, err := strconv.ParseUint(it.OrderID, 10, 64)
			if err == nil {
				oid = id
			}
		}
		if oid == 0 && kv["order"] != "" {
			id, err := strconv.ParseUint(kv["order"], 10, 64)
			if err == nil && id != 0 {
				oid = id
				it.OrderID = strconv.FormatUint(id, 10)
			}
		}
		if oid == 0 {
			if key := kv["key"]; strings.HasPrefix(key, "order=") {
				id, err := strconv.ParseUint(strings.TrimPrefix(key, "order="), 10, 64)
				if err == nil && id != 0 {
					oid = id
					it.OrderID = strconv.FormatUint(id, 10)
				}
			}
		}
		if oid != 0 {
			o, err := lookupOrder(oid)
			if err != nil {
				return err
			}
			if o != nil {
				it.OrderNo = o.DisplayNo()
				it.OrderAmount = amountStr(o.Amount)
				it.OrderTitle = o.TitleSnapshot
				it.OrderSource = FormatOrderSource(it.OrderNo, it.OrderAmount)
				addr, err := lookupAddr(o.UserID)
				if err != nil {
					return err
				}
				it.SourceAddress = addr
			}
		}
		if it.SourceAddress == "" && kv["source"] != "" {
			uid, err := strconv.ParseUint(kv["source"], 10, 64)
			if err == nil && uid != 0 {
				addr, err := lookupAddr(uid)
				if err != nil {
					return err
				}
				it.SourceAddress = addr
			}
		}
	}
	return nil
}

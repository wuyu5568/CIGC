package biz

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cigc/app/internal/conf"
	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

// MatchCap 保留旧签名；封顶只看结算金额，不再读取套餐 daily_cap。
func MatchCap(paid decimal.Decimal, _ []*Package) decimal.Decimal {
	return CapForAmount(paid)
}

// SettleRun 是一个上海自然日的日结占位与计数。
type SettleRun struct {
	ID          uint64
	SettleDate  time.Time
	Forced      bool
	UserCount   int
	CapUpdated  int
	DirectCount int
	MatchCount  int
	ManageCount int
	StaticCount int
	Remark      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SettleRunRepo 日结防重：INSERT 占位，再 Upsert 计数。
type SettleRunRepo interface {
	FindByDate(ctx context.Context, settleDate time.Time) (*SettleRun, error)
	FindLatest(ctx context.Context) (*SettleRun, error)
	TryClaim(ctx context.Context, settleDate time.Time, remark string) (claimed bool, err error)
	Upsert(ctx context.Context, run *SettleRun) error
	DeleteAfter(ctx context.Context, after time.Time) (int, error)
}

// SettleResult 一次日结摘要。
type SettleResult struct {
	Skipped         bool
	Forced          bool
	SettleDate      string
	UserCount       int
	CapUpdated      int
	DirectCount     int
	MatchCount      int
	ManageCount     int
	StaticCount     int
	OverflowCleared int
}

// SettleResetResult 把测试日拨回今天：删除今天之后的 settle_runs。
type SettleResetResult struct {
	TodayDate    string
	Deleted      int
	NextTestDate string
}

// TestDataClearResult 清测试业务数据、保留用户账户。
type TestDataClearResult struct {
	UsersKept        int
	OrdersCleared    int
	LedgerCleared    int
	HoldsCleared     int
	WithdrawsCleared int
}

// TestDataRepo 清订单/流水/冻结等，不删用户与邀请/安置。
type TestDataRepo interface {
	ClearKeepUsers(ctx context.Context) (*TestDataClearResult, error)
}

// SettleStatusView 管理端今日/最近一次日结快照。
type SettleStatusView struct {
	TodayDate    string
	TodaySettled bool
	AllowForce   bool
	NextTestDate string
	BusinessDate string
	Today        *SettleRun
	Latest       *SettleRun
}

// OrderPaidHook 订单标已支付后的秒结（直推/对碰/管理 + 按封顶解冻）。
type OrderPaidHook interface {
	OnOrderPaid(ctx context.Context, o *Order) error
}

// SettleUseCase 日结：占位 → 补漏直推/对碰/管理 → 静态释放 → 回写 cap_effective。
// 直推/对碰/管理主路径在支付时秒结。
type SettleUseCase struct {
	users      UserRepo
	packages   PackageRepo
	runs       SettleRunRepo
	orders     OrderRepo
	balances   UserBalanceRepo
	ledger     LedgerRepo
	configs    ConfigRepo
	placements PlacementRepo
	matches    MatchRepo
	daily      DailyCapRepo
	testData   TestDataRepo
	ispay      IspayPrice
	tx         TxRunner
	allowForce bool
	timezone   string
	now        func() time.Time
	mu         sync.Mutex
}

// NewSettleUseCase 构造日结用例。now 默认为 time.Now，测试可替换。
func NewSettleUseCase(
	users UserRepo,
	packages PackageRepo,
	runs SettleRunRepo,
	orders OrderRepo,
	balances UserBalanceRepo,
	ledger LedgerRepo,
	configs ConfigRepo,
	placements PlacementRepo,
	matches MatchRepo,
	daily DailyCapRepo,
	tx TxRunner,
	app *conf.App,
) *SettleUseCase {
	tz := "Asia/Shanghai"
	allowForce := false
	if app != nil {
		if app.SettleTimezone != "" {
			tz = app.SettleTimezone
		}
		allowForce = app.AllowForceSettle
	}
	if tx == nil {
		tx = NopTx{}
	}
	if daily == nil {
		daily = newMemDailyCap()
	}
	return &SettleUseCase{
		users:      users,
		packages:   packages,
		runs:       runs,
		orders:     orders,
		balances:   balances,
		ledger:     ledger,
		configs:    configs,
		placements: placements,
		matches:    matches,
		daily:      daily,
		ispay:      NewFixedIspayPrice(),
		tx:         tx,
		allowForce: allowForce,
		timezone:   tz,
		now:        time.Now,
	}
}

// SetTestData 注入测试数据清理仓储；生产由 Data 注入。仅 allow_force_settle 可调用。
func (uc *SettleUseCase) SetTestData(repo TestDataRepo) {
	if uc == nil {
		return
	}
	uc.testData = repo
}

func (uc *SettleUseCase) spot(ctx context.Context) decimal.Decimal {
	if uc != nil && uc.configs != nil {
		if p := configSpot(ctx, uc.configs); p.IsPositive() {
			v, err := uc.configs.GetValue(ctx, ConfigIspayPrice)
			if err == nil && strings.TrimSpace(v) != "" {
				return p
			}
		}
	}
	if uc != nil && uc.ispay != nil {
		if p, err := uc.ispay.Spot(ctx); err == nil && p.IsPositive() {
			return money.Round(p)
		}
	}
	return IspaySpotFallback()
}

func (uc *SettleUseCase) creditSplit(ctx context.Context, userID uint64, orderID *uint64, entryType, ispayType string, full decimal.Decimal, settleDay *time.Time, remark string) error {
	u, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if !u.IsActivated() {
		return uc.creditOverflow(ctx, userID, orderID, entryType, ispayType, full, settleDay, remark)
	}
	if isDynamicEntry(entryType) {
		return uc.creditDynamic(ctx, userID, orderID, entryType, ispayType, full, settleDay, remark)
	}
	return uc.creditHalf(ctx, userID, orderID, entryType, ispayType, full, settleDay, remark)
}

func (uc *SettleUseCase) settleDayOr(ctx context.Context, day *time.Time) time.Time {
	if day != nil && !day.IsZero() {
		return dailyCapDate(*day)
	}
	return uc.currentBusinessDay(ctx)
}

// currentBusinessDay 测试日结把日历推到未来后，新冻结/秒结按最近一次结算日记账，
// 这样 16 号产生的冻结会在 20 号到期，而不是跟 15 号一起在 19 号清掉。
func (uc *SettleUseCase) currentBusinessDay(ctx context.Context) time.Time {
	if uc == nil {
		n := time.Now()
		return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location())
	}
	today, _ := uc.settleDay(uc.now())
	if uc.runs == nil {
		return today
	}
	latest, err := uc.runs.FindLatest(ctx)
	if err != nil || latest == nil {
		return today
	}
	latestDay := uc.dayInZone(latest.SettleDate)
	if latestDay.After(today) {
		return latestDay
	}
	return today
}

func (uc *SettleUseCase) creditDynamic(ctx context.Context, userID uint64, orderID *uint64, entryType, ispayType string, full decimal.Decimal, settleDay *time.Time, remark string) error {
	full = money.Round(full)
	if !full.IsPositive() {
		return nil
	}
	u, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	day := uc.settleDayOr(ctx, settleDay)
	dayCopy := day
	cap := money.Round(u.CapEffective)
	under, overflow := full, decimal.Zero
	if uc.daily != nil {
		under, overflow, err = uc.daily.TakeUnder(ctx, userID, day, cap, full)
		if err != nil {
			return err
		}
	}
	if under.IsPositive() {
		if err := uc.creditHalf(ctx, userID, orderID, entryType, ispayType, under, &dayCopy, remark); err != nil {
			return err
		}
	}
	if overflow.IsPositive() {
		if err := uc.creditOverflow(ctx, userID, orderID, entryType, ispayType, overflow, &dayCopy, remark); err != nil {
			return err
		}
	}
	return nil
}

func (uc *SettleUseCase) creditOverflow(ctx context.Context, userID uint64, orderID *uint64, entryType, ispayType string, full decimal.Decimal, settleDay *time.Time, remark string) error {
	spot := uc.spot(ctx)
	usdt, coin := SplitHalfIspay(full, spot)
	if usdt.IsPositive() {
		if err := uc.balances.AddLockBalance(ctx, userID, usdt); err != nil {
			return err
		}
		if err := uc.ledger.Create(ctx, &LedgerEntry{
			UserID: userID, OrderID: orderID, EntryType: entryType, Amount: usdt,
			BalanceKind: BalanceLock, SettleDate: settleDay, Remark: remark,
		}); err != nil {
			return err
		}
	}
	if coin.IsPositive() {
		if err := uc.balances.AddLockIspay(ctx, userID, coin); err != nil {
			return err
		}
		if err := uc.ledger.Create(ctx, &LedgerEntry{
			UserID: userID, OrderID: orderID, EntryType: ispayType, Amount: coin,
			BalanceKind: BalanceLockIspay, SettleDate: settleDay, Remark: remark,
		}); err != nil {
			return err
		}
	}
	if uc.daily == nil || (!usdt.IsPositive() && !coin.IsPositive()) {
		return nil
	}
	now := uc.now()
	day := uc.settleDayOr(ctx, settleDay)
	return uc.daily.CreateHold(ctx, &CapOverflowHold{
		UserID:     userID,
		Value:      money.Round(full),
		USDT:       usdt,
		Ispay:      coin,
		SourceType: entryType,
		OrderID:    orderID,
		SettleDate: day,
		Remark:     remark,
		CreatedAt:  now,
	})
}

func (uc *SettleUseCase) creditHalf(ctx context.Context, userID uint64, orderID *uint64, entryType, ispayType string, full decimal.Decimal, settleDay *time.Time, remark string) error {
	u, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if !u.IsActivated() {
		return uc.creditOverflow(ctx, userID, orderID, entryType, ispayType, full, settleDay, remark)
	}
	spot := uc.spot(ctx)
	usdt, coin := SplitHalfIspay(full, spot)
	if usdt.IsPositive() {
		if err := uc.balances.AddAvailableBalance(ctx, userID, usdt); err != nil {
			return err
		}
		if err := uc.ledger.Create(ctx, &LedgerEntry{
			UserID:      userID,
			OrderID:     orderID,
			EntryType:   entryType,
			Amount:      usdt,
			BalanceKind: BalanceAvailable,
			SettleDate:  settleDay,
			Remark:      remark,
		}); err != nil {
			return err
		}
	}
	if coin.IsPositive() {
		if err := uc.balances.AddIspayBalance(ctx, userID, coin); err != nil {
			return err
		}
		if err := uc.ledger.Create(ctx, &LedgerEntry{
			UserID:      userID,
			OrderID:     orderID,
			EntryType:   ispayType,
			Amount:      coin,
			BalanceKind: BalanceIspay,
			SettleDate:  settleDay,
			Remark:      remark,
		}); err != nil {
			return err
		}
	}
	return nil
}

// AllowForce 是否允许 force=1（生产应关闭）。
func (uc *SettleUseCase) AllowForce() bool {
	return uc.allowForce
}

func (uc *SettleUseCase) location() *time.Location {
	loc, err := time.LoadLocation(uc.timezone)
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

func (uc *SettleUseCase) settleDay(now time.Time) (time.Time, string) {
	loc := uc.location()
	n := now.In(loc)
	day := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
	return day, day.Format("2006-01-02")
}

func (uc *SettleUseCase) dayInZone(t time.Time) time.Time {
	loc := uc.location()
	n := t.In(loc)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
}

// forceSettleDay 测试日结：结算「下一日」。取 max(今日+1, 最近一次结算日+1)。
func (uc *SettleUseCase) forceSettleDay(ctx context.Context) (time.Time, string, error) {
	today, _ := uc.settleDay(uc.now())
	next := today.AddDate(0, 0, 1)
	if uc.runs != nil {
		latest, err := uc.runs.FindLatest(ctx)
		if err != nil {
			return time.Time{}, "", err
		}
		if latest != nil {
			cand := uc.dayInZone(latest.SettleDate).AddDate(0, 0, 1)
			if cand.After(next) {
				next = cand
			}
		}
	}
	return next, next.Format("2006-01-02"), nil
}

// ResetTestDay 删除今天之后的测试日结记录，测试日历回到当前上海自然日。
// 不改余额、不恢复已清除的冻结批次。仅 allow_force_settle 可用。
func (uc *SettleUseCase) ResetTestDay(ctx context.Context) (*SettleResetResult, error) {
	if uc == nil {
		return nil, ErrForceSettleDisabled
	}
	uc.mu.Lock()
	defer uc.mu.Unlock()
	if !uc.allowForce {
		return nil, ErrForceSettleDisabled
	}
	today, dateStr := uc.settleDay(uc.now())
	deleted := 0
	if uc.runs != nil {
		n, err := uc.runs.DeleteAfter(ctx, today)
		if err != nil {
			return nil, err
		}
		deleted = n
	}
	_, nextStr, err := uc.forceSettleDay(ctx)
	if err != nil {
		return nil, err
	}
	return &SettleResetResult{TodayDate: dateStr, Deleted: deleted, NextTestDate: nextStr}, nil
}

// ClearTestData 清订单、流水、冻结、日结等测试数据，保留全部用户及邀请/安置。
// 余额、已支付、日封顶归零。仅 allow_force_settle 可用。
func (uc *SettleUseCase) ClearTestData(ctx context.Context) (*TestDataClearResult, error) {
	if uc == nil {
		return nil, ErrForceSettleDisabled
	}
	uc.mu.Lock()
	defer uc.mu.Unlock()
	if !uc.allowForce {
		return nil, ErrForceSettleDisabled
	}
	if uc.testData == nil {
		return nil, ErrForceSettleDisabled
	}
	return uc.testData.ClearKeepUsers(ctx)
}

func (uc *SettleUseCase) displaySettleDate(ctx context.Context) string {
	return uc.currentBusinessDay(ctx).Format("2006-01-02")
}

type orderStaticAcc struct {
	todayUSDT, todayIspay, releasedUSDT, releasedIspay decimal.Decimal
	releasedDays                                       int
	lastSettle                                         string
}

// OrderStaticReleases 批量算订单静态：今日 / 已释放 / 待释放（各一半 U、一半 ispay）。
func (uc *SettleUseCase) OrderStaticReleases(ctx context.Context, orders []*Order) (map[uint64]OrderStaticRelease, error) {
	out := make(map[uint64]OrderStaticRelease, len(orders))
	if len(orders) == 0 {
		return out, nil
	}
	ids := make([]uint64, 0, len(orders))
	for _, o := range orders {
		if o == nil {
			continue
		}
		ids = append(ids, o.ID)
		out[o.ID] = OrderStaticRelease{}
	}
	byOrder := map[uint64]*orderStaticAcc{}
	if uc.ledger != nil && len(ids) > 0 {
		entries, err := uc.ledger.ListByOrderIDsAndTypes(ctx, ids, []string{LedgerStatic, LedgerStaticIspay})
		if err != nil {
			return nil, err
		}
		today := uc.displaySettleDate(ctx)
		for _, e := range entries {
			if e == nil || e.OrderID == nil {
				continue
			}
			a := byOrder[*e.OrderID]
			if a == nil {
				a = &orderStaticAcc{}
				byOrder[*e.OrderID] = a
			}
			day := ""
			if e.SettleDate != nil {
				day = e.SettleDate.Format("2006-01-02")
			}
			if day != "" && day > a.lastSettle {
				a.lastSettle = day
			}
			switch e.EntryType {
			case LedgerStatic:
				a.releasedUSDT = a.releasedUSDT.Add(e.Amount)
				a.releasedDays++
				if day == today {
					a.todayUSDT = a.todayUSDT.Add(e.Amount)
				}
			case LedgerStaticIspay:
				a.releasedIspay = a.releasedIspay.Add(e.Amount)
				if day == today {
					a.todayIspay = a.todayIspay.Add(e.Amount)
				}
			}
		}
	}
	spot := uc.spot(ctx)
	for _, o := range orders {
		if o == nil {
			continue
		}
		a := byOrder[o.ID]
		if a == nil {
			a = &orderStaticAcc{}
		}
		rel := ComputeOrderStaticRelease(o, spot, a.todayUSDT, a.todayIspay, a.releasedUSDT, a.releasedIspay, a.releasedDays)
		rel.SettleDate = a.lastSettle
		out[o.ID] = rel
	}
	return out, nil
}

func (uc *SettleUseCase) directRate(ctx context.Context) decimal.Decimal {
	raw := defaultDirectRate
	if uc.configs != nil {
		if v, err := uc.configs.GetValue(ctx, ConfigDirectRate); err == nil && v != "" {
			raw = v
		}
	}
	d, err := decimal.NewFromString(raw)
	if err != nil {
		d = decimal.RequireFromString(defaultDirectRate)
	}
	return d
}

func (uc *SettleUseCase) matchRate(ctx context.Context) decimal.Decimal {
	raw := defaultMatchRate
	if uc.configs != nil {
		if v, err := uc.configs.GetValue(ctx, ConfigMatchRate); err == nil && v != "" {
			raw = v
		}
	}
	d, err := decimal.NewFromString(raw)
	if err != nil {
		d = decimal.RequireFromString(defaultMatchRate)
	}
	return d
}

func (uc *SettleUseCase) manageRate(ctx context.Context) decimal.Decimal {
	raw := defaultManageRate
	if uc.configs != nil {
		if v, err := uc.configs.GetValue(ctx, ConfigManageRate); err == nil && v != "" {
			raw = v
		}
	}
	d, err := decimal.NewFromString(raw)
	if err != nil {
		d = decimal.RequireFromString(defaultManageRate)
	}
	return d
}

func (uc *SettleUseCase) manageGens(ctx context.Context) int {
	return ConfigIntValue(ctx, uc.configs, ConfigManageGens, defaultManageGens, 1, maxManageGens)
}

func (uc *SettleUseCase) overflowClearHours(ctx context.Context) int {
	return ConfigIntValue(ctx, uc.configs, ConfigOverflowHours, OverflowClearHours, minOverflowHours, maxOverflowHours)
}

// OnOrderPaid 支付瞬间：立刻抬升封顶、按本单封顶解冻，并秒结直推/对碰/管理。
func (uc *SettleUseCase) OnOrderPaid(ctx context.Context, o *Order) error {
	if uc == nil || o == nil || o.ID == 0 || o.UserID == 0 {
		return nil
	}
	LoadCapTiersFromRepo(ctx, uc.configs)
	if err := ApplyPaidOrderCap(ctx, uc.users, uc.packages, o.UserID, o.Amount); err != nil {
		return err
	}
	u, err := uc.users.FindByID(ctx, o.UserID)
	if err != nil {
		return err
	}
	if money.Round(u.PaidAmount).Equal(money.Round(o.Amount)) {
		if err := uc.wrapUnheldLockAsOverflow(ctx, o.UserID); err != nil {
			return err
		}
	}
	if err := uc.releaseOverflowByOrderCap(ctx, o.UserID, o.Amount); err != nil {
		return err
	}
	cap := orderActivateCap(ctx, uc.packages, o.Amount)
	if err := uc.releaseLockExcludingOverflow(ctx, o.UserID, cap, nil, remarkActivateUSDT, remarkActivateIspay); err != nil {
		return err
	}
	settleDay := uc.currentBusinessDay(ctx)
	dayCopy := settleDay
	if _, err := uc.creditDirectForOrder(ctx, o, uc.directRate(ctx), &dayCopy); err != nil {
		return err
	}
	if uc.matches == nil || uc.placements == nil {
		return nil
	}
	if err := uc.applyOneMatchOrder(ctx, o, settleDay); err != nil {
		return err
	}
	parents, err := uc.placementParents(ctx)
	if err != nil {
		return err
	}
	rate := uc.matchRate(ctx)
	oid := o.ID
	seen := map[uint64]struct{}{}
	for _, a := range matchAncestorAdds(parents, o.UserID, money.Round(o.Amount)) {
		if _, ok := seen[a.userID]; ok {
			continue
		}
		seen[a.userID] = struct{}{}
		if _, err := uc.creditMatchForUser(ctx, a.userID, rate, &dayCopy, &oid); err != nil {
			return err
		}
	}
	if _, err := uc.payManageRewards(ctx, settleDay); err != nil {
		return err
	}
	return nil
}

// Run 普通跑结算当前上海自然日；测试 force 结算下一日（最近结算日或今日的次日），并按该日 0:00 清除到期冻结。
// 顺序：封账昨日超额并清除到期批次 → 回写 cap_effective → 补漏直推/对碰/管理 → 静态释放 → 按当日封顶解冻剩余非超额冻结。
func (uc *SettleUseCase) Run(ctx context.Context, force bool) (*SettleResult, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	LoadCapTiersFromRepo(ctx, uc.configs)

	if force && !uc.allowForce {
		return nil, ErrForceSettleDisabled
	}

	settleDay, dateStr := uc.settleDay(uc.now())
	if force {
		var err error
		settleDay, dateStr, err = uc.forceSettleDay(ctx)
		if err != nil {
			return nil, err
		}
	}
	if uc.runs != nil && !force {
		claimed, err := uc.runs.TryClaim(ctx, settleDay, "settle claim")
		if err != nil {
			return nil, err
		}
		if !claimed {
			return &SettleResult{Skipped: true, SettleDate: dateStr}, nil
		}
	}

	result := &SettleResult{Forced: force, SettleDate: dateStr}

	expireAt := uc.now()
	if force {
		// 测试日结算按该结算日 0:00 判断冻结到期，不必等真实时钟过配置的小时数。
		expireAt = settleDay
	}
	cleared, err := uc.expireCapOverflowAt(ctx, expireAt)
	if err != nil {
		return nil, err
	}
	result.OverflowCleared = cleared
	// force 结算日可能晚于时钟日，再按本批结算日封账一次。
	if err := uc.packageOverflowLots(ctx, settleDay); err != nil {
		return nil, err
	}

	capRes, err := uc.refreshCaps(ctx)
	if err != nil {
		return nil, err
	}
	result.UserCount = capRes.UserCount
	result.CapUpdated = capRes.CapUpdated

	directN, err := uc.payDirectRewards(ctx, settleDay)
	if err != nil {
		return nil, err
	}
	result.DirectCount = directN

	matchN, err := uc.payMatchRewards(ctx, settleDay)
	if err != nil {
		return nil, err
	}
	result.MatchCount = matchN

	manageN, err := uc.payManageRewards(ctx, settleDay)
	if err != nil {
		return nil, err
	}
	result.ManageCount = manageN

	staticN, err := uc.payStaticRewards(ctx, settleDay)
	if err != nil {
		return nil, err
	}
	result.StaticCount = staticN

	if err := uc.releaseDailyLock(ctx, settleDay); err != nil {
		return nil, err
	}

	if uc.runs != nil {
		remark := "settle"
		if force {
			remark = "settle force=1 (test only)"
		}
		if err := uc.runs.Upsert(ctx, &SettleRun{
			SettleDate:  settleDay,
			Forced:      force,
			UserCount:   result.UserCount,
			CapUpdated:  result.CapUpdated,
			DirectCount: result.DirectCount,
			MatchCount:  result.MatchCount,
			ManageCount: result.ManageCount,
			StaticCount: result.StaticCount,
			Remark:      remark,
		}); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (uc *SettleUseCase) payDirectRewards(ctx context.Context, settleDay time.Time) (int, error) {
	if uc.orders == nil || uc.balances == nil || uc.ledger == nil {
		return 0, nil
	}
	from := settleDay
	to := settleDay.AddDate(0, 0, 1)
	orders, err := uc.orders.ListPaidBetween(ctx, from, to)
	if err != nil {
		return 0, err
	}
	rate := uc.directRate(ctx)
	count := 0
	dayCopy := settleDay
	for _, o := range orders {
		if o == nil || o.Status != OrderPaid {
			continue
		}
		paid, err := uc.creditDirectForOrder(ctx, o, rate, &dayCopy)
		if err != nil {
			return count, err
		}
		if paid {
			count++
		}
	}
	return count, nil
}

func (uc *SettleUseCase) creditDirectForOrder(ctx context.Context, o *Order, rate decimal.Decimal, settleDay *time.Time) (bool, error) {
	exists, err := uc.ledger.ExistsByOrderAndType(ctx, o.ID, LedgerDirect)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	buyer, err := uc.users.FindByID(ctx, o.UserID)
	if err != nil {
		return false, err
	}
	if buyer.InviterID == nil {
		return false, nil
	}
	inviterID := *buyer.InviterID
	reward := money.Round(money.Round(o.Amount).Mul(rate))
	if !reward.IsPositive() {
		return false, nil
	}
	oid := o.ID
	created := false
	err = uc.tx.InTx(ctx, func(ctx context.Context) error {
		ok, err := uc.ledger.ExistsByOrderAndType(ctx, o.ID, LedgerDirect)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if err := uc.creditSplit(ctx, inviterID, &oid, LedgerDirect, LedgerDirectIspay, reward, settleDay, fmt.Sprintf("direct order=%d rate=%s", o.ID, rate.String())); err != nil {
			return err
		}
		created = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return created, nil
}

func (uc *SettleUseCase) payMatchRewards(ctx context.Context, settleDay time.Time) (int, error) {
	if uc.matches == nil || uc.placements == nil || uc.orders == nil || uc.ledger == nil {
		return 0, nil
	}
	if err := uc.applyTodayMatchVolume(ctx, settleDay); err != nil {
		return 0, err
	}
	bals, err := uc.matches.ListAll(ctx)
	if err != nil {
		return 0, err
	}
	rate := uc.matchRate(ctx)
	count := 0
	dayCopy := settleDay
	for _, b := range bals {
		paid, err := uc.creditMatchForUser(ctx, b.UserID, rate, &dayCopy, nil)
		if err != nil {
			return count, err
		}
		if paid {
			count++
		}
	}
	return count, nil
}

func (uc *SettleUseCase) applyTodayMatchVolume(ctx context.Context, settleDay time.Time) error {
	from := settleDay
	to := settleDay.AddDate(0, 0, 1)
	orders, err := uc.orders.ListPaidBetween(ctx, from, to)
	if err != nil {
		return err
	}
	for _, o := range orders {
		if err := uc.applyOneMatchOrder(ctx, o, settleDay); err != nil {
			return err
		}
	}
	return nil
}

func (uc *SettleUseCase) applyOneMatchOrder(ctx context.Context, o *Order, settleDay time.Time) error {
	if o == nil || o.Status != OrderPaid || !o.Amount.IsPositive() {
		return nil
	}
	parents, err := uc.placementParents(ctx)
	if err != nil {
		return err
	}
	adds := matchAncestorAdds(parents, o.UserID, money.Round(o.Amount))
	return uc.tx.InTx(ctx, func(ctx context.Context) error {
		applied, err := uc.matches.TryApplyOrder(ctx, o.ID, settleDay)
		if err != nil {
			return err
		}
		if !applied {
			return nil
		}
		for _, a := range adds {
			if err := uc.matches.AddRemain(ctx, a.userID, a.side, a.delta); err != nil {
				return err
			}
		}
		return nil
	})
}

type matchAdd struct {
	userID uint64
	side   string
	delta  decimal.Decimal
}

func (uc *SettleUseCase) placementParents(ctx context.Context) (map[uint64]*Placement, error) {
	rows, err := uc.placements.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[uint64]*Placement, len(rows))
	for _, p := range rows {
		if p != nil {
			out[p.UserID] = p
		}
	}
	return out, nil
}

func matchAncestorAdds(parents map[uint64]*Placement, buyerID uint64, amount decimal.Decimal) []matchAdd {
	if amount.IsZero() || !amount.IsPositive() {
		return nil
	}
	var adds []matchAdd
	cur := buyerID
	seen := map[uint64]struct{}{}
	for i := 0; i < maxMatchAncestorWalk; i++ {
		if _, ok := seen[cur]; ok {
			break
		}
		seen[cur] = struct{}{}
		p := parents[cur]
		if p == nil {
			break
		}
		adds = append(adds, matchAdd{userID: p.SponsorID, side: p.Side, delta: amount})
		cur = p.SponsorID
	}
	return adds
}

func (uc *SettleUseCase) creditMatchForUser(ctx context.Context, userID uint64, rate decimal.Decimal, settleDay *time.Time, orderID *uint64) (bool, error) {
	credited := false
	err := uc.tx.InTx(ctx, func(ctx context.Context) error {
		u, err := uc.users.FindByID(ctx, userID)
		if err != nil {
			return err
		}
		bal, err := uc.matches.Get(ctx, userID)
		if err != nil {
			return err
		}
		credit, pair, newL, newR := MatchPair(bal.LeftRemain, bal.RightRemain, rate)
		if !pair.IsPositive() {
			return nil
		}
		if err := uc.matches.SaveRemains(ctx, userID, newL, newR); err != nil {
			return err
		}
		remark := fmt.Sprintf("match pair=%s rate=%s credit=%s cap=%s", pair.String(), rate.String(), credit.String(), money.Round(u.CapEffective).String())
		if !credit.IsPositive() {
			return nil
		}
		if err := uc.creditSplit(ctx, userID, orderID, LedgerMatch, LedgerMatchIspay, credit, settleDay, remark); err != nil {
			return err
		}
		credited = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return credited, nil
}

func (uc *SettleUseCase) payManageRewards(ctx context.Context, settleDay time.Time) (int, error) {
	if uc.ledger == nil || uc.balances == nil {
		return 0, nil
	}
	matches, err := uc.ledger.ListByTypeAndDate(ctx, LedgerMatch, settleDay)
	if err != nil {
		return 0, err
	}
	rate := uc.manageRate(ctx)
	want := uc.manageGens(ctx)
	count := 0
	dayCopy := settleDay
	seen := map[string]struct{}{}
	for _, m := range matches {
		if m == nil {
			continue
		}
		uniq := manageUniqFromMatch(m)
		key := fmt.Sprintf("%d:%s", m.UserID, uniq)
		if _, ok := seen[key]; ok {
			continue
		}
		full := MatchLedgerCredit(m)
		if !full.IsPositive() {
			continue
		}
		seen[key] = struct{}{}
		pool := money.Round(full.Mul(rate))
		if !pool.IsPositive() {
			continue
		}
		shares := ManageShares(pool, want)
		ancestors, err := FindActivatedRecommendAncestors(ctx, uc.users, m.UserID, want)
		if err != nil {
			return count, err
		}
		for i, ancestorID := range ancestors {
			if i >= len(shares) || !shares[i].IsPositive() {
				continue
			}
			paid, err := uc.creditManage(ctx, ancestorID, shares[i], i+1, m.UserID, uniq, &dayCopy)
			if err != nil {
				return count, err
			}
			if paid {
				count++
			}
		}
	}
	return count, nil
}

func (uc *SettleUseCase) creditManage(ctx context.Context, userID uint64, amount decimal.Decimal, gen int, sourceUserID uint64, uniq string, settleDay *time.Time) (bool, error) {
	remark := manageRemark(gen, sourceUserID, uniq)
	exists, err := uc.ledger.ExistsByUserTypeDateRemark(ctx, userID, LedgerManage, *settleDay, remark)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	created := false
	err = uc.tx.InTx(ctx, func(ctx context.Context) error {
		ok, err := uc.ledger.ExistsByUserTypeDateRemark(ctx, userID, LedgerManage, *settleDay, remark)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if err := uc.creditSplit(ctx, userID, nil, LedgerManage, LedgerManageIspay, amount, settleDay, remark); err != nil {
			return err
		}
		created = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return created, nil
}

func (uc *SettleUseCase) payStaticRewards(ctx context.Context, settleDay time.Time) (int, error) {
	if uc.orders == nil || uc.balances == nil || uc.ledger == nil {
		return 0, nil
	}
	to := settleDay.AddDate(0, 0, 1)
	orders, err := uc.orders.ListPaidBefore(ctx, to)
	if err != nil {
		return 0, err
	}
	spot := uc.spot(ctx)
	count := 0
	dayCopy := settleDay
	for _, o := range orders {
		if o == nil || o.Status != OrderPaid || !ValidReleaseDays(o.ReleaseDays) {
			continue
		}
		paid, err := uc.creditStaticForOrder(ctx, o, spot, &dayCopy)
		if err != nil {
			return count, err
		}
		if paid {
			count++
		}
	}
	return count, nil
}

func (uc *SettleUseCase) creditStaticForOrder(ctx context.Context, o *Order, spot decimal.Decimal, settleDay *time.Time) (bool, error) {
	exists, err := uc.ledger.ExistsByOrderTypeAndDate(ctx, o.ID, LedgerStatic, *settleDay)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	n, err := uc.ledger.CountByOrderAndType(ctx, o.ID, LedgerStatic)
	if err != nil {
		return false, err
	}
	if n >= o.ReleaseDays {
		return false, nil
	}
	_, _, dailyValue, _, _, ok := StaticDaily(o.Amount, o.ReleaseDays, spot)
	if !ok || !dailyValue.IsPositive() {
		return false, nil
	}
	oid := o.ID
	created := false
	err = uc.tx.InTx(ctx, func(ctx context.Context) error {
		ok, err := uc.ledger.ExistsByOrderTypeAndDate(ctx, o.ID, LedgerStatic, *settleDay)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		n, err := uc.ledger.CountByOrderAndType(ctx, o.ID, LedgerStatic)
		if err != nil {
			return err
		}
		if n >= o.ReleaseDays {
			return nil
		}
		remark := fmt.Sprintf("static order=%d days=%d day=%d", o.ID, o.ReleaseDays, n+1)
		if err := uc.creditSplit(ctx, o.UserID, &oid, LedgerStatic, LedgerStaticIspay, dailyValue, settleDay, remark); err != nil {
			return err
		}
		created = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return created, nil
}

func (uc *SettleUseCase) refreshCaps(ctx context.Context) (*SettleResult, error) {
	LoadCapTiersFromRepo(ctx, uc.configs)
	users, err := uc.users.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	maxPaid := map[uint64]decimal.Decimal{}
	if uc.orders != nil {
		maxPaid, err = uc.orders.MaxPaidAmountByUser(ctx)
		if err != nil {
			return nil, err
		}
	}
	result := &SettleResult{UserCount: len(users)}
	for _, u := range users {
		want := CapForAmount(maxPaid[u.ID])
		cur := money.Round(u.CapEffective)
		if want.Equal(cur) {
			continue
		}
		if err := uc.users.SetCapEffective(ctx, u.ID, want); err != nil {
			return nil, err
		}
		result.CapUpdated++
	}
	return result, nil
}

func (uc *SettleUseCase) releaseDailyLock(ctx context.Context, settleDay time.Time) error {
	if uc.users == nil || uc.balances == nil || uc.ledger == nil {
		return nil
	}
	users, err := uc.users.ListAll(ctx)
	if err != nil {
		return err
	}
	dayCopy := settleDay
	for _, u := range users {
		if u == nil || !u.IsActivated() {
			continue
		}
		if !money.Round(u.LockBalance).IsPositive() && !money.Round(u.LockIspay).IsPositive() {
			continue
		}
		ok, err := uc.ledger.ExistsByUserTypeDateRemark(ctx, u.ID, LedgerActivate, settleDay, remarkDailyLockUSDT)
		if err != nil {
			return err
		}
		if ok {
			continue
		}
		cap := money.Round(u.CapEffective)
		if err := uc.releaseLockExcludingOverflow(ctx, u.ID, cap, &dayCopy, remarkDailyLockUSDT, remarkDailyLockIspay); err != nil {
			return err
		}
	}
	return nil
}

func (uc *SettleUseCase) wrapUnheldLockAsOverflow(ctx context.Context, userID uint64) error {
	if uc == nil || uc.daily == nil || userID == 0 {
		return nil
	}
	u, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	ou, oi, err := uc.overflowReserved(ctx, userID)
	if err != nil {
		return err
	}
	ru := money.Round(u.LockBalance.Sub(ou))
	ri := money.Round(u.LockIspay.Sub(oi))
	if ru.IsNegative() {
		ru = decimal.Zero
	}
	if ri.IsNegative() {
		ri = decimal.Zero
	}
	if !ru.IsPositive() && !ri.IsPositive() {
		return nil
	}
	spot := uc.spot(ctx)
	value := money.Round(ru.Add(ri.Mul(spot)))
	if !value.IsPositive() {
		value = ru
		if !value.IsPositive() {
			value = ri
		}
	}
	day := uc.settleDayOr(ctx, nil)
	return uc.daily.CreateHold(ctx, &CapOverflowHold{
		UserID:     userID,
		Value:      value,
		USDT:       ru,
		Ispay:      ri,
		SourceType: overflowSourceInactive,
		SettleDate: day,
		Remark:     remarkInactiveWrap,
		CreatedAt:  uc.now(),
	})
}

func (uc *SettleUseCase) overflowReserved(ctx context.Context, userID uint64) (decimal.Decimal, decimal.Decimal, error) {
	if uc.daily == nil || userID == 0 {
		return decimal.Zero, decimal.Zero, nil
	}
	return uc.daily.ActiveTotals(ctx, userID)
}

func (uc *SettleUseCase) releaseLockExcludingOverflow(ctx context.Context, userID uint64, cap decimal.Decimal, settleDay *time.Time, usdtRemark, ispayRemark string) error {
	ru, ri, err := uc.overflowReserved(ctx, userID)
	if err != nil {
		return err
	}
	return ReleaseLockByCapReserved(ctx, uc.users, uc.balances, uc.ledger, userID, cap, ru, ri, settleDay, usdtRemark, ispayRemark)
}

func (uc *SettleUseCase) releaseOverflowByOrderCap(ctx context.Context, userID uint64, orderAmount decimal.Decimal) error {
	if uc == nil || uc.daily == nil || uc.users == nil || userID == 0 {
		return nil
	}
	quota := orderActivateCap(ctx, uc.packages, orderAmount)
	if !quota.IsPositive() {
		return nil
	}
	return uc.tx.InTx(ctx, func(ctx context.Context) error {
		u, err := uc.users.FindByID(ctx, userID)
		if err != nil {
			return err
		}
		if !u.IsActivated() {
			return nil
		}
		day := uc.currentBusinessDay(ctx)
		holds, err := uc.daily.ListActiveHolds(ctx, userID)
		if err != nil {
			return err
		}
		remain := money.Round(quota)
		now := uc.now()
		for _, h := range holds {
			if h == nil || !remain.IsPositive() {
				break
			}
			cur, err := uc.daily.GetHold(ctx, h.ID)
			if err != nil {
				return err
			}
			if cur == nil || cur.ReleasedAt != nil || cur.BurnedAt != nil || !cur.Value.IsPositive() {
				continue
			}
			take := cur.Value
			if take.GreaterThan(remain) {
				take = remain
			}
			moveUSDT, moveIspay := overflowTakeParts(cur, take)
			if moveUSDT.IsPositive() {
				if err := uc.balances.SubLockBalance(ctx, userID, moveUSDT); err != nil {
					return err
				}
				if err := uc.balances.AddAvailableBalance(ctx, userID, moveUSDT); err != nil {
					return err
				}
				if err := uc.ledger.Create(ctx, &LedgerEntry{
					UserID: userID, OrderID: cur.OrderID, EntryType: LedgerActivate,
					Amount: moveUSDT.Neg(), BalanceKind: BalanceLock, SettleDate: &day,
					Remark: remarkOverflowOrderUnlock,
				}); err != nil {
					return err
				}
				if err := uc.ledger.Create(ctx, &LedgerEntry{
					UserID: userID, OrderID: cur.OrderID, EntryType: LedgerActivate,
					Amount: moveUSDT, BalanceKind: BalanceAvailable, SettleDate: &day,
					Remark: remarkOverflowOrderUnlock,
				}); err != nil {
					return err
				}
			}
			if moveIspay.IsPositive() {
				if err := uc.balances.SubLockIspay(ctx, userID, moveIspay); err != nil {
					return err
				}
				if err := uc.balances.AddIspayBalance(ctx, userID, moveIspay); err != nil {
					return err
				}
				if err := uc.ledger.Create(ctx, &LedgerEntry{
					UserID: userID, OrderID: cur.OrderID, EntryType: LedgerActivateIspay,
					Amount: moveIspay.Neg(), BalanceKind: BalanceLockIspay, SettleDate: &day,
					Remark: remarkOverflowOrderUnlock,
				}); err != nil {
					return err
				}
				if err := uc.ledger.Create(ctx, &LedgerEntry{
					UserID: userID, OrderID: cur.OrderID, EntryType: LedgerActivateIspay,
					Amount: moveIspay, BalanceKind: BalanceIspay, SettleDate: &day,
					Remark: remarkOverflowOrderUnlock,
				}); err != nil {
					return err
				}
			}
			cur.Value = money.Round(cur.Value.Sub(take))
			cur.USDT = money.Round(cur.USDT.Sub(moveUSDT))
			cur.Ispay = money.Round(cur.Ispay.Sub(moveIspay))
			if !cur.Value.IsPositive() {
				t := now
				cur.ReleasedAt = &t
				cur.Value = decimal.Zero
				cur.USDT = decimal.Zero
				cur.Ispay = decimal.Zero
			}
			if err := uc.daily.SaveHold(ctx, cur); err != nil {
				return err
			}
			remain = money.Round(remain.Sub(take))
		}
		return nil
	})
}

func overflowTakeParts(h *CapOverflowHold, take decimal.Decimal) (usdt, ispay decimal.Decimal) {
	if h == nil {
		return decimal.Zero, decimal.Zero
	}
	take = money.Round(take)
	if !take.IsPositive() || !h.Value.IsPositive() {
		return decimal.Zero, decimal.Zero
	}
	if take.GreaterThanOrEqual(h.Value) {
		return money.Round(h.USDT), money.Round(h.Ispay)
	}
	usdt = money.Round(h.USDT.Mul(take).Div(h.Value))
	ispay = money.Round(h.Ispay.Mul(take).Div(h.Value))
	return usdt, ispay
}

func (uc *SettleUseCase) packageOverflowLots(ctx context.Context, settleDay time.Time) error {
	if uc == nil || uc.daily == nil {
		return nil
	}
	return uc.daily.StampUnpackaged(ctx, dailyCapDate(settleDay), uc.overflowClearHours(ctx))
}

// ExpireCapOverflow 按当前时钟封账昨日未打包超额，再清除已到期批次：只减冻结、不转入可提现。
func (uc *SettleUseCase) ExpireCapOverflow(ctx context.Context) (int, error) {
	return uc.expireCapOverflowAt(ctx, uc.now())
}

func (uc *SettleUseCase) expireCapOverflowAt(ctx context.Context, now time.Time) (int, error) {
	if uc == nil || uc.daily == nil {
		return 0, nil
	}
	day, _ := uc.settleDay(now)
	if err := uc.packageOverflowLots(ctx, day); err != nil {
		return 0, err
	}
	holds, err := uc.daily.ListExpired(ctx, now, 200)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, h := range holds {
		if h == nil {
			continue
		}
		err := uc.tx.InTx(ctx, func(ctx context.Context) error {
			cur, err := uc.daily.GetHold(ctx, h.ID)
			if err != nil {
				return err
			}
			if cur == nil || cur.ReleasedAt != nil || cur.BurnedAt != nil || !cur.Value.IsPositive() {
				return nil
			}
			if !holdIsPackaged(cur) || cur.ExpiresAt.After(now) {
				return nil
			}
			if cur.USDT.IsPositive() {
				if err := uc.balances.SubLockBalance(ctx, cur.UserID, cur.USDT); err != nil {
					return err
				}
				if err := uc.ledger.Create(ctx, &LedgerEntry{
					UserID: cur.UserID, OrderID: cur.OrderID, EntryType: LedgerActivate,
					Amount: cur.USDT.Neg(), BalanceKind: BalanceLock, SettleDate: &day,
					Remark: remarkOverflowClear72h,
				}); err != nil {
					return err
				}
			}
			if cur.Ispay.IsPositive() {
				if err := uc.balances.SubLockIspay(ctx, cur.UserID, cur.Ispay); err != nil {
					return err
				}
				if err := uc.ledger.Create(ctx, &LedgerEntry{
					UserID: cur.UserID, OrderID: cur.OrderID, EntryType: LedgerActivateIspay,
					Amount: cur.Ispay.Neg(), BalanceKind: BalanceLockIspay, SettleDate: &day,
					Remark: remarkOverflowClear72h,
				}); err != nil {
					return err
				}
			}
			t := now
			cur.BurnedAt = &t
			cur.Value = decimal.Zero
			cur.USDT = decimal.Zero
			cur.Ispay = decimal.Zero
			if err := uc.daily.SaveHold(ctx, cur); err != nil {
				return err
			}
			n++
			return nil
		})
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// Status 返回今日是否已占位及最近一次记录。
func (uc *SettleUseCase) Status(ctx context.Context) (*SettleStatusView, error) {
	settleDay, dateStr := uc.settleDay(uc.now())
	view := &SettleStatusView{
		TodayDate:    dateStr,
		AllowForce:   uc.allowForce,
		BusinessDate: uc.currentBusinessDay(ctx).Format("2006-01-02"),
	}
	if next, nextStr, err := uc.forceSettleDay(ctx); err == nil {
		_ = next
		view.NextTestDate = nextStr
	}
	if uc.runs == nil {
		return view, nil
	}
	today, err := uc.runs.FindByDate(ctx, settleDay)
	if err != nil {
		return nil, err
	}
	view.Today = today
	view.TodaySettled = today != nil
	latest, err := uc.runs.FindLatest(ctx)
	if err != nil {
		return nil, err
	}
	view.Latest = latest
	return view, nil
}

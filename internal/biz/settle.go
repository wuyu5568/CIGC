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

// MatchCap 按单笔订单金额向下匹配套餐档位，返回该档 daily_cap。
// 封顶取用户最大已付单，不按 paid_amount 累加。已下架套餐仍参与匹配。
func MatchCap(paid decimal.Decimal, pkgs []*Package) decimal.Decimal {
	paid = money.Round(paid)
	var best *Package
	for _, p := range pkgs {
		if p == nil {
			continue
		}
		amt := money.Round(p.Amount)
		if amt.GreaterThan(paid) {
			continue
		}
		if best == nil || amt.GreaterThan(money.Round(best.Amount)) {
			best = p
		}
	}
	if best == nil {
		return decimal.Zero
	}
	return money.Round(best.DailyCap)
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
}

// SettleResult 一次日结摘要。
type SettleResult struct {
	Skipped     bool
	Forced      bool
	SettleDate  string
	UserCount   int
	CapUpdated  int
	DirectCount int
	MatchCount  int
	ManageCount int
	StaticCount int
}

// SettleStatusView 管理端今日/最近一次日结快照。
type SettleStatusView struct {
	TodayDate    string
	TodaySettled bool
	AllowForce   bool
	NextTestDate string
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
		ispay:      NewFixedIspayPrice(),
		tx:         tx,
		allowForce: allowForce,
		timezone:   tz,
		now:        time.Now,
	}
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
	spot := uc.spot(ctx)
	usdt, coin := SplitHalfIspay(full, spot)
	u, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	active := u.IsActivated()
	if usdt.IsPositive() {
		kind := BalanceLock
		if active {
			kind = BalanceAvailable
			if err := uc.balances.AddAvailableBalance(ctx, userID, usdt); err != nil {
				return err
			}
		} else if err := uc.balances.AddLockBalance(ctx, userID, usdt); err != nil {
			return err
		}
		if err := uc.ledger.Create(ctx, &LedgerEntry{
			UserID:      userID,
			OrderID:     orderID,
			EntryType:   entryType,
			Amount:      usdt,
			BalanceKind: kind,
			SettleDate:  settleDay,
			Remark:      remark,
		}); err != nil {
			return err
		}
	}
	if coin.IsPositive() {
		kind := BalanceLockIspay
		if active {
			kind = BalanceIspay
			if err := uc.balances.AddIspayBalance(ctx, userID, coin); err != nil {
				return err
			}
		} else if err := uc.balances.AddLockIspay(ctx, userID, coin); err != nil {
			return err
		}
		if err := uc.ledger.Create(ctx, &LedgerEntry{
			UserID:      userID,
			OrderID:     orderID,
			EntryType:   ispayType,
			Amount:      coin,
			BalanceKind: kind,
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

func (uc *SettleUseCase) displaySettleDate(ctx context.Context) string {
	_, today := uc.settleDay(uc.now())
	if uc.runs == nil {
		return today
	}
	latest, err := uc.runs.FindLatest(ctx)
	if err != nil || latest == nil {
		return today
	}
	latestStr := uc.dayInZone(latest.SettleDate).Format("2006-01-02")
	if latestStr > today {
		return latestStr
	}
	return today
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

// OnOrderPaid 支付瞬间：立刻抬升封顶、按本单封顶解冻，并秒结直推/对碰/管理。
func (uc *SettleUseCase) OnOrderPaid(ctx context.Context, o *Order) error {
	if uc == nil || o == nil || o.ID == 0 || o.UserID == 0 {
		return nil
	}
	if err := ApplyPaidOrderCap(ctx, uc.users, uc.packages, o.UserID, o.Amount); err != nil {
		return err
	}
	cap := orderActivateCap(ctx, uc.packages, o.Amount)
	if err := ReleaseInactiveLock(ctx, uc.users, uc.balances, uc.ledger, o.UserID, cap); err != nil {
		return err
	}
	settleDay, _ := uc.settleDay(uc.now())
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

// Run 普通跑结算当前上海自然日；测试 force 结算下一日（最近结算日或今日的次日）。
// 顺序：补漏直推 → 补漏对碰 → 补漏管理奖 → 静态释放 → 回写封顶。
func (uc *SettleUseCase) Run(ctx context.Context, force bool) (*SettleResult, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

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

	capRes, err := uc.refreshCaps(ctx)
	if err != nil {
		return nil, err
	}
	result.UserCount = capRes.UserCount
	result.CapUpdated = capRes.CapUpdated
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
	u, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}
	credited := false
	err = uc.tx.InTx(ctx, func(ctx context.Context) error {
		bal, err := uc.matches.Get(ctx, userID)
		if err != nil {
			return err
		}
		credit, pair, newL, newR := MatchPair(bal.LeftRemain, bal.RightRemain, rate, u.CapEffective)
		if !pair.IsPositive() {
			return nil
		}
		if err := uc.matches.SaveRemains(ctx, userID, newL, newR); err != nil {
			return err
		}
		remark := fmt.Sprintf("match pair=%s rate=%s cap=%s", pair.String(), rate.String(), money.Round(u.CapEffective).String())
		if credit.IsPositive() {
			if err := uc.creditSplit(ctx, userID, orderID, LedgerMatch, LedgerMatchIspay, credit, settleDay, remark); err != nil {
				return err
			}
			credited = true
			return nil
		}
		return uc.ledger.Create(ctx, &LedgerEntry{
			UserID:      userID,
			OrderID:     orderID,
			EntryType:   LedgerMatch,
			Amount:      credit,
			BalanceKind: BalanceAvailable,
			SettleDate:  settleDay,
			Remark:      remark,
		})
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
	count := 0
	dayCopy := settleDay
	for _, m := range matches {
		if m == nil || !m.Amount.IsPositive() {
			continue
		}
		full := money.Round(m.Amount.Mul(decimal.NewFromInt(2)))
		pool := money.Round(full.Mul(rate))
		if !pool.IsPositive() {
			continue
		}
		shares := ManageShares(pool)
		ancestors, err := FindActivatedRecommendAncestors(ctx, uc.users, m.UserID, manageAncestorWant)
		if err != nil {
			return count, err
		}
		uniq := manageUniqFromMatch(m)
		for i, ancestorID := range ancestors {
			if i >= 3 || !shares[i].IsPositive() {
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
	pkgs, err := uc.packages.ListAll(ctx)
	if err != nil {
		return nil, err
	}
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
		want := MatchCap(maxPaid[u.ID], pkgs)
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
		if err := ReleaseLockByCap(ctx, uc.users, uc.balances, uc.ledger, u.ID, cap, &dayCopy, remarkDailyLockUSDT, remarkDailyLockIspay); err != nil {
			return err
		}
	}
	return nil
}

// Status 返回今日是否已占位及最近一次记录。
func (uc *SettleUseCase) Status(ctx context.Context) (*SettleStatusView, error) {
	settleDay, dateStr := uc.settleDay(uc.now())
	view := &SettleStatusView{
		TodayDate:  dateStr,
		AllowForce: uc.allowForce,
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

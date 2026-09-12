package biz

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cigc/app/internal/conf"
	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

// MatchCap 按累计已支付向下匹配套餐档位，返回该档 daily_cap。
// 已下架套餐仍参与匹配；一档都够不上则为 0。
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

// SettleResult 一次日结摘要。管理奖仍为 0。
type SettleResult struct {
	Skipped     bool
	Forced      bool
	SettleDate  string
	UserCount   int
	CapUpdated  int
	DirectCount int
	MatchCount  int
	ManageCount int
}

// SettleStatusView 管理端今日/最近一次日结快照。
type SettleStatusView struct {
	TodayDate    string
	TodaySettled bool
	AllowForce   bool
	Today        *SettleRun
	Latest       *SettleRun
}

// SettleUseCase 日结：占位 → 直推 → 对碰 → 回写 cap_effective。
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
		tx:         tx,
		allowForce: allowForce,
		timezone:   tz,
		now:        time.Now,
	}
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

// Run 结算当前上海自然日。普通跑先占位；冲突则 already_settled。
// 顺序：直推入账（当日 paid）→ 对碰 → 回写封顶。force 时已发过的直推/对碰跳过。
func (uc *SettleUseCase) Run(ctx context.Context, force bool) (*SettleResult, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if force && !uc.allowForce {
		return nil, ErrForceSettleDisabled
	}

	settleDay, dateStr := uc.settleDay(uc.now())
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

	capRes, err := uc.refreshCaps(ctx)
	if err != nil {
		return nil, err
	}
	result.UserCount = capRes.UserCount
	result.CapUpdated = capRes.CapUpdated

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
		if err := uc.balances.AddAvailableBalance(ctx, inviterID, reward); err != nil {
			return err
		}
		if err := uc.ledger.Create(ctx, &LedgerEntry{
			UserID:      inviterID,
			OrderID:     &oid,
			EntryType:   LedgerDirect,
			Amount:      reward,
			BalanceKind: BalanceAvailable,
			SettleDate:  settleDay,
			Remark:      fmt.Sprintf("direct order=%d rate=%s", o.ID, rate.String()),
		}); err != nil {
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
		paid, err := uc.creditMatchForUser(ctx, b.UserID, rate, &dayCopy)
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
	parents, err := uc.placementParents(ctx)
	if err != nil {
		return err
	}
	for _, o := range orders {
		if o == nil || o.Status != OrderPaid || !o.Amount.IsPositive() {
			continue
		}
		adds := matchAncestorAdds(parents, o.UserID, money.Round(o.Amount))
		if len(adds) == 0 {
			_, err := uc.matches.TryApplyOrder(ctx, o.ID, settleDay)
			if err != nil {
				return err
			}
			continue
		}
		err := uc.tx.InTx(ctx, func(ctx context.Context) error {
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
		if err != nil {
			return err
		}
	}
	return nil
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

func (uc *SettleUseCase) creditMatchForUser(ctx context.Context, userID uint64, rate decimal.Decimal, settleDay *time.Time) (bool, error) {
	exists, err := uc.ledger.ExistsByUserTypeAndDate(ctx, userID, LedgerMatch, *settleDay)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	u, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}
	credited := false
	err = uc.tx.InTx(ctx, func(ctx context.Context) error {
		ok, err := uc.ledger.ExistsByUserTypeAndDate(ctx, userID, LedgerMatch, *settleDay)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
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
		if credit.IsPositive() {
			if err := uc.balances.AddAvailableBalance(ctx, userID, credit); err != nil {
				return err
			}
			credited = true
		}
		return uc.ledger.Create(ctx, &LedgerEntry{
			UserID:      userID,
			EntryType:   LedgerMatch,
			Amount:      credit,
			BalanceKind: BalanceAvailable,
			SettleDate:  settleDay,
			Remark:      fmt.Sprintf("match pair=%s rate=%s cap=%s", pair.String(), rate.String(), money.Round(u.CapEffective).String()),
		})
	})
	if err != nil {
		return false, err
	}
	return credited, nil
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
	result := &SettleResult{UserCount: len(users)}
	for _, u := range users {
		want := MatchCap(u.PaidAmount, pkgs)
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

// Status 返回今日是否已占位及最近一次记录。
func (uc *SettleUseCase) Status(ctx context.Context) (*SettleStatusView, error) {
	settleDay, dateStr := uc.settleDay(uc.now())
	view := &SettleStatusView{
		TodayDate:  dateStr,
		AllowForce: uc.allowForce,
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

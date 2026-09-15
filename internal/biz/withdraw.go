package biz

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

const (
	WithdrawPending   = "pending"
	WithdrawRewarded  = "rewarded"
	WithdrawDoing     = "doing"
	WithdrawPass      = "pass"
	WithdrawRejected  = "rejected"
	WithdrawCancelled = "cancelled"

	ConfigMinWithdraw         = "min_withdraw_amount"
	ConfigWithdrawFeeRate     = "withdraw_fee_rate"
	ConfigWithdrawDaily       = "withdraw_daily_limit"
	ConfigWithdrawDailyIspay  = "withdraw_daily_limit_ispay"
	ConfigDirectRate          = "direct_rate"
	ConfigMatchRate           = "match_rate"
	ConfigManageRate          = "manage_rate"
	ConfigIspayPrice          = "ispay_price"
	defaultMinWithdraw        = "10"
	defaultWithdrawFee        = "0.10"
	defaultWithdrawDaily      = "1000"
	defaultWithdrawDailyIspay = "1000"
	defaultDirectRate         = "0.10"
	defaultMatchRate          = "0.10"
	defaultManageRate         = "0.30"

	WithdrawAssetUSDT  = "usdt"
	WithdrawAssetIspay = "ispay"
)

// ParseWithdrawAsset 把兼容字段 coinType / type 收成 usdt 或 ispay。
func ParseWithdrawAsset(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "0", "1", "usdt":
		return WithdrawAssetUSDT, nil
	case "2", "3", "ispay", "newispay":
		return WithdrawAssetIspay, nil
	default:
		return "", ErrInvalidWithdrawAsset
	}
}

// ParseWithdrawListAsset 管理端筛选：空为全部，USDT / RAW_NEW 对齐 dapp-admin。
func ParseWithdrawListAsset(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "all":
		return "", nil
	case "usdt", "1":
		return WithdrawAssetUSDT, nil
	case "raw_new", "2", "3", "ispay", "newispay":
		return WithdrawAssetIspay, nil
	default:
		return "", ErrInvalidWithdrawAsset
	}
}

func withdrawAssetOf(w *Withdraw) string {
	if w == nil || strings.TrimSpace(w.Asset) == "" {
		return WithdrawAssetUSDT
	}
	return w.Asset
}

// Withdraw 是内部账户提现单。USDT 审过后由热钱包打款；ISPAY 本刀只审不解款。
type Withdraw struct {
	ID             uint64
	UserID         uint64
	Amount         decimal.Decimal
	FeeAmount      decimal.Decimal
	CreditedAmount decimal.Decimal
	Status         string
	Remark         string
	TxHash         string
	Asset          string
	PayoutError    string
	ReviewedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AdminWithdrawRow 管理端提现行。
type AdminWithdrawRow struct {
	Withdraw
	Address string
}

// WithdrawPage 提现分页。
type WithdrawPage struct {
	Items []*Withdraw
	Total int
}

// AdminWithdrawPage 管理端提现分页。
type AdminWithdrawPage struct {
	Items []*AdminWithdrawRow
	Total int
}

// WithdrawRepo 提现单读写。
type WithdrawRepo interface {
	Create(ctx context.Context, w *Withdraw) (*Withdraw, error)
	FindByID(ctx context.Context, id uint64) (*Withdraw, error)
	CasStatus(ctx context.Context, id uint64, from, to, remark string, reviewedAt *time.Time) error
	ListByUser(ctx context.Context, userID uint64) ([]*Withdraw, error)
	ListAdmin(ctx context.Context, address, status, asset string, page, pageSize int) ([]*AdminWithdrawRow, int, error)
	ListPayoutQueue(ctx context.Context, limit int) ([]*AdminWithdrawRow, error)
	UpdatePayoutMeta(ctx context.Context, id uint64, txHash, payoutError string) error
	SumUsedToday(ctx context.Context, userID uint64, asset string, from, to time.Time) (decimal.Decimal, error)
}

// BusinessConfig 是一条可管理的业务配置。
type BusinessConfig struct {
	ID        uint64
	Key       string
	Name      string
	Value     string
	SortOrder int
}

// ConfigRepo 读写 business_configs。
type ConfigRepo interface {
	GetValue(ctx context.Context, key string) (string, error)
	List(ctx context.Context) ([]*BusinessConfig, error)
	FindByID(ctx context.Context, id uint64) (*BusinessConfig, error)
	SetValue(ctx context.Context, id uint64, value string) error
	Upsert(ctx context.Context, row *BusinessConfig) error
}

// TxRunner 把一组写操作放进同一事务；测试可用空实现。
type TxRunner interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// NopTx 不包事务，直接执行（单元测试）。
type NopTx struct{}

// InTx 直接调用 fn。
func (NopTx) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// ChainPayer 热钱包打 BSC USDT。
type ChainPayer interface {
	TransferUSDT(ctx context.Context, to string, amount decimal.Decimal) (txHash string, err error)
	Receipt(ctx context.Context, txHash string) (ok, pending bool, err error)
}

// PayoutResult 一次批量打款摘要。
type PayoutResult struct {
	Enabled bool
	Scanned int
	Sent    int
	Passed  int
	Failed  int
	Skipped int
}

const payoutBatchLimit = 20

// WithdrawUseCase 提现申请、审核与 USDT 打款。
type WithdrawUseCase struct {
	users      UserRepo
	balances   UserBalanceRepo
	ledger     LedgerRepo
	withdraws  WithdrawRepo
	configs    ConfigRepo
	tx         TxRunner
	now        func() time.Time
	loc        *time.Location
	payer      ChainPayer
	payoutOn   bool
	payoutMax  decimal.Decimal
	payoutFrom string
	payoutMu   sync.Mutex
}

// NewWithdrawUseCase 构造提现用例。
func NewWithdrawUseCase(
	users UserRepo,
	balances UserBalanceRepo,
	ledger LedgerRepo,
	withdraws WithdrawRepo,
	configs ConfigRepo,
	tx TxRunner,
) *WithdrawUseCase {
	if tx == nil {
		tx = NopTx{}
	}
	return &WithdrawUseCase{
		users:     users,
		balances:  balances,
		ledger:    ledger,
		withdraws: withdraws,
		configs:   configs,
		tx:        tx,
		now:       time.Now,
		loc:       shanghaiLoc(),
	}
}

func shanghaiLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

// SetTimezone 日切时区，空则上海。
func (uc *WithdrawUseCase) SetTimezone(tz string) {
	if uc == nil {
		return
	}
	if strings.TrimSpace(tz) == "" {
		uc.loc = shanghaiLoc()
		return
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		uc.loc = shanghaiLoc()
		return
	}
	uc.loc = loc
}

func (uc *WithdrawUseCase) location() *time.Location {
	if uc == nil || uc.loc == nil {
		return shanghaiLoc()
	}
	return uc.loc
}

func (uc *WithdrawUseCase) shanghaiDayRange(now time.Time) (time.Time, time.Time) {
	loc := uc.location()
	t := now.In(loc)
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	return start, start.Add(24 * time.Hour)
}

func (uc *WithdrawUseCase) minAmount(ctx context.Context) decimal.Decimal {
	return uc.minAmountOf(ctx, WithdrawAssetUSDT)
}

func (uc *WithdrawUseCase) minAmountOf(ctx context.Context, asset string) decimal.Decimal {
	key := ConfigMinWithdraw
	fallback := defaultMinWithdraw
	if asset == WithdrawAssetIspay {
		key = ConfigMinWithdrawIspay
		fallback = defaultMinIspay
	}
	raw := fallback
	if uc.configs != nil {
		if v, err := uc.configs.GetValue(ctx, key); err == nil && strings.TrimSpace(v) != "" {
			raw = v
		}
	}
	d, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil || d.IsNegative() {
		d = decimal.RequireFromString(fallback)
	}
	if asset != WithdrawAssetIspay && !d.IsPositive() {
		d = decimal.RequireFromString(defaultMinWithdraw)
	}
	return money.Round(d)
}

func (uc *WithdrawUseCase) usdtFeeRate(ctx context.Context) decimal.Decimal {
	return uc.feeRateOf(ctx, WithdrawAssetUSDT)
}

func (uc *WithdrawUseCase) feeRateOf(ctx context.Context, asset string) decimal.Decimal {
	key := ConfigWithdrawFeeRate
	fallback := defaultWithdrawFee
	if asset == WithdrawAssetIspay {
		key = ConfigWithdrawFeeIspay
		fallback = defaultFeeIspay
	}
	raw := fallback
	if uc.configs != nil {
		if v, err := uc.configs.GetValue(ctx, key); err == nil && strings.TrimSpace(v) != "" {
			raw = v
		}
	}
	d, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil || d.IsNegative() || d.GreaterThan(decimal.NewFromInt(1)) {
		d = decimal.RequireFromString(fallback)
	}
	return money.Round(d)
}

func (uc *WithdrawUseCase) withdrawOpen(ctx context.Context) bool {
	return ConfigIntValue(ctx, uc.configs, ConfigWithdrawEnabled, defaultWithdrawOn, 0, 1) == 1
}

// Enabled 提现申请开关；缺省或非法视为开放。
func (uc *WithdrawUseCase) Enabled(ctx context.Context) bool {
	if uc == nil {
		return true
	}
	return uc.withdrawOpen(ctx)
}

func (uc *WithdrawUseCase) dailyLimitOf(ctx context.Context, asset string) decimal.Decimal {
	key := ConfigWithdrawDaily
	fallback := defaultWithdrawDaily
	if asset == WithdrawAssetIspay {
		key = ConfigWithdrawDailyIspay
		fallback = defaultWithdrawDailyIspay
	}
	raw := fallback
	if uc.configs != nil {
		if v, err := uc.configs.GetValue(ctx, key); err == nil && strings.TrimSpace(v) != "" {
			raw = v
		}
	}
	d, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil || d.IsNegative() {
		d = decimal.RequireFromString(fallback)
	}
	return money.Round(d)
}

func withdrawCountsDaily(status string) bool {
	switch status {
	case WithdrawPending, WithdrawRewarded, WithdrawDoing, WithdrawPass:
		return true
	default:
		return false
	}
}

func (uc *WithdrawUseCase) usedToday(ctx context.Context, userID uint64, asset string) (decimal.Decimal, error) {
	if uc == nil || uc.withdraws == nil || userID == 0 {
		return decimal.Zero, nil
	}
	from, to := uc.shanghaiDayRange(uc.now())
	used, err := uc.withdraws.SumUsedToday(ctx, userID, asset, from, to)
	if err != nil {
		return decimal.Zero, err
	}
	return money.Round(used), nil
}

func (uc *WithdrawUseCase) checkPerTxCap(ctx context.Context, asset string, amount decimal.Decimal) error {
	limit := uc.dailyLimitOf(ctx, asset)
	if !limit.IsPositive() {
		return nil
	}
	if money.Round(amount).GreaterThan(limit) {
		return ErrWithdrawDailyCap
	}
	return nil
}

func splitWithdrawFee(amount, rate decimal.Decimal) (fee, credited decimal.Decimal, err error) {
	fee = money.Round(amount.Mul(rate))
	credited = money.Round(amount.Sub(fee))
	if !credited.IsPositive() {
		return decimal.Zero, decimal.Zero, ErrWithdrawFeeExceeds
	}
	return fee, credited, nil
}

// WithdrawLimits 用户端最低额、费率与单笔上限。
type WithdrawLimits struct {
	Min       decimal.Decimal
	Rate      decimal.Decimal
	MinTwo    decimal.Decimal
	RateTwo   decimal.Decimal
	Daily     decimal.Decimal
	Today     decimal.Decimal
	Remain    decimal.Decimal
	DailyTwo  decimal.Decimal
	TodayTwo  decimal.Decimal
	RemainTwo decimal.Decimal
}

// UserLimits 用户端展示的最低额、费率与单笔上限；USDT / ISPAY 分开读配置。
func (uc *WithdrawUseCase) UserLimits(ctx context.Context, userID uint64) WithdrawLimits {
	out := WithdrawLimits{
		Min:     decimal.RequireFromString(defaultMinWithdraw),
		Rate:    decimal.RequireFromString(defaultWithdrawFee),
		MinTwo:  decimal.RequireFromString(defaultMinIspay),
		RateTwo: decimal.RequireFromString(defaultFeeIspay),
	}
	if uc == nil {
		return out
	}
	out.Min = uc.minAmountOf(ctx, WithdrawAssetUSDT)
	out.Rate = uc.feeRateOf(ctx, WithdrawAssetUSDT)
	out.MinTwo = uc.minAmountOf(ctx, WithdrawAssetIspay)
	out.RateTwo = uc.feeRateOf(ctx, WithdrawAssetIspay)
	out.Daily = uc.dailyLimitOf(ctx, WithdrawAssetUSDT)
	out.DailyTwo = uc.dailyLimitOf(ctx, WithdrawAssetIspay)
	if used, err := uc.usedToday(ctx, userID, WithdrawAssetUSDT); err == nil {
		out.Today = used
	}
	if used, err := uc.usedToday(ctx, userID, WithdrawAssetIspay); err == nil {
		out.TodayTwo = used
	}
	out.Remain = out.Daily
	out.RemainTwo = out.DailyTwo
	return out
}

// Create 从 available 扣到 frozen，写冻结流水；USDT 直接进打款队列。
func (uc *WithdrawUseCase) Create(ctx context.Context, userID uint64, amount decimal.Decimal) (*Withdraw, error) {
	return uc.CreateAsset(ctx, userID, amount, WithdrawAssetUSDT)
}

// CreateAsset 按币种提现。未激活一律拒绝。USDT：available→frozen，状态 rewarded 进打款队列；ISPAY：ispay→frozen_ispay，仍 pending 待审。
func (uc *WithdrawUseCase) CreateAsset(ctx context.Context, userID uint64, amount decimal.Decimal, asset string) (*Withdraw, error) {
	if asset != WithdrawAssetUSDT && asset != WithdrawAssetIspay {
		return nil, ErrInvalidWithdrawAsset
	}
	amount = money.Round(amount)
	if !amount.IsPositive() {
		return nil, ErrInvalidAmount
	}
	if !uc.withdrawOpen(ctx) {
		return nil, ErrWithdrawClosed
	}
	min := uc.minAmountOf(ctx, asset)
	if amount.LessThan(min) {
		return nil, ErrWithdrawBelowMin
	}
	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.IsDisabled() {
		return nil, ErrUserDisabled
	}
	if !user.IsActivated() {
		return nil, ErrUserInactive
	}
	fee, credited, errFee := splitWithdrawFee(amount, uc.feeRateOf(ctx, asset))
	if errFee != nil {
		return nil, errFee
	}
	if err := uc.checkPerTxCap(ctx, asset, amount); err != nil {
		return nil, err
	}
	if asset == WithdrawAssetIspay {
		if money.Round(user.IspayBalance).LessThan(amount) {
			return nil, ErrInsufficientBalance
		}
	} else if money.Round(user.AvailableBalance).LessThan(amount) {
		return nil, ErrInsufficientBalance
	}

	var created *Withdraw
	err = uc.tx.InTx(ctx, func(ctx context.Context) error {
		if asset == WithdrawAssetIspay {
			if err := uc.balances.SubIspayBalance(ctx, userID, amount); err != nil {
				return err
			}
			if err := uc.balances.AddFrozenIspay(ctx, userID, amount); err != nil {
				return err
			}
			neg := amount.Neg()
			if err := uc.ledger.Create(ctx, &LedgerEntry{
				UserID: userID, EntryType: LedgerFreeze, Amount: neg,
				BalanceKind: BalanceIspay, Remark: "withdraw freeze ispay",
			}); err != nil {
				return err
			}
			if err := uc.ledger.Create(ctx, &LedgerEntry{
				UserID: userID, EntryType: LedgerFreeze, Amount: amount,
				BalanceKind: BalanceFrozenIspay, Remark: "withdraw freeze ispay",
			}); err != nil {
				return err
			}
		} else {
			if err := uc.balances.SubAvailableBalance(ctx, userID, amount); err != nil {
				return err
			}
			if err := uc.balances.AddFrozenBalance(ctx, userID, amount); err != nil {
				return err
			}
			neg := amount.Neg()
			if err := uc.ledger.Create(ctx, &LedgerEntry{
				UserID: userID, EntryType: LedgerFreeze, Amount: neg,
				BalanceKind: BalanceAvailable, Remark: "withdraw freeze",
			}); err != nil {
				return err
			}
			if err := uc.ledger.Create(ctx, &LedgerEntry{
				UserID: userID, EntryType: LedgerFreeze, Amount: amount,
				BalanceKind: BalanceFrozen, Remark: "withdraw freeze",
			}); err != nil {
				return err
			}
		}
		w, err := uc.withdraws.Create(ctx, &Withdraw{
			UserID:         userID,
			Amount:         amount,
			FeeAmount:      fee,
			CreditedAmount: credited,
			Asset:          asset,
			Status:         withdrawCreateStatus(asset),
			CreatedAt:      uc.now(),
		})
		if err != nil {
			return err
		}
		created = w
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func withdrawCreateStatus(asset string) string {
	if asset == WithdrawAssetUSDT {
		return WithdrawRewarded
	}
	return WithdrawPending
}

// ListUser 用户提现列表后分页。
func (uc *WithdrawUseCase) ListUser(ctx context.Context, userID uint64, page int) (*WithdrawPage, error) {
	rows, err := uc.withdraws.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []*Withdraw{}
	}
	items := paginateWithdraws(rows, page, DefaultRewardPageSize)
	return &WithdrawPage{Items: items, Total: len(rows)}, nil
}

func paginateWithdraws(items []*Withdraw, page, pageSize int) []*Withdraw {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultRewardPageSize
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []*Withdraw{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

// ListAdmin 管理端分页，可筛 address、status、asset。
func (uc *WithdrawUseCase) ListAdmin(ctx context.Context, address, status, asset string, page int) (*AdminWithdrawPage, error) {
	if page < 1 {
		page = 1
	}
	rows, total, err := uc.withdraws.ListAdmin(ctx, address, status, asset, page, DefaultRewardPageSize)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []*AdminWithdrawRow{}
	}
	return &AdminWithdrawPage{Items: rows, Total: total}, nil
}

// Pass pending → rewarded，冻结保持，等待后续打款模块。
func (uc *WithdrawUseCase) Pass(ctx context.Context, id uint64) (*Withdraw, error) {
	if id == 0 {
		return nil, ErrInvalidAmount
	}
	var out *Withdraw
	err := uc.tx.InTx(ctx, func(ctx context.Context) error {
		w, err := uc.withdraws.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if w.Status != WithdrawPending {
			return ErrWithdrawConflict
		}
		now := uc.now()
		if err := uc.withdraws.CasStatus(ctx, id, WithdrawPending, WithdrawRewarded, w.Remark, &now); err != nil {
			return err
		}
		w.Status = WithdrawRewarded
		w.ReviewedAt = &now
		out = w
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (uc *WithdrawUseCase) unfreezePending(ctx context.Context, w *Withdraw, remark string) error {
	amount := money.Round(w.Amount)
	if withdrawAssetOf(w) == WithdrawAssetIspay {
		if err := uc.balances.SubFrozenIspay(ctx, w.UserID, amount); err != nil {
			return err
		}
		if err := uc.balances.AddIspayBalance(ctx, w.UserID, amount); err != nil {
			return err
		}
		neg := amount.Neg()
		if err := uc.ledger.Create(ctx, &LedgerEntry{
			UserID: w.UserID, EntryType: LedgerUnfreeze, Amount: neg,
			BalanceKind: BalanceFrozenIspay, Remark: remark + " ispay",
		}); err != nil {
			return err
		}
		return uc.ledger.Create(ctx, &LedgerEntry{
			UserID: w.UserID, EntryType: LedgerUnfreeze, Amount: amount,
			BalanceKind: BalanceIspay, Remark: remark + " ispay",
		})
	}
	if err := uc.balances.SubFrozenBalance(ctx, w.UserID, amount); err != nil {
		return err
	}
	if err := uc.balances.AddAvailableBalance(ctx, w.UserID, amount); err != nil {
		return err
	}
	neg := amount.Neg()
	if err := uc.ledger.Create(ctx, &LedgerEntry{
		UserID: w.UserID, EntryType: LedgerUnfreeze, Amount: neg,
		BalanceKind: BalanceFrozen, Remark: remark,
	}); err != nil {
		return err
	}
	return uc.ledger.Create(ctx, &LedgerEntry{
		UserID: w.UserID, EntryType: LedgerUnfreeze, Amount: amount,
		BalanceKind: BalanceAvailable, Remark: remark,
	})
}

// Reject pending/rewarded → rejected（尚未打款），frozen 退回 available。
func (uc *WithdrawUseCase) Reject(ctx context.Context, id uint64) (*Withdraw, error) {
	if id == 0 {
		return nil, ErrInvalidAmount
	}
	var out *Withdraw
	err := uc.tx.InTx(ctx, func(ctx context.Context) error {
		w, err := uc.withdraws.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if w.Status != WithdrawPending && w.Status != WithdrawRewarded {
			return ErrWithdrawConflict
		}
		if err := uc.unfreezePending(ctx, w, "withdraw reject unfreeze"); err != nil {
			return err
		}
		now := uc.now()
		if err := uc.withdraws.CasStatus(ctx, id, w.Status, WithdrawRejected, w.Remark, &now); err != nil {
			return err
		}
		w.Status = WithdrawRejected
		w.ReviewedAt = &now
		out = w
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Cancel 用户取消本人 pending/rewarded（尚未打款），解冻口径与拒绝相同。
func (uc *WithdrawUseCase) Cancel(ctx context.Context, userID, id uint64) (*Withdraw, error) {
	if userID == 0 || id == 0 {
		return nil, ErrInvalidAmount
	}
	var out *Withdraw
	err := uc.tx.InTx(ctx, func(ctx context.Context) error {
		w, err := uc.withdraws.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if w.UserID != userID {
			return ErrForbidden
		}
		if w.Status != WithdrawPending && w.Status != WithdrawRewarded {
			return ErrWithdrawConflict
		}
		if err := uc.unfreezePending(ctx, w, "withdraw cancel unfreeze"); err != nil {
			return err
		}
		now := uc.now()
		if err := uc.withdraws.CasStatus(ctx, id, w.Status, WithdrawCancelled, w.Remark, &now); err != nil {
			return err
		}
		w.Status = WithdrawCancelled
		w.ReviewedAt = &now
		out = w
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SetPayout 注入热钱包打款；payer 为空则只能扫到 disabled。
func (uc *WithdrawUseCase) SetPayout(payer ChainPayer, enabled bool, maxUSDT decimal.Decimal) {
	if uc == nil {
		return
	}
	uc.payer = payer
	uc.payoutOn = enabled && payer != nil && maxUSDT.IsPositive()
	uc.payoutMax = money.Round(maxUSDT)
	uc.payoutFrom = ""
	if p, ok := payer.(interface{ FromAddress() string }); ok {
		uc.payoutFrom = p.FromAddress()
	}
}

// HotWalletAddress 热钱包地址（无私钥）。
func (uc *WithdrawUseCase) HotWalletAddress() string {
	if uc == nil {
		return ""
	}
	return uc.payoutFrom
}

// PayoutMaxUSDT 单笔打款上限。
func (uc *WithdrawUseCase) PayoutMaxUSDT() decimal.Decimal {
	if uc == nil {
		return decimal.Zero
	}
	return uc.payoutMax
}

// PayoutEnabled 是否允许打款。
func (uc *WithdrawUseCase) PayoutEnabled() bool {
	return uc != nil && uc.payoutOn && uc.payer != nil
}

// RunPayout 扫 rewarded/doing 的 USDT 单并打款。id>0 只处理该单。
func (uc *WithdrawUseCase) RunPayout(ctx context.Context, id uint64) (*PayoutResult, error) {
	uc.payoutMu.Lock()
	defer uc.payoutMu.Unlock()
	res := &PayoutResult{Enabled: uc.PayoutEnabled()}
	if !res.Enabled {
		return res, ErrPayoutDisabled
	}
	var rows []*AdminWithdrawRow
	if id > 0 {
		w, err := uc.withdraws.FindByID(ctx, id)
		if err != nil {
			return res, err
		}
		rows = []*AdminWithdrawRow{{Withdraw: *w}}
	} else {
		list, err := uc.withdraws.ListPayoutQueue(ctx, payoutBatchLimit)
		if err != nil {
			return res, err
		}
		rows = list
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		res.Scanned++
		if err := uc.payoutOne(ctx, res, &row.Withdraw); err != nil {
			return res, err
		}
	}
	return res, nil
}

func (uc *WithdrawUseCase) payoutOne(ctx context.Context, res *PayoutResult, w *Withdraw) error {
	if withdrawAssetOf(w) != WithdrawAssetUSDT {
		res.Skipped++
		_ = uc.withdraws.UpdatePayoutMeta(ctx, w.ID, w.TxHash, "本刀只打 USDT")
		return nil
	}
	amt := money.Round(w.CreditedAmount)
	if !amt.IsPositive() {
		amt = money.Round(w.Amount)
	}
	if uc.payoutMax.IsPositive() && amt.GreaterThan(uc.payoutMax) {
		res.Skipped++
		_ = uc.withdraws.UpdatePayoutMeta(ctx, w.ID, w.TxHash, "exceeds payout max")
		return nil
	}
	switch w.Status {
	case WithdrawDoing:
		if strings.TrimSpace(w.TxHash) != "" {
			return uc.finishPayout(ctx, res, w)
		}
		now := uc.now()
		if err := uc.withdraws.CasStatus(ctx, w.ID, WithdrawDoing, WithdrawRewarded, w.Remark, &now); err != nil {
			return err
		}
		_ = uc.withdraws.UpdatePayoutMeta(ctx, w.ID, "", "reset empty doing")
		w.Status = WithdrawRewarded
		w.TxHash = ""
	case WithdrawRewarded:
	default:
		res.Skipped++
		return nil
	}
	user, err := uc.users.FindByID(ctx, w.UserID)
	if err != nil {
		res.Failed++
		_ = uc.withdraws.UpdatePayoutMeta(ctx, w.ID, w.TxHash, trimPayoutErr(err))
		return nil
	}
	to := strings.TrimSpace(user.Address)
	now := uc.now()
	if w.Status == WithdrawRewarded {
		if err := uc.withdraws.CasStatus(ctx, w.ID, WithdrawRewarded, WithdrawDoing, w.Remark, &now); err != nil {
			res.Failed++
			return nil
		}
		w.Status = WithdrawDoing
	}
	hash, err := uc.payer.TransferUSDT(ctx, to, amt)
	if err != nil {
		res.Failed++
		_ = uc.withdraws.CasStatus(ctx, w.ID, WithdrawDoing, WithdrawRewarded, w.Remark, &now)
		_ = uc.withdraws.UpdatePayoutMeta(ctx, w.ID, "", trimPayoutErr(err))
		return nil
	}
	res.Sent++
	w.TxHash = strings.ToLower(strings.TrimSpace(hash))
	_ = uc.withdraws.UpdatePayoutMeta(ctx, w.ID, w.TxHash, "")
	return uc.finishPayout(ctx, res, w)
}

func (uc *WithdrawUseCase) finishPayout(ctx context.Context, res *PayoutResult, w *Withdraw) error {
	ok, pending, err := uc.payer.Receipt(ctx, w.TxHash)
	if err != nil {
		res.Failed++
		_ = uc.withdraws.UpdatePayoutMeta(ctx, w.ID, w.TxHash, trimPayoutErr(err))
		return nil
	}
	if pending {
		_ = uc.withdraws.UpdatePayoutMeta(ctx, w.ID, w.TxHash, "waiting receipt")
		return nil
	}
	now := uc.now()
	if !ok {
		res.Failed++
		_ = uc.withdraws.CasStatus(ctx, w.ID, WithdrawDoing, WithdrawRewarded, w.Remark, &now)
		_ = uc.withdraws.UpdatePayoutMeta(ctx, w.ID, w.TxHash, "tx reverted")
		return nil
	}
	amount := money.Round(w.Amount)
	err = uc.tx.InTx(ctx, func(ctx context.Context) error {
		if err := uc.balances.SubFrozenBalance(ctx, w.UserID, amount); err != nil {
			return err
		}
		if err := uc.ledger.Create(ctx, &LedgerEntry{
			UserID: w.UserID, EntryType: LedgerWithdraw, Amount: amount.Neg(),
			BalanceKind: BalanceFrozen, Remark: "payout " + w.TxHash,
		}); err != nil {
			return err
		}
		return uc.withdraws.CasStatus(ctx, w.ID, WithdrawDoing, WithdrawPass, w.Remark, &now)
	})
	if err != nil {
		res.Failed++
		_ = uc.withdraws.UpdatePayoutMeta(ctx, w.ID, w.TxHash, trimPayoutErr(err))
		return nil
	}
	res.Passed++
	return nil
}

func trimPayoutErr(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	if len(s) > 240 {
		return s[:240]
	}
	return s
}

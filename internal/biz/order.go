package biz

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

const (
	OrderPending    = "pending"
	OrderConfirming = "confirming"
	OrderPaid       = "paid"
	OrderAbnormal   = "abnormal"
)

// Package 是可购买套餐档位，金额同时对应封顶匹配档。
type Package struct {
	ID          uint64
	Amount      decimal.Decimal
	Title       string
	GoodsDesc   string
	DailyCap    decimal.Decimal
	ReleaseDays int
	SortOrder   int
	Enabled     bool
}

// Order 是套餐购买单。仅 paid 计入 paid_amount。
type Order struct {
	ID            uint64
	OrderNo       string
	UserID        uint64
	PackageID     uint64
	Amount        decimal.Decimal
	TitleSnapshot string
	GoodsSnapshot string
	Status        string
	TxHash        *string
	LogIndex      int
	ReleaseDays   int
	PaidAt        *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// FormatOrderNo 认购订单编号，如 C000016。
func FormatOrderNo(id uint64) string {
	return fmt.Sprintf("C%06d", id)
}

// FormatOrderSource 编号与金额的组合展示。
func FormatOrderSource(orderNo, amount string) string {
	orderNo = strings.TrimSpace(orderNo)
	amount = strings.TrimSpace(amount)
	switch {
	case orderNo != "" && amount != "":
		return orderNo + " / " + amount
	case orderNo != "":
		return orderNo
	default:
		return amount
	}
}

// DisplayNo 返回订单编号；缺省时按 ID 生成。
func (o *Order) DisplayNo() string {
	if o == nil {
		return ""
	}
	if no := strings.TrimSpace(o.OrderNo); no != "" {
		return no
	}
	if o.ID == 0 {
		return ""
	}
	return FormatOrderNo(o.ID)
}

// PackageRepo 套餐查询。
type PackageRepo interface {
	ListEnabled(ctx context.Context) ([]*Package, error)
	ListAll(ctx context.Context) ([]*Package, error)
	FindByAmount(ctx context.Context, amount decimal.Decimal) (*Package, error)
	FindByID(ctx context.Context, id uint64) (*Package, error)
	Create(ctx context.Context, p *Package) (*Package, error)
	Update(ctx context.Context, p *Package) (*Package, error)
	Delete(ctx context.Context, id uint64) error
}

// OrderRepo 订单读写。
type OrderRepo interface {
	Create(ctx context.Context, o *Order) (*Order, error)
	FindByID(ctx context.Context, id uint64) (*Order, error)
	ListByUser(ctx context.Context, userID uint64) ([]*Order, error)
	MarkPaid(ctx context.Context, id uint64, userID uint64, amount decimal.Decimal, paidAt time.Time) error
	MarkPaidWithChain(ctx context.Context, id uint64, userID uint64, amount decimal.Decimal, paidAt time.Time, txHash string, logIndex int) error
	FindOldestPendingByUserAmount(ctx context.Context, userID uint64, amount decimal.Decimal) (*Order, error)
	FindByTxEvent(ctx context.Context, txHash string, logIndex int) (*Order, error)
	ListAdmin(ctx context.Context, address, status string, page, pageSize int) ([]*AdminOrderRow, int, error)
	// ListPaidBetween 返回 paid_at ∈ [from, to) 且 status=paid 的订单。
	ListPaidBetween(ctx context.Context, from, to time.Time) ([]*Order, error)
	// ListPaidBefore 返回 paid_at < to 且 status=paid 的订单（含释放档）。
	ListPaidBefore(ctx context.Context, to time.Time) ([]*Order, error)
	// MaxPaidAmountByUser 每用户已支付订单的最大单金额（不累加）。
	MaxPaidAmountByUser(ctx context.Context) (map[uint64]decimal.Decimal, error)
}

// AdminOrderRow 管理端订单行（含钱包地址）。
type AdminOrderRow struct {
	Order
	Address string
}

// AdminOrderPage 管理端订单分页。
type AdminOrderPage struct {
	Items []*AdminOrderRow
	Total int
}

// OrderUseCase 套餐展示、下单、管理端标记已支付。
type OrderUseCase struct {
	packages PackageRepo
	orders   OrderRepo
	users    UserRepo
	balances UserBalanceRepo
	ledger   LedgerRepo
	tx       TxRunner
	paidHook OrderPaidHook
}

// NewOrderUseCase 构造订单用例。
func NewOrderUseCase(packages PackageRepo, orders OrderRepo, users UserRepo, balances UserBalanceRepo, ledger LedgerRepo) *OrderUseCase {
	return &OrderUseCase{packages: packages, orders: orders, users: users, balances: balances, ledger: ledger, tx: NopTx{}}
}

// SetTx 生产注入 Data 事务；测试默认 NopTx。
func (uc *OrderUseCase) SetTx(tx TxRunner) {
	if uc != nil && tx != nil {
		uc.tx = tx
	}
}

// SetPaidHook 支付后秒结；生产由 SettleUseCase 注入。
func (uc *OrderUseCase) SetPaidHook(h OrderPaidHook) {
	if uc != nil {
		uc.paidHook = h
	}
}

// ListPackages 返回已上架套餐。
func (uc *OrderUseCase) ListPackages(ctx context.Context) ([]*Package, error) {
	return uc.packages.ListEnabled(ctx)
}

// ListAllPackages 管理端套餐一览（含下架）。
func (uc *OrderUseCase) ListAllPackages(ctx context.Context) ([]*Package, error) {
	return uc.packages.ListAll(ctx)
}

// UpdatePackage 管理端改套餐金额与默认释放天数。
func (uc *OrderUseCase) UpdatePackage(ctx context.Context, id uint64, amount decimal.Decimal, days int) (*Package, error) {
	amount = money.Round(amount)
	if !amount.IsPositive() {
		return nil, ErrInvalidAmount
	}
	if !ValidReleaseDays(days) {
		return nil, ErrInvalidReleaseDays
	}
	cur, err := uc.packages.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	other, err := uc.packages.FindByAmount(ctx, amount)
	if err != nil && !errors.Is(err, ErrPackageNotFound) {
		return nil, err
	}
	if err == nil && other != nil && other.ID != id {
		return nil, ErrPackageAmountTaken
	}
	cur.Amount = amount
	cur.ReleaseDays = days
	return uc.packages.Update(ctx, cur)
}

func normalizePackage(p *Package) error {
	if p == nil {
		return ErrPackageNotFound
	}
	p.Title = strings.TrimSpace(p.Title)
	p.GoodsDesc = strings.TrimSpace(p.GoodsDesc)
	p.Amount = money.Round(p.Amount)
	p.DailyCap = money.Round(p.DailyCap)
	if p.Title == "" {
		return ErrPackageTitle
	}
	if p.GoodsDesc == "" {
		p.GoodsDesc = p.Title
	}
	if !p.Amount.IsPositive() {
		return ErrInvalidAmount
	}
	if p.DailyCap.IsNegative() {
		return ErrInvalidAmount
	}
	if !ValidReleaseDays(p.ReleaseDays) {
		return ErrInvalidReleaseDays
	}
	return nil
}

func (uc *OrderUseCase) ensureAmountFree(ctx context.Context, id uint64, amount decimal.Decimal) error {
	other, err := uc.packages.FindByAmount(ctx, amount)
	if err != nil && !errors.Is(err, ErrPackageNotFound) {
		return err
	}
	if err == nil && other != nil && other.ID != id {
		return ErrPackageAmountTaken
	}
	return nil
}

// CreatePackage 管理端新增 Web3 商城商品。
func (uc *OrderUseCase) CreatePackage(ctx context.Context, p *Package) (*Package, error) {
	if err := normalizePackage(p); err != nil {
		return nil, err
	}
	if err := uc.ensureAmountFree(ctx, 0, p.Amount); err != nil {
		return nil, err
	}
	p.ID = 0
	return uc.packages.Create(ctx, p)
}

// GetPackage 管理端读一档。
func (uc *OrderUseCase) GetPackage(ctx context.Context, id uint64) (*Package, error) {
	return uc.packages.FindByID(ctx, id)
}

// SavePackage 管理端改名称、描述、金额、封顶、天数、排序、上下架。
func (uc *OrderUseCase) SavePackage(ctx context.Context, p *Package) (*Package, error) {
	if p == nil || p.ID == 0 {
		return nil, ErrPackageNotFound
	}
	if _, err := uc.packages.FindByID(ctx, p.ID); err != nil {
		return nil, err
	}
	if err := normalizePackage(p); err != nil {
		return nil, err
	}
	if err := uc.ensureAmountFree(ctx, p.ID, p.Amount); err != nil {
		return nil, err
	}
	return uc.packages.Update(ctx, p)
}

// DeletePackage 删除商品；已有订单则拒绝，请先下架。
func (uc *OrderUseCase) DeletePackage(ctx context.Context, id uint64) error {
	if id == 0 {
		return ErrPackageNotFound
	}
	if _, err := uc.packages.FindByID(ctx, id); err != nil {
		return err
	}
	return uc.packages.Delete(ctx, id)
}

// CreateOrder 按套餐金额开 pending 单；金额必须精确匹配一档。days 为 300/600/750。
func (uc *OrderUseCase) CreateOrder(ctx context.Context, userID uint64, amount decimal.Decimal, days int) (*Order, error) {
	amount = money.Round(amount)
	if !amount.IsPositive() {
		return nil, ErrInvalidAmount
	}
	if !ValidReleaseDays(days) {
		return nil, ErrInvalidReleaseDays
	}
	if _, err := uc.users.FindByID(ctx, userID); err != nil {
		return nil, err
	}
	pkg, err := uc.packages.FindByAmount(ctx, amount)
	if err != nil {
		return nil, err
	}
	if !pkg.Enabled {
		return nil, ErrPackageDisabled
	}
	return uc.orders.Create(ctx, &Order{
		UserID:        userID,
		PackageID:     pkg.ID,
		Amount:        pkg.Amount,
		TitleSnapshot: pkg.Title,
		GoodsSnapshot: pkg.GoodsDesc,
		Status:        OrderPending,
		ReleaseDays:   days,
	})
}

// BuyWithRecharge 用充值页可用余额买套餐，扣款成功后立即标 paid。
func (uc *OrderUseCase) BuyWithRecharge(ctx context.Context, userID uint64, amount decimal.Decimal, days int) (*Order, error) {
	amount = money.Round(amount)
	if !amount.IsPositive() {
		return nil, ErrInvalidAmount
	}
	if !ValidReleaseDays(days) {
		return nil, ErrInvalidReleaseDays
	}
	if _, err := uc.users.FindByID(ctx, userID); err != nil {
		return nil, err
	}
	pkg, err := uc.packages.FindByAmount(ctx, amount)
	if err != nil {
		return nil, err
	}
	if !pkg.Enabled {
		return nil, ErrPackageDisabled
	}
	if uc.tx == nil {
		uc.tx = NopTx{}
	}
	var out *Order
	err = uc.tx.InTx(ctx, func(ctx context.Context) error {
		if err := uc.balances.SubRechargeBalance(ctx, userID, pkg.Amount); err != nil {
			return err
		}
		o, err := uc.orders.Create(ctx, &Order{
			UserID:        userID,
			PackageID:     pkg.ID,
			Amount:        pkg.Amount,
			TitleSnapshot: pkg.Title,
			GoodsSnapshot: pkg.GoodsDesc,
			Status:        OrderPending,
			ReleaseDays:   days,
		})
		if err != nil {
			return err
		}
		if uc.ledger != nil {
			oid := o.ID
			if err := uc.ledger.Create(ctx, &LedgerEntry{
				UserID:      userID,
				OrderID:     &oid,
				EntryType:   LedgerRechargeBuy,
				Amount:      pkg.Amount.Neg(),
				BalanceKind: BalanceRecharge,
				Remark:      fmt.Sprintf("buy order=%d", o.ID),
			}); err != nil {
				return err
			}
		}
		now := time.Now()
		if err := uc.orders.MarkPaid(ctx, o.ID, o.UserID, o.Amount, now); err != nil {
			return err
		}
		o.Status = OrderPaid
		o.PaidAt = &now
		if uc.paidHook != nil {
			if err := uc.paidHook.OnOrderPaid(ctx, o); err != nil {
				return err
			}
		} else {
			if err := ApplyPaidOrderCap(ctx, uc.users, uc.packages, o.UserID, o.Amount); err != nil {
				return err
			}
			if err := ReleaseInactiveLock(ctx, uc.users, uc.balances, uc.ledger, o.UserID, orderActivateCap(ctx, uc.packages, o.Amount)); err != nil {
				return err
			}
		}
		out = o
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListOrders 用户订单，新单在前。
func (uc *OrderUseCase) ListOrders(ctx context.Context, userID uint64) ([]*Order, error) {
	return uc.orders.ListByUser(ctx, userID)
}

// MarkPaid 将 pending 单标为已支付，并累加会员 paid_amount。封顶立即按本单抬升。
func (uc *OrderUseCase) MarkPaid(ctx context.Context, orderID uint64) (*Order, error) {
	o, err := uc.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if o.Status != OrderPending {
		return nil, ErrOrderConflict
	}
	now := time.Now()
	if err := uc.orders.MarkPaid(ctx, orderID, o.UserID, o.Amount, now); err != nil {
		return nil, err
	}
	o.Status = OrderPaid
	o.PaidAt = &now
	if uc.paidHook != nil {
		if err := uc.paidHook.OnOrderPaid(ctx, o); err != nil {
			return nil, err
		}
	} else {
		if err := ApplyPaidOrderCap(ctx, uc.users, uc.packages, o.UserID, o.Amount); err != nil {
			return nil, err
		}
		if err := ReleaseInactiveLock(ctx, uc.users, uc.balances, uc.ledger, o.UserID, orderActivateCap(ctx, uc.packages, o.Amount)); err != nil {
			return nil, err
		}
	}
	return o, nil
}

// ListAdminOrders 管理端买单一览；默认全部状态，可按 address/status 筛。
func (uc *OrderUseCase) ListAdminOrders(ctx context.Context, address, status string, page int) (*AdminOrderPage, error) {
	if page < 1 {
		page = 1
	}
	rows, total, err := uc.orders.ListAdmin(ctx, address, status, page, DefaultRewardPageSize)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []*AdminOrderRow{}
	}
	return &AdminOrderPage{Items: rows, Total: total}, nil
}

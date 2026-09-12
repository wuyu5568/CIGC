package biz

import (
	"context"
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
	ID        uint64
	Amount    decimal.Decimal
	Title     string
	GoodsDesc string
	DailyCap  decimal.Decimal
	SortOrder int
	Enabled   bool
}

// Order 是套餐购买单。仅 paid 计入 paid_amount。
type Order struct {
	ID            uint64
	UserID        uint64
	PackageID     uint64
	Amount        decimal.Decimal
	TitleSnapshot string
	GoodsSnapshot string
	Status        string
	TxHash        *string
	LogIndex      int
	PaidAt        *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// PackageRepo 套餐查询。
type PackageRepo interface {
	ListEnabled(ctx context.Context) ([]*Package, error)
	ListAll(ctx context.Context) ([]*Package, error)
	FindByAmount(ctx context.Context, amount decimal.Decimal) (*Package, error)
	FindByID(ctx context.Context, id uint64) (*Package, error)
}

// OrderRepo 订单读写。
type OrderRepo interface {
	Create(ctx context.Context, o *Order) (*Order, error)
	FindByID(ctx context.Context, id uint64) (*Order, error)
	ListByUser(ctx context.Context, userID uint64) ([]*Order, error)
	MarkPaid(ctx context.Context, id uint64, userID uint64, amount decimal.Decimal, paidAt time.Time) error
	ListAdmin(ctx context.Context, address, status string, page, pageSize int) ([]*AdminOrderRow, int, error)
	// ListPaidBetween 返回 paid_at ∈ [from, to) 且 status=paid 的订单。
	ListPaidBetween(ctx context.Context, from, to time.Time) ([]*Order, error)
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
}

// NewOrderUseCase 构造订单用例。
func NewOrderUseCase(packages PackageRepo, orders OrderRepo, users UserRepo) *OrderUseCase {
	return &OrderUseCase{packages: packages, orders: orders, users: users}
}

// ListPackages 返回已上架套餐。
func (uc *OrderUseCase) ListPackages(ctx context.Context) ([]*Package, error) {
	return uc.packages.ListEnabled(ctx)
}

// CreateOrder 按套餐金额开 pending 单；金额必须精确匹配一档。
func (uc *OrderUseCase) CreateOrder(ctx context.Context, userID uint64, amount decimal.Decimal) (*Order, error) {
	amount = money.Round(amount)
	if !amount.IsPositive() {
		return nil, ErrInvalidAmount
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
	})
}

// ListOrders 用户订单，新单在前。
func (uc *OrderUseCase) ListOrders(ctx context.Context, userID uint64) ([]*Order, error) {
	return uc.orders.ListByUser(ctx, userID)
}

// MarkPaid 将 pending 单标为已支付，并累加会员 paid_amount。不改 cap_effective（留给日结）。
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

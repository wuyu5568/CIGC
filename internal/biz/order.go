package biz

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/cigc/app/internal/pkg/sanitize"
	"github.com/shopspring/decimal"
)

const (
	OrderPending    = "pending"
	OrderConfirming = "confirming"
	OrderPaid       = "paid"
	OrderAbnormal   = "abnormal"
)

// Package 是可购买商品；日封顶按订单金额套档，不读本表 daily_cap。
type Package struct {
	ID          uint64
	Amount      decimal.Decimal
	Title       string
	GoodsDesc   string
	DailyCap    decimal.Decimal
	ReleaseDays int
	SortOrder   int
	Enabled     bool
	Image       string
	Detail      string
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

// FormatOrderNo 旧单缺省编号：C + 6 位 ID。新单用 RandomOrderNo。
func FormatOrderNo(id uint64) string {
	return fmt.Sprintf("C%06d", id)
}

// RandomOrderNo 新单编号：C + 随机六位数字。
func RandomOrderNo() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil || n == nil {
		return FormatOrderNo(uint64(time.Now().UnixNano() % 1000000))
	}
	return fmt.Sprintf("C%06d", n.Int64())
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
	FindByAmount(ctx context.Context, amount decimal.Decimal, days int) (*Package, error)
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

// PackageReleaseDays 套餐默认释放档；非法或缺省按 300。
func PackageReleaseDays(p *Package) int {
	if p == nil || !ValidReleaseDays(p.ReleaseDays) {
		return ReleaseDays300
	}
	return p.ReleaseDays
}

// FilterPackagesByDays 按释放天数精确过滤；days=0 表示不过滤。
func FilterPackagesByDays(pkgs []*Package, days int) []*Package {
	if days == 0 {
		return pkgs
	}
	out := make([]*Package, 0, len(pkgs))
	for _, p := range pkgs {
		if PackageReleaseDays(p) == days {
			out = append(out, p)
		}
	}
	return out
}

// FilterEnabledPackages 只保留上架商品。
func FilterEnabledPackages(pkgs []*Package) []*Package {
	out := make([]*Package, 0, len(pkgs))
	for _, p := range pkgs {
		if p != nil && p.Enabled {
			out = append(out, p)
		}
	}
	return out
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
	other, err := uc.packages.FindByAmount(ctx, amount, days)
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

func (uc *OrderUseCase) ensureAmountFree(ctx context.Context, id uint64, amount decimal.Decimal, days int) error {
	other, err := uc.packages.FindByAmount(ctx, amount, days)
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
	if err := uc.ensureAmountFree(ctx, 0, p.Amount, p.ReleaseDays); err != nil {
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
	cur, err := uc.packages.FindByID(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if err := normalizePackage(p); err != nil {
		return nil, err
	}
	if err := uc.ensureAmountFree(ctx, p.ID, p.Amount, p.ReleaseDays); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Image) == "" {
		p.Image = cur.Image
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

const (
	DefaultWeb3GoodsPageSize = 10
	MaxWeb3GoodsPageSize     = 100
	MaxWeb3GoodsDetailBytes  = 200 * 1024
)

// Web3GoodsInput 管理端商品写入：名称、描述、主图、详情、单价、上下架。
type Web3GoodsInput struct {
	ID          uint64
	Name        string
	Desc        string
	Amount      decimal.Decimal
	DailyCap    decimal.Decimal
	Days        int
	Sort        int
	OnSale      bool
	Image       string
	Detail      string
	HasDailyCap bool
	HasSort     bool
	HasOnSale   bool
	HasImage    bool
	HasDetail   bool
}

func paginateWeb3Goods(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultWeb3GoodsPageSize
	}
	if pageSize > MaxWeb3GoodsPageSize {
		pageSize = MaxWeb3GoodsPageSize
	}
	return page, pageSize
}

func normalizeWeb3Goods(in *Web3GoodsInput, create bool) error {
	if in == nil {
		return ErrPackageNotFound
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Desc = strings.TrimSpace(in.Desc)
	in.Image = strings.TrimSpace(in.Image)
	in.Amount = money.Round(in.Amount)
	in.DailyCap = money.Round(in.DailyCap)
	if !in.Amount.IsPositive() {
		return ErrInvalidAmount
	}
	if !ValidReleaseDays(in.Days) {
		in.Days = ReleaseDays300
	}
	if create {
		in.DailyCap = decimal.Zero
		if !in.HasSort {
			in.Sort = 0
		}
		if !in.HasOnSale {
			in.OnSale = true
		}
	}
	if in.HasImage {
		in.Image = strings.TrimSpace(in.Image)
	}
	if create || in.HasDetail {
		if len(in.Detail) > MaxWeb3GoodsDetailBytes {
			return ErrPackageDetail
		}
		in.Detail = sanitize.HTML(in.Detail)
		in.HasDetail = true
	}
	return nil
}

func (uc *OrderUseCase) requireWeb3Goods(ctx context.Context, id uint64) (*Package, error) {
	if id == 0 {
		return nil, ErrPackageNotFound
	}
	return uc.packages.FindByID(ctx, id)
}

// ListWeb3Goods 分页列出商品。days=0 表示不过滤；onSaleOnly 为 true 时只返回上架（用户端）。
func (uc *OrderUseCase) ListWeb3Goods(ctx context.Context, days, page, pageSize int, onSaleOnly bool) ([]*Package, int, error) {
	if days != 0 && !ValidReleaseDays(days) {
		return nil, 0, ErrInvalidReleaseDays
	}
	page, pageSize = paginateWeb3Goods(page, pageSize)
	pkgs, err := uc.packages.ListAll(ctx)
	if err != nil {
		return nil, 0, err
	}
	all := FilterPackagesByDays(pkgs, days)
	if onSaleOnly {
		all = FilterEnabledPackages(all)
	}
	total := len(all)
	start := (page - 1) * pageSize
	if start >= total {
		return []*Package{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

// GetWeb3Goods 按 id 取商品。onSaleOnly 时下架视为不存在。
func (uc *OrderUseCase) GetWeb3Goods(ctx context.Context, id uint64, onSaleOnly bool) (*Package, error) {
	p, err := uc.requireWeb3Goods(ctx, id)
	if err != nil {
		return nil, err
	}
	if onSaleOnly && (p == nil || !p.Enabled) {
		return nil, ErrPackageNotFound
	}
	return p, nil
}

func (in *Web3GoodsInput) toPackage(id uint64, image, detail string) *Package {
	return &Package{
		ID:          id,
		Amount:      in.Amount,
		Title:       in.Name,
		GoodsDesc:   in.Desc,
		DailyCap:    in.DailyCap,
		ReleaseDays: in.Days,
		SortOrder:   in.Sort,
		Enabled:     in.OnSale,
		Image:       image,
		Detail:      detail,
	}
}

// CreateWeb3Goods 新增商品。天数仅作库默认值，购买时由用户另选。
func (uc *OrderUseCase) CreateWeb3Goods(ctx context.Context, in *Web3GoodsInput) (*Package, error) {
	if err := normalizeWeb3Goods(in, true); err != nil {
		return nil, err
	}
	return uc.packages.Create(ctx, in.toPackage(0, in.Image, in.Detail))
}

// UpdateWeb3Goods 编辑商品。不传图片/详情则保留原文；传空字符串则清空。
func (uc *OrderUseCase) UpdateWeb3Goods(ctx context.Context, in *Web3GoodsInput) (*Package, error) {
	if in == nil {
		return nil, ErrPackageNotFound
	}
	cur, err := uc.requireWeb3Goods(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if err := normalizeWeb3Goods(in, false); err != nil {
		return nil, err
	}
	in.DailyCap = cur.DailyCap
	in.Sort = cur.SortOrder
	in.Days = PackageReleaseDays(cur)
	if !in.HasOnSale {
		in.OnSale = cur.Enabled
	}
	image := cur.Image
	if in.HasImage {
		image = in.Image
	}
	detail := cur.Detail
	if in.HasDetail {
		detail = in.Detail
	}
	return uc.packages.Update(ctx, in.toPackage(cur.ID, image, detail))
}

// SetWeb3GoodsOnSale 上架/下架。
func (uc *OrderUseCase) SetWeb3GoodsOnSale(ctx context.Context, id uint64, onSale bool) (*Package, error) {
	cur, err := uc.requireWeb3Goods(ctx, id)
	if err != nil {
		return nil, err
	}
	cur.Enabled = onSale
	return uc.packages.Update(ctx, cur)
}

// DeleteWeb3Goods 删除商品；已有订单则拒绝。
func (uc *OrderUseCase) DeleteWeb3Goods(ctx context.Context, id uint64) error {
	if _, err := uc.requireWeb3Goods(ctx, id); err != nil {
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
	pkg, err := uc.packages.FindByAmount(ctx, amount, days)
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

// BuyWithRecharge 用充值余额按金额+天数匹配商品（旧客户端）。
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
	pkg, err := uc.packages.FindByAmount(ctx, amount, days)
	if err != nil {
		return nil, err
	}
	return uc.buyPackageWithRecharge(ctx, userID, pkg, days)
}

// BuyWithRechargeGoods 用充值余额按商品 id 购买，金额以服务端为准，天数在结算时选择。
func (uc *OrderUseCase) BuyWithRechargeGoods(ctx context.Context, userID, goodsID uint64, days int) (*Order, error) {
	if !ValidReleaseDays(days) {
		return nil, ErrInvalidReleaseDays
	}
	if _, err := uc.users.FindByID(ctx, userID); err != nil {
		return nil, err
	}
	pkg, err := uc.packages.FindByID(ctx, goodsID)
	if err != nil {
		return nil, err
	}
	return uc.buyPackageWithRecharge(ctx, userID, pkg, days)
}

func (uc *OrderUseCase) buyPackageWithRecharge(ctx context.Context, userID uint64, pkg *Package, days int) (*Order, error) {
	if pkg == nil || !pkg.Enabled {
		return nil, ErrPackageDisabled
	}
	if uc.tx == nil {
		uc.tx = NopTx{}
	}
	var out *Order
	err := uc.tx.InTx(ctx, func(ctx context.Context) error {
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

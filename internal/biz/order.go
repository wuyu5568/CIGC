package biz

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sort"
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
	Contents    map[string]PackageContent
	SKUs        []PackageSKU
}

type PackageContent struct {
	Title     string
	GoodsDesc string
	Image     string
	Detail    string
}

// PackageSKU 商品规格；有规格时下单按规格单价。
type PackageSKU struct {
	ID        uint64
	PackageID uint64
	Name      string
	NameEn    string
	Amount    decimal.Decimal
	Image     string
	SortOrder int
	Enabled   bool
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

// CartItem 购物车一行：商品 id、可选规格 id 与数量。金额一律按服务端单价 × qty。
type CartItem struct {
	GoodsID uint64
	SkuID   uint64
	Qty     int
}

const (
	maxCartDistinct    = 50
	maxCartQtyPerGoods = 999
	titleSnapshotMax   = 128
	goodsSnapshotMax   = 512
)

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
	UpdateSortOrder(ctx context.Context, id uint64, sort int) error
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

// SortPackagesByAmount 按单价从低到高，同价再按 id。
func SortPackagesByAmount(pkgs []*Package) {
	sort.SliceStable(pkgs, func(i, j int) bool {
		a, b := pkgs[i], pkgs[j]
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		if !a.Amount.Equal(b.Amount) {
			return a.Amount.LessThan(b.Amount)
		}
		return a.ID < b.ID
	})
}

// SortPackagesBySortOrder 按后台拖拽顺序；未排序时退回单价、id。
func SortPackagesBySortOrder(pkgs []*Package) {
	sort.SliceStable(pkgs, func(i, j int) bool {
		a, b := pkgs[i], pkgs[j]
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		if a.SortOrder != b.SortOrder {
			return a.SortOrder < b.SortOrder
		}
		if !a.Amount.Equal(b.Amount) {
			return a.Amount.LessThan(b.Amount)
		}
		return a.ID < b.ID
	})
}

// ListPackages 返回已上架套餐，按单价从低到高。
func (uc *OrderUseCase) ListPackages(ctx context.Context) ([]*Package, error) {
	pkgs, err := uc.packages.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	SortPackagesByAmount(pkgs)
	return pkgs, nil
}

// ListAllPackages 管理端套餐一览（含下架），按单价从低到高。
func (uc *OrderUseCase) ListAllPackages(ctx context.Context) ([]*Package, error) {
	pkgs, err := uc.packages.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	SortPackagesByAmount(pkgs)
	return pkgs, nil
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
	maxWeb3SKUs              = 20
	skuNameMaxRunes          = 64
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
	HasSKUs     bool
	Contents    map[string]PackageContent
	SKUs        []PackageSKU
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
	for locale, content := range in.Contents {
		if locale != "zh" && locale != "en" {
			delete(in.Contents, locale)
			continue
		}
		content.Title = strings.TrimSpace(content.Title)
		content.GoodsDesc = strings.TrimSpace(content.GoodsDesc)
		content.Image = strings.TrimSpace(content.Image)
		if len(content.Detail) > MaxWeb3GoodsDetailBytes {
			return ErrPackageDetail
		}
		content.Detail = sanitize.HTML(content.Detail)
		in.Contents[locale] = content
	}
	if err := normalizeWeb3SKUs(in); err != nil {
		return err
	}
	return nil
}

func normalizeWeb3SKUs(in *Web3GoodsInput) error {
	if in == nil || !in.HasSKUs {
		return nil
	}
	if len(in.SKUs) > maxWeb3SKUs {
		return ErrSKUInvalid
	}
	out := make([]PackageSKU, 0, len(in.SKUs))
	for i, sku := range in.SKUs {
		name := strings.TrimSpace(sku.Name)
		nameEn := strings.TrimSpace(sku.NameEn)
		if name == "" && nameEn == "" {
			return ErrSKUInvalid
		}
		if name == "" {
			name = nameEn
		}
		if len([]rune(name)) > skuNameMaxRunes || len([]rune(nameEn)) > skuNameMaxRunes {
			return ErrSKUInvalid
		}
		amt := money.Round(sku.Amount)
		if !amt.IsPositive() {
			amt = in.Amount
		}
		if !amt.IsPositive() {
			return ErrInvalidAmount
		}
		image := strings.TrimSpace(sku.Image)
		if len(image) > 512 {
			return ErrSKUInvalid
		}
		out = append(out, PackageSKU{
			ID:        sku.ID,
			Name:      name,
			NameEn:    nameEn,
			Amount:    amt,
			Image:     image,
			SortOrder: i + 1,
			Enabled:   sku.Enabled,
		})
	}
	in.SKUs = out
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
	SortPackagesBySortOrder(all)
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
	contents := make(map[string]PackageContent, len(in.Contents)+1)
	for locale, content := range in.Contents {
		contents[locale] = content
	}
	if _, ok := contents["zh"]; !ok {
		contents["zh"] = PackageContent{
			Title: in.Name, GoodsDesc: in.Desc, Image: image, Detail: detail,
		}
	}
	zh := contents["zh"]
	return &Package{
		ID:          id,
		Amount:      in.Amount,
		Title:       zh.Title,
		GoodsDesc:   zh.GoodsDesc,
		DailyCap:    in.DailyCap,
		ReleaseDays: in.Days,
		SortOrder:   in.Sort,
		Enabled:     in.OnSale,
		Image:       zh.Image,
		Detail:      zh.Detail,
		Contents:    contents,
		SKUs:        append([]PackageSKU(nil), in.SKUs...),
	}
}

// CreateWeb3Goods 新增商品。天数仅作库默认值，购买时由用户另选。
func (uc *OrderUseCase) CreateWeb3Goods(ctx context.Context, in *Web3GoodsInput) (*Package, error) {
	if err := normalizeWeb3Goods(in, true); err != nil {
		return nil, err
	}
	if !in.HasSort {
		next, err := uc.nextWeb3GoodsSort(ctx)
		if err != nil {
			return nil, err
		}
		in.Sort = next
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
	if !in.HasSort {
		in.Sort = cur.SortOrder
	}
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
	next := in.toPackage(cur.ID, image, detail)
	if next.Contents == nil {
		next.Contents = map[string]PackageContent{}
	}
	for locale, content := range cur.Contents {
		if _, supplied := in.Contents[locale]; !supplied {
			next.Contents[locale] = content
		}
	}
	if !in.HasSKUs {
		next.SKUs = append([]PackageSKU(nil), cur.SKUs...)
	}
	return uc.packages.Update(ctx, next)
}

func (uc *OrderUseCase) nextWeb3GoodsSort(ctx context.Context) (int, error) {
	pkgs, err := uc.packages.ListAll(ctx)
	if err != nil {
		return 0, err
	}
	max := 0
	for _, p := range pkgs {
		if p != nil && p.SortOrder > max {
			max = p.SortOrder
		}
	}
	return max + 1, nil
}

// SortWeb3Goods 按 ids 调整商品顺序；只传部分 id 时，在原列表对应位置内重排。
func (uc *OrderUseCase) SortWeb3Goods(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return ErrInvalidAmount
	}
	seen := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			return ErrPackageNotFound
		}
		if _, ok := seen[id]; ok {
			return ErrInvalidAmount
		}
		seen[id] = struct{}{}
	}
	pkgs, err := uc.packages.ListAll(ctx)
	if err != nil {
		return err
	}
	SortPackagesBySortOrder(pkgs)
	byID := make(map[uint64]*Package, len(pkgs))
	for _, p := range pkgs {
		if p == nil || p.ID == 0 {
			continue
		}
		byID[p.ID] = p
	}
	pos := make([]int, 0, len(ids))
	for i, p := range pkgs {
		if p == nil {
			continue
		}
		if _, ok := seen[p.ID]; ok {
			pos = append(pos, i)
		}
	}
	if len(pos) != len(ids) {
		return ErrPackageNotFound
	}
	for i, id := range ids {
		p, ok := byID[id]
		if !ok {
			return ErrPackageNotFound
		}
		pkgs[pos[i]] = p
	}
	if uc.tx == nil {
		uc.tx = NopTx{}
	}
	return uc.tx.InTx(ctx, func(ctx context.Context) error {
		for i, p := range pkgs {
			if p == nil || p.ID == 0 {
				continue
			}
			sortVal := i + 1
			if p.SortOrder == sortVal {
				continue
			}
			if err := uc.packages.UpdateSortOrder(ctx, p.ID, sortVal); err != nil {
				return err
			}
			p.SortOrder = sortVal
		}
		return nil
	})
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
	if len(enabledSKUs(pkg)) > 0 {
		return nil, ErrSKUInvalid
	}
	return uc.buyPackageWithRecharge(ctx, userID, pkg, days)
}

// BuyCartWithRecharge 购物车一次下单：按商品单价汇总，扣合计充值余额，落 1 笔 paid 订单。
func (uc *OrderUseCase) BuyCartWithRecharge(ctx context.Context, userID uint64, items []CartItem, days int) (*Order, error) {
	if !ValidReleaseDays(days) {
		return nil, ErrInvalidReleaseDays
	}
	if _, err := uc.users.FindByID(ctx, userID); err != nil {
		return nil, err
	}
	merged, err := mergeCartItems(items)
	if err != nil {
		return nil, err
	}
	var (
		total   decimal.Decimal
		head    *Package
		headAmt decimal.Decimal
		labels  = make([]string, 0, len(merged))
	)
	for _, it := range merged {
		pkg, err := uc.packages.FindByID(ctx, it.GoodsID)
		if err != nil {
			return nil, err
		}
		if pkg == nil || !pkg.Enabled {
			return nil, ErrPackageDisabled
		}
		unit, label, err := resolveCartSKU(pkg, it.SkuID)
		if err != nil {
			return nil, err
		}
		lineAmt := money.Round(unit.Mul(decimal.NewFromInt(int64(it.Qty))))
		if !lineAmt.IsPositive() {
			return nil, ErrInvalidAmount
		}
		total = total.Add(lineAmt)
		if head == nil || unit.GreaterThan(headAmt) {
			head = pkg
			headAmt = unit
		}
		labels = append(labels, cartLineLabelName(label, it.Qty))
	}
	total = money.Round(total)
	if head == nil || !total.IsPositive() {
		return nil, ErrInvalidAmount
	}
	snap := strings.Join(labels, "、")
	return uc.payRechargeOrder(ctx, userID, head, total, snap, snap, days)
}

func mergeCartItems(items []CartItem) ([]CartItem, error) {
	if len(items) == 0 {
		return nil, ErrInvalidAmount
	}
	type key struct{ GoodsID, SkuID uint64 }
	qtyBy := make(map[key]int, len(items))
	order := make([]key, 0, len(items))
	for _, it := range items {
		if it.GoodsID == 0 {
			return nil, ErrPackageNotFound
		}
		if it.Qty <= 0 {
			return nil, ErrInvalidAmount
		}
		k := key{GoodsID: it.GoodsID, SkuID: it.SkuID}
		if _, ok := qtyBy[k]; !ok {
			if len(order) >= maxCartDistinct {
				return nil, ErrInvalidAmount
			}
			order = append(order, k)
		}
		next := qtyBy[k] + it.Qty
		if next > maxCartQtyPerGoods {
			return nil, ErrInvalidAmount
		}
		qtyBy[k] = next
	}
	out := make([]CartItem, 0, len(order))
	for _, k := range order {
		out = append(out, CartItem{GoodsID: k.GoodsID, SkuID: k.SkuID, Qty: qtyBy[k]})
	}
	return out, nil
}

func enabledSKUs(pkg *Package) []PackageSKU {
	if pkg == nil || len(pkg.SKUs) == 0 {
		return nil
	}
	out := make([]PackageSKU, 0, len(pkg.SKUs))
	for _, sku := range pkg.SKUs {
		if sku.Enabled {
			out = append(out, sku)
		}
	}
	return out
}

func resolveCartSKU(pkg *Package, skuID uint64) (decimal.Decimal, string, error) {
	if pkg == nil {
		return decimal.Zero, "", ErrPackageDisabled
	}
	skus := enabledSKUs(pkg)
	if len(skus) == 0 {
		if skuID != 0 {
			return decimal.Zero, "", ErrSKUInvalid
		}
		return pkg.Amount, packageDisplayName(pkg), nil
	}
	if skuID == 0 {
		return decimal.Zero, "", ErrSKUInvalid
	}
	for _, sku := range skus {
		if sku.ID == skuID {
			name := strings.TrimSpace(sku.Name)
			if name == "" {
				name = strings.TrimSpace(sku.NameEn)
			}
			base := packageDisplayName(pkg)
			if name != "" {
				base = base + " / " + name
			}
			return sku.Amount, base, nil
		}
	}
	return decimal.Zero, "", ErrSKUInvalid
}

func packageDisplayName(pkg *Package) string {
	if pkg == nil {
		return ""
	}
	name := strings.TrimSpace(pkg.Title)
	if name == "" {
		name = strings.TrimSpace(pkg.GoodsDesc)
	}
	if name == "" {
		return fmt.Sprintf("#%d", pkg.ID)
	}
	return name
}

func cartLineLabel(pkg *Package, qty int) string {
	return cartLineLabelName(packageDisplayName(pkg), qty)
}

func cartLineLabelName(name string, qty int) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "#"
	}
	if qty > 1 {
		return fmt.Sprintf("%s×%d", name, qty)
	}
	return name
}

func clipRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func (uc *OrderUseCase) buyPackageWithRecharge(ctx context.Context, userID uint64, pkg *Package, days int) (*Order, error) {
	if pkg == nil {
		return nil, ErrPackageDisabled
	}
	return uc.payRechargeOrder(ctx, userID, pkg, pkg.Amount, pkg.Title, pkg.GoodsDesc, days)
}

func (uc *OrderUseCase) payRechargeOrder(ctx context.Context, userID uint64, pkg *Package, amount decimal.Decimal, title, desc string, days int) (*Order, error) {
	if pkg == nil || !pkg.Enabled {
		return nil, ErrPackageDisabled
	}
	amount = money.Round(amount)
	if !amount.IsPositive() {
		return nil, ErrInvalidAmount
	}
	title = clipRunes(title, titleSnapshotMax)
	desc = clipRunes(desc, goodsSnapshotMax)
	if uc.tx == nil {
		uc.tx = NopTx{}
	}
	var out *Order
	err := uc.tx.InTx(ctx, func(ctx context.Context) error {
		if err := uc.balances.SubRechargeBalance(ctx, userID, amount); err != nil {
			return err
		}
		o, err := uc.orders.Create(ctx, &Order{
			UserID:        userID,
			PackageID:     pkg.ID,
			Amount:        amount,
			TitleSnapshot: title,
			GoodsSnapshot: desc,
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
				Amount:      amount.Neg(),
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

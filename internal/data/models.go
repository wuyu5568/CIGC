package data

import (
	"time"

	"github.com/shopspring/decimal"
)

// 下列模型仅做表映射，不调用 AutoMigrate。

type UserModel struct {
	ID               uint64 `gorm:"primaryKey"`
	Address          string `gorm:"size:64;uniqueIndex"`
	InviterID        *uint64
	AvailableBalance decimal.Decimal `gorm:"type:decimal(36,8)"`
	RechargeBalance  decimal.Decimal `gorm:"type:decimal(36,8)"`
	FrozenBalance    decimal.Decimal `gorm:"type:decimal(36,8)"`
	FrozenIspay      decimal.Decimal `gorm:"type:decimal(36,8)"`
	IspayBalance     decimal.Decimal `gorm:"type:decimal(36,8)"`
	LockBalance      decimal.Decimal `gorm:"type:decimal(36,8)"`
	LockIspay        decimal.Decimal `gorm:"type:decimal(36,8)"`
	PaidAmount       decimal.Decimal `gorm:"type:decimal(36,8)"`
	CapEffective     decimal.Decimal `gorm:"type:decimal(36,8)"`
	DisabledAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (UserModel) TableName() string { return "users" }

type LoginChallengeModel struct {
	ID        uint64 `gorm:"primaryKey"`
	Address   string `gorm:"size:64;index"`
	Nonce     string `gorm:"size:64;uniqueIndex"`
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (LoginChallengeModel) TableName() string { return "login_challenges" }

type UserRecommendModel struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex"`
	Path      string `gorm:"size:2048"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (UserRecommendModel) TableName() string { return "user_recommends" }

type UserPlacementModel struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex"`
	SponsorID uint64 `gorm:"column:sponsor_id;index"`
	Side      string `gorm:"size:1"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (UserPlacementModel) TableName() string { return "user_placements" }

type UserMatchBalanceModel struct {
	UserID      uint64          `gorm:"primaryKey"`
	LeftRemain  decimal.Decimal `gorm:"type:decimal(36,8)"`
	RightRemain decimal.Decimal `gorm:"type:decimal(36,8)"`
	UpdatedAt   time.Time
}

func (UserMatchBalanceModel) TableName() string { return "user_match_balances" }

type UserDailyDynamicModel struct {
	UserID     uint64          `gorm:"primaryKey;column:user_id"`
	SettleDate time.Time       `gorm:"primaryKey;type:date;column:settle_date"`
	Used       decimal.Decimal `gorm:"type:decimal(36,8)"`
}

func (UserDailyDynamicModel) TableName() string { return "user_daily_dynamic" }

type CapOverflowHoldModel struct {
	ID         uint64          `gorm:"primaryKey"`
	UserID     uint64          `gorm:"column:user_id;index"`
	Value      decimal.Decimal `gorm:"type:decimal(36,8)"`
	USDT       decimal.Decimal `gorm:"column:usdt;type:decimal(36,8)"`
	Ispay      decimal.Decimal `gorm:"type:decimal(36,8)"`
	SourceType string          `gorm:"column:source_type;size:32"`
	OrderID    *uint64         `gorm:"column:order_id"`
	SettleDate time.Time       `gorm:"type:date;column:settle_date"`
	Remark     string          `gorm:"size:255"`
	CreatedAt  time.Time
	ExpiresAt  *time.Time
	ReleasedAt *time.Time
	BurnedAt   *time.Time
}

func (CapOverflowHoldModel) TableName() string { return "cap_overflow_holds" }

type MatchOrderAppliedModel struct {
	OrderID    uint64 `gorm:"primaryKey"`
	SettleDate time.Time
	CreatedAt  time.Time
}

func (MatchOrderAppliedModel) TableName() string { return "match_order_applied" }

type PackageModel struct {
	ID          uint64          `gorm:"primaryKey"`
	Amount      decimal.Decimal `gorm:"type:decimal(36,8)"`
	Title       string
	GoodsDesc   string          `gorm:"column:goods_desc"`
	DailyCap    decimal.Decimal `gorm:"column:daily_cap;type:decimal(36,8)"`
	ReleaseDays int             `gorm:"column:release_days"`
	SortOrder   int             `gorm:"column:sort_order"`
	Enabled     bool
	Image       string `gorm:"column:image;size:512"`
	Detail      string `gorm:"column:detail;type:mediumtext"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (PackageModel) TableName() string { return "packages" }

type PackageContentModel struct {
	ID        uint64 `gorm:"primaryKey"`
	PackageID uint64 `gorm:"column:package_id;uniqueIndex:uk_package_contents_locale"`
	Locale    string `gorm:"size:10;uniqueIndex:uk_package_contents_locale"`
	Title     string `gorm:"size:128"`
	GoodsDesc string `gorm:"column:goods_desc;size:512"`
	Image     string `gorm:"size:512"`
	Detail    string `gorm:"type:mediumtext"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (PackageContentModel) TableName() string { return "package_contents" }

type PackageSKUModel struct {
	ID        uint64          `gorm:"primaryKey"`
	PackageID uint64          `gorm:"column:package_id;index:idx_package_skus_package"`
	Name      string          `gorm:"size:128"`
	NameEn    string          `gorm:"column:name_en;size:128"`
	Amount    decimal.Decimal `gorm:"type:decimal(36,8)"`
	Image     string          `gorm:"size:512"`
	SortOrder int             `gorm:"column:sort_order"`
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (PackageSKUModel) TableName() string { return "package_skus" }

type OrderModel struct {
	ID            uint64 `gorm:"primaryKey"`
	OrderNo       string `gorm:"column:order_no;size:32"`
	UserID        uint64 `gorm:"index"`
	PackageID     uint64
	Amount        decimal.Decimal `gorm:"type:decimal(36,8)"`
	TitleSnapshot string          `gorm:"column:title_snapshot"`
	GoodsSnapshot string          `gorm:"column:goods_snapshot"`
	Status        string          `gorm:"size:16;index"`
	ReleaseDays   int             `gorm:"column:release_days"`
	TxHash        *string         `gorm:"column:tx_hash;size:80"`
	LogIndex      int             `gorm:"column:log_index"`
	PaidAt        *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (OrderModel) TableName() string { return "orders" }

type LedgerEntryModel struct {
	ID          uint64 `gorm:"primaryKey"`
	UserID      uint64 `gorm:"index"`
	OrderID     *uint64
	EntryType   string          `gorm:"size:32;index"`
	Amount      decimal.Decimal `gorm:"type:decimal(36,8)"`
	BalanceKind string          `gorm:"size:16"`
	SettleDate  *time.Time      `gorm:"type:date"`
	Remark      string          `gorm:"size:255"`
	CreatedAt   time.Time       `gorm:"index"`
}

func (LedgerEntryModel) TableName() string { return "ledger_entries" }

type WithdrawModel struct {
	ID             uint64          `gorm:"primaryKey"`
	UserID         uint64          `gorm:"index"`
	Amount         decimal.Decimal `gorm:"type:decimal(36,8)"`
	FeeAmount      decimal.Decimal `gorm:"type:decimal(36,8)"`
	CreditedAmount decimal.Decimal `gorm:"type:decimal(36,8)"`
	Asset          string          `gorm:"size:16"`
	Status         string          `gorm:"size:16;index"`
	Remark         string          `gorm:"size:255"`
	TxHash         string          `gorm:"column:tx_hash;size:80"`
	PayoutError    string          `gorm:"column:payout_error;size:255"`
	ReviewedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (WithdrawModel) TableName() string { return "withdraws" }

type BusinessConfigModel struct {
	ID        uint64 `gorm:"primaryKey"`
	ConfigKey string `gorm:"column:config_key;size:64;uniqueIndex"`
	Name      string `gorm:"size:128"`
	Value     string `gorm:"type:text"`
	SortOrder int    `gorm:"column:sort_order"`
	UpdatedAt time.Time
}

func (BusinessConfigModel) TableName() string { return "business_configs" }

type SettleRunModel struct {
	ID          uint64    `gorm:"primaryKey"`
	SettleDate  time.Time `gorm:"type:date;uniqueIndex"`
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

func (SettleRunModel) TableName() string { return "settle_runs" }

type ChainScanCursorModel struct {
	Name        string `gorm:"primaryKey;size:64"`
	BlockNumber uint64
	UpdatedAt   time.Time
}

func (ChainScanCursorModel) TableName() string { return "chain_scan_cursors" }

type ChainDepositModel struct {
	ID          uint64 `gorm:"primaryKey"`
	TxHash      string `gorm:"size:80"`
	LogIndex    int
	FromAddr    string          `gorm:"column:from_addr;size:64"`
	ToAddr      string          `gorm:"column:to_addr;size:64"`
	Amount      decimal.Decimal `gorm:"type:decimal(36,8)"`
	BlockNumber uint64
	Status      string `gorm:"size:16"`
	OrderID     *uint64
	Remark      string `gorm:"size:255"`
	CreatedAt   time.Time
}

func (ChainDepositModel) TableName() string { return "chain_deposits" }

type ShippingAddressModel struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"column:user_id;uniqueIndex"`
	Name      string `gorm:"size:64"`
	Contact   string `gorm:"size:64"`
	Address   string `gorm:"size:512"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ShippingAddressModel) TableName() string { return "shipping_addresses" }

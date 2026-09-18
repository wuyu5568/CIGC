package conf

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cigc/app/internal/pkg/wallet"
	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

// Bootstrap 是 YAML 根配置。密钥项以环境变量为准。
type Bootstrap struct {
	Server Server `yaml:"server"`
	Data   Data   `yaml:"data"`
	Auth   Auth   `yaml:"auth"`
	App    App    `yaml:"app"`
}

type Server struct {
	HTTP HTTP `yaml:"http"`
}

type HTTP struct {
	Addr    string        `yaml:"addr"`
	Timeout time.Duration `yaml:"timeout"`
}

type Data struct {
	Database Database `yaml:"database"`
}

type Database struct {
	Driver string `yaml:"driver"`
	Source string `yaml:"source"`
}

type Auth struct {
	JWTKey        string        `yaml:"jwt_key"`
	AdminUsername string        `yaml:"admin_username"`
	AdminPassword string        `yaml:"admin_password"`
	ChallengeTTL  time.Duration `yaml:"challenge_ttl"`
}

type App struct {
	GenesisAddress       string         `yaml:"genesis_address"`
	SettleCron           string         `yaml:"settle_cron"`
	SettleTimezone       string         `yaml:"settle_timezone"`
	AllowForceSettle     bool           `yaml:"allow_force_settle"`
	FullDownline         bool           `yaml:"full_downline"`
	PayoutEnabled        bool           `yaml:"payout_enabled"`
	PayoutCron           string         `yaml:"payout_cron"`
	BscRPC               string         `yaml:"bsc_rpc"`
	UsdtAddress          string         `yaml:"usdt_address"`
	IspayAddress         string         `yaml:"ispay_address"`
	BuyContract          string         `yaml:"buy_contract"`
	ReceiveAddress       string         `yaml:"receive_address"`
	ReceiveAddresses     []ReceiveShare `yaml:"receive_addresses"`
	DepositCron          string         `yaml:"deposit_cron"`
	DepositConfirmations int            `yaml:"deposit_confirmations"`
	HotWalletKey         string         `yaml:"hot_wallet_key"`
	PayoutMaxUSDT        float64        `yaml:"-"`
	PayoutMaxIspay       float64        `yaml:"-"`
	UploadDir            string         `yaml:"upload_dir"`
}

// ReceiveShare 是一条收款地址及其分配百分比（如 75 表示 75%）。
type ReceiveShare struct {
	Address string `yaml:"address" json:"address"`
	Percent string `yaml:"percent" json:"percent"`
}

const (
	EnvHTTPAddr             = "CIGC_HTTP_ADDR"
	EnvDatabaseDSN          = "CIGC_DATABASE_DSN"
	EnvJWTKey               = "CIGC_JWT_KEY"
	EnvAdminUsername        = "CIGC_ADMIN_USERNAME"
	EnvAdminPassword        = "CIGC_ADMIN_PASSWORD"
	EnvGenesisAddress       = "CIGC_GENESIS_ADDRESS"
	EnvSettleCron           = "CIGC_SETTLE_CRON"
	EnvSettleTimezone       = "CIGC_SETTLE_TIMEZONE"
	EnvAllowForceSettle     = "CIGC_ALLOW_FORCE_SETTLE"
	EnvFullDownline         = "CIGC_FULL_DOWNLINE"
	EnvPayoutEnabled        = "CIGC_PAYOUT_ENABLED"
	EnvPayoutCron           = "CIGC_PAYOUT_CRON"
	EnvHotWalletKey         = "CIGC_HOT_WALLET_KEY"
	EnvBscRPC               = "CIGC_BSC_RPC"
	EnvPayoutMaxUSDT        = "CIGC_PAYOUT_MAX_USDT"
	EnvPayoutMaxIspay       = "CIGC_PAYOUT_MAX_ISPAY"
	EnvUSDTAddress          = "CIGC_USDT_ADDRESS"
	EnvIspayAddress         = "CIGC_ISPAY_ADDRESS"
	EnvBuyContract          = "CIGC_BUY_CONTRACT"
	EnvReceiveAddress       = "CIGC_RECEIVE_ADDRESS"
	EnvReceiveAddresses     = "CIGC_RECEIVE_ADDRESSES"
	EnvDepositCron          = "CIGC_DEPOSIT_CRON"
	EnvDepositConfirmations = "CIGC_DEPOSIT_CONFIRMATIONS"
	EnvUploadDir            = "CIGC_UPLOAD_DIR"
)

// Load 读取 YAML，再用 CIGC_* 环境变量覆盖，最后做启动校验。
func Load(path string) (*Bootstrap, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var bc Bootstrap
	if err := yaml.Unmarshal(raw, &bc); err != nil {
		return nil, err
	}
	applyEnvOverrides(&bc)
	if err := Validate(&bc); err != nil {
		return nil, err
	}
	return &bc, nil
}

// hasReceiveShares 是否配置了收款地址列表或单地址。
func hasReceiveShares(app *App) bool {
	if app == nil {
		return false
	}
	return len(app.ReceiveAddresses) > 0 || strings.TrimSpace(app.ReceiveAddress) != ""
}

// Validate 检查必填项与打款安全开关。
func Validate(bc *Bootstrap) error {
	if bc == nil {
		return fmt.Errorf("config is nil")
	}
	if strings.TrimSpace(bc.Data.Database.Source) == "" {
		return fmt.Errorf("%s is required", EnvDatabaseDSN)
	}
	if strings.TrimSpace(bc.Auth.JWTKey) == "" {
		return fmt.Errorf("%s is required", EnvJWTKey)
	}
	if err := ValidateAppSafety(&bc.App); err != nil {
		return err
	}
	return ValidateReceive(&bc.App)
}

// ValidateReceive 多收款地址必须能规范化，且百分比合计 100。
func ValidateReceive(app *App) error {
	if app == nil || !hasReceiveShares(app) {
		return nil
	}
	items := app.ReceiveAddresses
	if len(items) == 0 && strings.TrimSpace(app.ReceiveAddress) != "" {
		items = []ReceiveShare{{Address: app.ReceiveAddress, Percent: "100"}}
	}
	sum := decimal.Zero
	seen := map[string]struct{}{}
	for i, it := range items {
		addr := wallet.NormalizeReceiveAddress(it.Address)
		if addr == "" {
			return fmt.Errorf("receive_addresses[%d]: invalid address", i)
		}
		if _, ok := seen[addr]; ok {
			return fmt.Errorf("receive_addresses[%d]: duplicate address", i)
		}
		seen[addr] = struct{}{}
		s := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(it.Percent), "%"))
		if s == "" {
			s = "100"
		}
		p, err := decimal.NewFromString(s)
		if err != nil || !p.IsPositive() {
			return fmt.Errorf("receive_addresses[%d]: bad percent", i)
		}
		sum = sum.Add(p)
	}
	if !sum.Equal(decimal.NewFromInt(100)) {
		return fmt.Errorf("receive_addresses percents must sum to 100, got %s", sum)
	}
	return nil
}

// ValidateAppSafety 开启打款时必须有上限、热钱包、RPC 和 USDT 合约。
func ValidateAppSafety(app *App) error {
	if app == nil || !app.PayoutEnabled {
		return nil
	}
	if app.PayoutMaxUSDT <= 0 {
		return fmt.Errorf("payout_enabled requires %s > 0", EnvPayoutMaxUSDT)
	}
	if strings.TrimSpace(app.HotWalletKey) == "" {
		return fmt.Errorf("payout_enabled requires %s", EnvHotWalletKey)
	}
	if strings.TrimSpace(app.BscRPC) == "" {
		return fmt.Errorf("payout_enabled requires %s", EnvBscRPC)
	}
	if strings.TrimSpace(app.UsdtAddress) == "" {
		return fmt.Errorf("payout_enabled requires %s", EnvUSDTAddress)
	}
	return nil
}

func applyEnvOverrides(bc *Bootstrap) {
	if v := strings.TrimSpace(os.Getenv(EnvHTTPAddr)); v != "" {
		bc.Server.HTTP.Addr = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvDatabaseDSN)); v != "" {
		bc.Data.Database.Source = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvJWTKey)); v != "" {
		bc.Auth.JWTKey = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvAdminUsername)); v != "" {
		bc.Auth.AdminUsername = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvAdminPassword)); v != "" {
		bc.Auth.AdminPassword = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvGenesisAddress)); v != "" {
		bc.App.GenesisAddress = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvSettleCron)); v != "" {
		bc.App.SettleCron = stripCronExpr(v)
	}
	if v := strings.TrimSpace(os.Getenv(EnvSettleTimezone)); v != "" {
		bc.App.SettleTimezone = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvAllowForceSettle)); v != "" {
		bc.App.AllowForceSettle = envTruthy(v)
	}
	if v := strings.TrimSpace(os.Getenv(EnvFullDownline)); v != "" {
		bc.App.FullDownline = envTruthy(v)
	}
	if v := strings.TrimSpace(os.Getenv(EnvHotWalletKey)); v != "" {
		bc.App.HotWalletKey = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvBscRPC)); v != "" {
		bc.App.BscRPC = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvUSDTAddress)); v != "" {
		bc.App.UsdtAddress = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvIspayAddress)); v != "" {
		bc.App.IspayAddress = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvBuyContract)); v != "" {
		bc.App.BuyContract = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvReceiveAddress)); v != "" {
		bc.App.ReceiveAddress = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvReceiveAddresses)); v != "" {
		var shares []ReceiveShare
		if err := yaml.Unmarshal([]byte(v), &shares); err == nil && len(shares) > 0 {
			bc.App.ReceiveAddresses = shares
		}
	}
	if v := strings.TrimSpace(os.Getenv(EnvDepositCron)); v != "" {
		bc.App.DepositCron = stripCronExpr(v)
	}
	if v := strings.TrimSpace(os.Getenv(EnvDepositConfirmations)); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n >= 0 {
			bc.App.DepositConfirmations = n
		}
	}
	if v := strings.TrimSpace(os.Getenv(EnvPayoutEnabled)); v != "" {
		bc.App.PayoutEnabled = envTruthy(v)
	}
	if v := strings.TrimSpace(os.Getenv(EnvPayoutCron)); v != "" {
		bc.App.PayoutCron = stripCronExpr(v)
	}
	if v := strings.TrimSpace(os.Getenv(EnvPayoutMaxUSDT)); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil && f > 0 {
			bc.App.PayoutMaxUSDT = f
		}
	}
	if v := strings.TrimSpace(os.Getenv(EnvPayoutMaxIspay)); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil && f > 0 {
			bc.App.PayoutMaxIspay = f
		}
	}
	if v := strings.TrimSpace(os.Getenv(EnvUploadDir)); v != "" {
		bc.App.UploadDir = v
	}
}

func envTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// stripCronExpr 去掉 docker compose 默认值里带进来的引号，例如 `"* * * * *"`。
func stripCronExpr(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return strings.TrimSpace(v[1 : len(v)-1])
		}
	}
	return v
}

package conf

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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
	GenesisAddress   string  `yaml:"genesis_address"`
	SettleCron       string  `yaml:"settle_cron"`
	SettleTimezone   string  `yaml:"settle_timezone"`
	AllowForceSettle bool    `yaml:"allow_force_settle"`
	PayoutEnabled    bool    `yaml:"payout_enabled"`
	PayoutCron       string  `yaml:"payout_cron"`
	BscRPC           string  `yaml:"bsc_rpc"`
	UsdtAddress      string  `yaml:"usdt_address"`
	HotWalletKey     string  `yaml:"hot_wallet_key"`
	PayoutMaxUSDT    float64 `yaml:"-"`
}

const (
	EnvHTTPAddr         = "CIGC_HTTP_ADDR"
	EnvDatabaseDSN      = "CIGC_DATABASE_DSN"
	EnvJWTKey           = "CIGC_JWT_KEY"
	EnvAdminUsername    = "CIGC_ADMIN_USERNAME"
	EnvAdminPassword    = "CIGC_ADMIN_PASSWORD"
	EnvGenesisAddress   = "CIGC_GENESIS_ADDRESS"
	EnvSettleCron       = "CIGC_SETTLE_CRON"
	EnvSettleTimezone   = "CIGC_SETTLE_TIMEZONE"
	EnvAllowForceSettle = "CIGC_ALLOW_FORCE_SETTLE"
	EnvPayoutEnabled    = "CIGC_PAYOUT_ENABLED"
	EnvHotWalletKey     = "CIGC_HOT_WALLET_KEY"
	EnvBscRPC           = "CIGC_BSC_RPC"
	EnvPayoutMaxUSDT    = "CIGC_PAYOUT_MAX_USDT"
	EnvUSDTAddress      = "CIGC_USDT_ADDRESS"
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
	return ValidateAppSafety(&bc.App)
}

// ValidateAppSafety 开启打款时必须配置单笔上限。
func ValidateAppSafety(app *App) error {
	if app == nil {
		return nil
	}
	if app.PayoutEnabled && app.PayoutMaxUSDT <= 0 {
		return fmt.Errorf("payout_enabled requires %s > 0", EnvPayoutMaxUSDT)
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
		bc.App.SettleCron = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvSettleTimezone)); v != "" {
		bc.App.SettleTimezone = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvAllowForceSettle)); v != "" {
		bc.App.AllowForceSettle = envTruthy(v)
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
	if v := strings.TrimSpace(os.Getenv(EnvPayoutEnabled)); v != "" {
		bc.App.PayoutEnabled = envTruthy(v)
	}
	if v := strings.TrimSpace(os.Getenv(EnvPayoutMaxUSDT)); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil && f > 0 {
			bc.App.PayoutMaxUSDT = f
		}
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

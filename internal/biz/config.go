package biz

import (
	"context"
	"strings"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

const (
	ConfigMinWithdrawIspay = "min_withdraw_amount_ispay"
	ConfigWithdrawFeeIspay = "withdraw_fee_rate_ispay"
	ConfigManageGens       = "manage_generations"
	ConfigOverflowHours    = "overflow_clear_hours"
	ConfigWithdrawEnabled  = "withdraw_enabled"

	defaultManageGens = 3
	maxManageGens     = 10
	minOverflowHours  = 1
	maxOverflowHours  = 720
	defaultMinIspay   = "0"
	defaultFeeIspay   = "0"
	defaultWithdrawOn = 1
)

// editableConfigKeys 管理端允许改的键。
var editableConfigKeys = map[string]struct{}{
	ConfigDirectRate:         {},
	ConfigMatchRate:          {},
	ConfigManageRate:         {},
	ConfigManageGens:         {},
	ConfigMinWithdraw:        {},
	ConfigMinWithdrawIspay:   {},
	ConfigWithdrawFeeRate:    {},
	ConfigWithdrawFeeIspay:   {},
	ConfigWithdrawDaily:      {},
	ConfigWithdrawDailyIspay: {},
	ConfigIspayPrice:         {},
	ConfigOverflowHours:      {},
	ConfigWithdrawEnabled:    {},
}

// ConfigUseCase 管理端读改 business_configs。
type ConfigUseCase struct {
	configs ConfigRepo
}

// NewConfigUseCase 构造配置用例。
func NewConfigUseCase(configs ConfigRepo) *ConfigUseCase {
	return &ConfigUseCase{configs: configs}
}

// List 按 sort_order 返回全部配置。
func (uc *ConfigUseCase) List(ctx context.Context) ([]*BusinessConfig, error) {
	if uc == nil || uc.configs == nil {
		return []*BusinessConfig{}, nil
	}
	rows, err := uc.configs.List(ctx)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []*BusinessConfig{}, nil
	}
	return rows, nil
}

// Update 按 id 改 value；只允许已知键，并做数值校验。
func (uc *ConfigUseCase) Update(ctx context.Context, id uint64, raw string) (*BusinessConfig, error) {
	if uc == nil || uc.configs == nil {
		return nil, ErrConfigNotFound
	}
	row, err := uc.configs.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrConfigNotFound
	}
	if _, ok := editableConfigKeys[row.Key]; !ok {
		return nil, ErrConfigForbidden
	}
	value, err := NormalizeConfigValue(row.Key, raw)
	if err != nil {
		return nil, err
	}
	if err := uc.configs.SetValue(ctx, id, value); err != nil {
		return nil, err
	}
	row.Value = value
	return row, nil
}

// Spot 读测试/配置中的 ispay 现价；无效则回落 2000。
func (uc *ConfigUseCase) Spot(ctx context.Context) decimal.Decimal {
	if uc == nil || uc.configs == nil {
		return IspaySpotFallback()
	}
	return configSpot(ctx, uc.configs)
}

func configSpot(ctx context.Context, configs ConfigRepo) decimal.Decimal {
	if configs == nil {
		return IspaySpotFallback()
	}
	v, err := configs.GetValue(ctx, ConfigIspayPrice)
	if err != nil || strings.TrimSpace(v) == "" {
		return IspaySpotFallback()
	}
	d, err := decimal.NewFromString(strings.TrimSpace(v))
	if err != nil || !d.IsPositive() {
		return IspaySpotFallback()
	}
	return money.Round(d)
}

// NormalizeConfigValue 规范化并校验可编辑配置。
func NormalizeConfigValue(key, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ErrConfigInvalid
	}
	d, err := decimal.NewFromString(raw)
	if err != nil {
		return "", ErrConfigInvalid
	}
	d = money.Round(d)
	switch key {
	case ConfigDirectRate, ConfigMatchRate, ConfigManageRate, ConfigWithdrawFeeRate, ConfigWithdrawFeeIspay:
		if d.IsNegative() || d.GreaterThan(decimal.NewFromInt(1)) {
			return "", ErrConfigInvalid
		}
	case ConfigMinWithdraw, ConfigIspayPrice:
		if !d.IsPositive() {
			return "", ErrConfigInvalid
		}
	case ConfigMinWithdrawIspay, ConfigWithdrawDaily, ConfigWithdrawDailyIspay:
		if d.IsNegative() {
			return "", ErrConfigInvalid
		}
	case ConfigManageGens:
		n := int(d.IntPart())
		if !d.Equal(decimal.NewFromInt(int64(n))) || n < 1 || n > maxManageGens {
			return "", ErrConfigInvalid
		}
		return decimal.NewFromInt(int64(n)).String(), nil
	case ConfigOverflowHours:
		n := int(d.IntPart())
		if !d.Equal(decimal.NewFromInt(int64(n))) || n < minOverflowHours || n > maxOverflowHours {
			return "", ErrConfigInvalid
		}
		return decimal.NewFromInt(int64(n)).String(), nil
	case ConfigWithdrawEnabled:
		n := int(d.IntPart())
		if !d.Equal(decimal.NewFromInt(int64(n))) || (n != 0 && n != 1) {
			return "", ErrConfigInvalid
		}
		return decimal.NewFromInt(int64(n)).String(), nil
	default:
		return "", ErrConfigForbidden
	}
	return d.String(), nil
}

// ConfigIntValue 读整数配置，非法则回落 fallback。
func ConfigIntValue(ctx context.Context, configs ConfigRepo, key string, fallback, min, max int) int {
	if fallback < min {
		fallback = min
	}
	if fallback > max {
		fallback = max
	}
	if configs == nil {
		return fallback
	}
	v, err := configs.GetValue(ctx, key)
	if err != nil || strings.TrimSpace(v) == "" {
		return fallback
	}
	d, err := decimal.NewFromString(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	n := int(d.IntPart())
	if !d.Equal(decimal.NewFromInt(int64(n))) || n < min || n > max {
		return fallback
	}
	return n
}

// ConfigMeta 配置页分组与说明。
func ConfigMeta(key string) (group, hint, effect string) {
	switch key {
	case ConfigDirectRate:
		return "奖励", "0.10 表示 10%。下级已支付订单金额 × 该比例。", "之后新买单立刻生效；已发放的不追溯。"
	case ConfigMatchRate:
		return "奖励", "0.10 表示对碰额 pair 的 10%。", "之后新对碰立刻生效；已发放的不追溯。"
	case ConfigManageRate:
		return "奖励", "对碰产值 × 该比例为管理奖总池，再按代数均分。", "之后新对碰立刻生效；已发放的不追溯。"
	case ConfigManageGens:
		return "奖励", "沿邀请链向上找几代已激活用户（1–10）。未激活跳过继续往上。", "之后新发的管理奖按新代数；已发的份额不追回。"
	case ConfigMinWithdraw:
		return "提现", "USDT 单笔最低申请额，单位 U。", "之后新申请生效。"
	case ConfigMinWithdrawIspay:
		return "提现", "ISPAY 单笔最低申请额；0 表示只需大于 0。", "之后新申请生效。"
	case ConfigWithdrawFeeRate:
		return "提现", "0.10 表示 10%。USDT 到账 = 申请额 − 手续费。", "之后新申请生效。"
	case ConfigWithdrawFeeIspay:
		return "提现", "0 表示免手续费。ISPAY 到账 = 申请额 − 手续费。", "之后新申请生效。链上打款仍未开通。"
	case ConfigWithdrawDaily:
		return "提现", "上海自然日 USDT 累计申请上限；0 不限制。", "立刻按当天已申请额计算剩余。"
	case ConfigWithdrawDailyIspay:
		return "提现", "上海自然日 ISPAY 累计申请上限；0 不限制。", "立刻按当天已申请额计算剩余。"
	case ConfigIspayPrice:
		return "价格", "测试用交易所现价（U）。拆一半 U / 一半 ispay 时用。", "立刻影响新入账、待释放展示。不是链上真实行情。"
	case ConfigOverflowHours:
		return "冻结", "封账（次日 0:00）后 N 小时清除超额/未激活冻结。默认 72。只减冻结、不转可提。", "只影响之后新封账的批次；已经盖了到期日的批次不变。"
	case ConfigWithdrawEnabled:
		return "提现", "1 开放提现申请，0 关闭。不影响已提交的单和链上打款开关。", "立刻生效；已提交的单仍可审核。"
	default:
		return "其他", "", "保存后生效。"
	}
}

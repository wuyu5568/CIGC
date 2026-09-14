package biz

import (
	"context"
	"strings"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

// editableConfigKeys 管理端允许改的键。
var editableConfigKeys = map[string]struct{}{
	ConfigDirectRate:         {},
	ConfigMatchRate:          {},
	ConfigManageRate:         {},
	ConfigMinWithdraw:        {},
	ConfigWithdrawFeeRate:    {},
	ConfigWithdrawDaily:      {},
	ConfigWithdrawDailyIspay: {},
	ConfigIspayPrice:         {},
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
	case ConfigDirectRate, ConfigMatchRate, ConfigManageRate, ConfigWithdrawFeeRate:
		if d.IsNegative() || d.GreaterThan(decimal.NewFromInt(1)) {
			return "", ErrConfigInvalid
		}
	case ConfigMinWithdraw, ConfigIspayPrice:
		if !d.IsPositive() {
			return "", ErrConfigInvalid
		}
	case ConfigWithdrawDaily, ConfigWithdrawDailyIspay:
		if d.IsNegative() {
			return "", ErrConfigInvalid
		}
	default:
		return "", ErrConfigForbidden
	}
	return d.String(), nil
}

package biz

import (
	"context"

	"github.com/cigc/app/internal/pkg/money"
	"github.com/shopspring/decimal"
)

const (
	ReleaseDays300 = 300
	ReleaseDays600 = 600
	ReleaseDays750 = 750

	// DefaultIspaySpot 测试阶段交易所价格；后续由 IspayPrice.Spot 替换。
	DefaultIspaySpot = "2000"

	LedgerStatic      = "static"
	LedgerStaticIspay = "static_ispay"
	LedgerDirectIspay = "direct_ispay"
	LedgerMatchIspay  = "match_ispay"
	LedgerManageIspay = "manage_ispay"

	BalanceIspay = "ispay"
)

// IspayPrice 交易所 ispay 现价。测试固定 2000，后续接外部接口。
type IspayPrice interface {
	Spot(ctx context.Context) (decimal.Decimal, error)
}

// FixedIspayPrice 固定现价（当前测试实现）。
type FixedIspayPrice struct {
	Price decimal.Decimal
}

// Spot 返回固定价；未设置则 2000。
func (p FixedIspayPrice) Spot(_ context.Context) (decimal.Decimal, error) {
	if p.Price.IsPositive() {
		return money.Round(p.Price), nil
	}
	return IspaySpotFallback(), nil
}

// NewFixedIspayPrice 测试阶段默认 2000U。
func NewFixedIspayPrice() IspayPrice {
	return FixedIspayPrice{Price: IspaySpotFallback()}
}

// IspaySpotFallback 默认测试现价。
func IspaySpotFallback() decimal.Decimal {
	return money.MustParse(DefaultIspaySpot)
}

// ReleaseBuyPrice 档位购买价：300天=1200、600天=1000、750天=800。
func ReleaseBuyPrice(days int) (decimal.Decimal, bool) {
	switch days {
	case ReleaseDays300:
		return decimal.NewFromInt(1200), true
	case ReleaseDays600:
		return decimal.NewFromInt(1000), true
	case ReleaseDays750:
		return decimal.NewFromInt(800), true
	default:
		return decimal.Zero, false
	}
}

// ValidReleaseDays 是否为合法释放档。
func ValidReleaseDays(days int) bool {
	_, ok := ReleaseBuyPrice(days)
	return ok
}

// SplitHalfIspay 把 U 计价收益对半：一半入 U，一半按现价折 ispay。
func SplitHalfIspay(value, spot decimal.Decimal) (usdt, ispay decimal.Decimal) {
	value = money.Round(value)
	spot = money.Round(spot)
	if !value.IsPositive() || !spot.IsPositive() {
		return decimal.Zero, decimal.Zero
	}
	usdt = money.Round(value.Div(decimal.NewFromInt(2)))
	ispay = money.Round(usdt.Div(spot))
	return usdt, ispay
}

// StaticDaily 按订单金额、释放天数和当天现价算出当日释放。
// 购币数 = 订单额 / 档位买价；日释放币 = 购币数 / 天数；日产值 = 日释放币 × 现价。
func StaticDaily(amount decimal.Decimal, days int, spot decimal.Decimal) (coins, dailyCoins, dailyValue, usdt, ispay decimal.Decimal, ok bool) {
	buy, ok := ReleaseBuyPrice(days)
	if !ok {
		return
	}
	amount = money.Round(amount)
	spot = money.Round(spot)
	if !amount.IsPositive() || !spot.IsPositive() {
		ok = false
		return
	}
	daysDec := decimal.NewFromInt(int64(days))
	coins = money.Round(amount.Div(buy))
	dailyCoins = money.Round(coins.Div(daysDec))
	dailyValue = money.Round(dailyCoins.Mul(spot))
	usdt, ispay = SplitHalfIspay(dailyValue, spot)
	ok = dailyValue.IsPositive()
	return
}

// OrderStaticRelease 一笔订单的静态释放进度。
type OrderStaticRelease struct {
	TodayUSDT      decimal.Decimal
	TodayIspay     decimal.Decimal
	ReleasedUSDT   decimal.Decimal
	ReleasedIspay  decimal.Decimal
	PendingUSDT    decimal.Decimal
	PendingIspay   decimal.Decimal
	SettleDate     string
}

// ComputeOrderStaticRelease 今日=当日静态释放量；已释放=历史入账；待释放=总产值-已释放产值再对半。
func ComputeOrderStaticRelease(o *Order, spot, todayUSDT, todayIspay, releasedUSDT, releasedIspay decimal.Decimal, releasedDays int) OrderStaticRelease {
	out := OrderStaticRelease{
		ReleasedUSDT:  money.Round(releasedUSDT),
		ReleasedIspay: money.Round(releasedIspay),
	}
	if o == nil || o.Status != OrderPaid || !ValidReleaseDays(o.ReleaseDays) {
		return out
	}
	spot = money.Round(spot)
	coins, _, dailyValue, dailyUSDT, dailyIspay, ok := StaticDaily(o.Amount, o.ReleaseDays, spot)
	if !ok {
		return out
	}
	if todayUSDT.IsPositive() || todayIspay.IsPositive() {
		out.TodayUSDT = money.Round(todayUSDT)
		out.TodayIspay = money.Round(todayIspay)
	} else if releasedDays < o.ReleaseDays && dailyValue.IsPositive() {
		out.TodayUSDT = dailyUSDT
		out.TodayIspay = dailyIspay
	}
	releasedValue := out.ReleasedUSDT.Add(out.ReleasedIspay.Mul(spot))
	totalValue := money.Round(coins.Mul(spot))
	pendingValue := money.Round(totalValue.Sub(releasedValue))
	if pendingValue.IsNegative() {
		pendingValue = decimal.Zero
	}
	out.PendingUSDT, out.PendingIspay = SplitHalfIspay(pendingValue, spot)
	return out
}

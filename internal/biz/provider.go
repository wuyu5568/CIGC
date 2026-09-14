package biz

import "github.com/google/wire"

// ProviderSet 是 biz 层 Wire 集合。
var ProviderSet = wire.NewSet(
	NewUserUseCase,
	NewOrderUseCase,
	NewSettleUseCase,
	NewLedgerUseCase,
	NewWithdrawUseCase,
	NewPlacementUseCase,
	NewDepositUseCase,
	NewConfigUseCase,
	NewAdjustUseCase,
	NewStatsUseCase,
)

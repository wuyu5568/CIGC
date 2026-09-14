package data

import "github.com/google/wire"

// ProviderSet 是 data 层 Wire 集合。
var ProviderSet = wire.NewSet(
	NewData,
	NewUserRepo,
	NewRecommendRepo,
	NewLedgerRepo,
	NewPackageRepo,
	NewOrderRepo,
	NewSettleRunRepo,
	NewWithdrawRepo,
	NewConfigRepo,
	NewUserBalanceRepo,
	NewPlacementRepo,
	NewMatchRepo,
	NewChainCursorRepo,
	NewChainDepositRepo,
	NewChainReader,
)

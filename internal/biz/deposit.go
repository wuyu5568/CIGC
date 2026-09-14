package biz

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cigc/app/internal/conf"
	"github.com/cigc/app/internal/pkg/money"
	"github.com/cigc/app/internal/pkg/wallet"
	"github.com/shopspring/decimal"
)

const (
	DepositMatched  = "matched"
	DepositAbnormal = "abnormal"
	DepositSkipped  = "skipped"

	ChainCursorDeposit   = "usdt_deposit"
	defaultConfirmations = 12
	defaultUSDTDecimals  = 18
	maxDepositScanBlocks = 2000
)

// ChainTransfer 是一笔已确认的 ERC20 Transfer。
type ChainTransfer struct {
	TxHash      string
	LogIndex    int
	From        string
	To          string
	AmountRaw   string // 链上整数金额（十进制或 0x 十六进制）
	BlockNumber uint64
}

// ChainDeposit 扫链入账审计行。
type ChainDeposit struct {
	ID          uint64
	TxHash      string
	LogIndex    int
	FromAddr    string
	ToAddr      string
	Amount      decimal.Decimal
	BlockNumber uint64
	Status      string
	OrderID     *uint64
	Remark      string
	CreatedAt   time.Time
}

// ChainReader 读链上日志与块高。
type ChainReader interface {
	BlockNumber(ctx context.Context) (uint64, error)
	ListTransfers(ctx context.Context, token, to string, fromBlock, toBlock uint64) ([]*ChainTransfer, error)
}

// ChainCursorRepo 扫块游标。
type ChainCursorRepo interface {
	Get(ctx context.Context, name string) (uint64, error)
	Set(ctx context.Context, name string, block uint64) error
}

// ChainDepositRepo 入账事件防重与审计。
type ChainDepositRepo interface {
	FindByEvent(ctx context.Context, txHash string, logIndex int) (*ChainDeposit, error)
	ListByOrder(ctx context.Context, orderID uint64) ([]*ChainDeposit, error)
	ListByFrom(ctx context.Context, fromAddr string, page, pageSize int) ([]*ChainDeposit, int, error)
	ListAdmin(ctx context.Context, fromAddr string, page, pageSize int) ([]*ChainDeposit, int, error)
	Create(ctx context.Context, d *ChainDeposit) error
}

// DepositPage 充值记录分页。
type DepositPage struct {
	Items []*ChainDeposit
	Total int
}

// DepositScanResult 一次扫链摘要。
type DepositScanResult struct {
	Skipped     bool
	FromBlock   uint64
	ToBlock     uint64
	HeadBlock   uint64
	Seen        int
	Matched     int
	Abnormal    int
	AlreadySeen int
}

// DepositUseCase BSC USDT 收款：注册用户转入配置地址则记入充值余额。
type DepositUseCase struct {
	users    UserRepo
	orders   OrderRepo
	deposits ChainDepositRepo
	cursors  ChainCursorRepo
	chain    ChainReader
	balances UserBalanceRepo
	ledger   LedgerRepo
	tx       TxRunner
	shares   []ReceiveShare
	byAddr   map[string]ReceiveShare
	token    string
	confirms int
	decimals int32
	now      func() time.Time
	paidHook OrderPaidHook
	mu       sync.Mutex
}

// NewDepositUseCase 构造核销用例。收款地址或 token 为空则 Scan 直接跳过。
func NewDepositUseCase(
	users UserRepo,
	orders OrderRepo,
	deposits ChainDepositRepo,
	cursors ChainCursorRepo,
	chain ChainReader,
	tx TxRunner,
	app *conf.App,
	ledger LedgerRepo,
) *DepositUseCase {
	token := ""
	confirms := defaultConfirmations
	var shares []ReceiveShare
	if app != nil {
		token = wallet.NormalizeOrEmpty(app.UsdtAddress)
		if app.DepositConfirmations > 0 {
			confirms = app.DepositConfirmations
		}
		if parsed, err := ParseReceiveShares(app); err == nil {
			shares = parsed
		}
	}
	if tx == nil {
		tx = NopTx{}
	}
	var bal UserBalanceRepo
	if b, ok := any(users).(UserBalanceRepo); ok {
		bal = b
	}
	return &DepositUseCase{
		users:    users,
		orders:   orders,
		deposits: deposits,
		cursors:  cursors,
		chain:    chain,
		balances: bal,
		ledger:   ledger,
		tx:       tx,
		shares:   shares,
		byAddr:   receiveSet(shares),
		token:    token,
		confirms: confirms,
		decimals: defaultUSDTDecimals,
		now:      time.Now,
	}
}

// SetPaidHook 链上核销完成后秒结。
func (uc *DepositUseCase) SetPaidHook(h OrderPaidHook) {
	if uc != nil {
		uc.paidHook = h
	}
}

// Enabled 是否已配置收款地址与合约、RPC 读端。
func (uc *DepositUseCase) Enabled() bool {
	return uc != nil && len(uc.shares) > 0 && uc.token != "" && uc.chain != nil && uc.deposits != nil && uc.cursors != nil
}

// Shares 返回已规范化的收款分配。
func (uc *DepositUseCase) Shares() []ReceiveShare {
	if uc == nil {
		return nil
	}
	out := make([]ReceiveShare, len(uc.shares))
	copy(out, uc.shares)
	return out
}

// PayPlan 按订单金额拆出各地址应付份额。
func (uc *DepositUseCase) PayPlan(amount decimal.Decimal) []ReceivePayItem {
	if uc == nil {
		return nil
	}
	return AllocateAmounts(amount, uc.shares)
}

// Scan 从游标扫到 head-confirmations，处理 Transfer。
func (uc *DepositUseCase) Scan(ctx context.Context) (*DepositScanResult, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	res := &DepositScanResult{}
	if !uc.Enabled() {
		res.Skipped = true
		return res, nil
	}
	head, err := uc.chain.BlockNumber(ctx)
	if err != nil {
		return nil, err
	}
	res.HeadBlock = head
	if head <= uint64(uc.confirms) {
		res.Skipped = true
		return res, nil
	}
	safe := head - uint64(uc.confirms)
	cursor, err := uc.cursors.Get(ctx, ChainCursorDeposit)
	if err != nil {
		return nil, err
	}
	from := cursor + 1
	if cursor == 0 {
		if safe > 500 {
			from = safe - 500
		} else {
			from = 0
		}
	}
	if from > safe {
		res.FromBlock = from
		res.ToBlock = safe
		return res, nil
	}
	to := safe
	if to-from+1 > maxDepositScanBlocks {
		to = from + maxDepositScanBlocks - 1
	}
	res.FromBlock = from
	res.ToBlock = to

	var logs []*ChainTransfer
	for _, sh := range uc.shares {
		part, err := uc.chain.ListTransfers(ctx, uc.token, sh.Address, from, to)
		if err != nil {
			return nil, err
		}
		logs = append(logs, part...)
	}
	sortTransfers(logs)
	for _, tr := range logs {
		outcome, err := uc.processTransfer(ctx, tr)
		if err != nil {
			return res, err
		}
		res.Seen++
		switch outcome {
		case DepositMatched:
			res.Matched++
		case DepositAbnormal:
			res.Abnormal++
		case DepositSkipped:
			res.AlreadySeen++
		}
	}
	if err := uc.cursors.Set(ctx, ChainCursorDeposit, to); err != nil {
		return res, err
	}
	return res, nil
}

// ProcessTransfer 处理单笔（测试与补扫用）。
func (uc *DepositUseCase) ProcessTransfer(ctx context.Context, tr *ChainTransfer) (string, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	return uc.processTransfer(ctx, tr)
}

func (uc *DepositUseCase) processTransfer(ctx context.Context, tr *ChainTransfer) (string, error) {
	if tr == nil {
		return DepositSkipped, nil
	}
	txHash := strings.ToLower(strings.TrimSpace(tr.TxHash))
	from := wallet.NormalizeOrEmpty(tr.From)
	to := wallet.NormalizeReceiveAddress(tr.To)
	if txHash == "" || from == "" {
		return DepositSkipped, nil
	}
	if to != "" {
		if _, ok := shareOf(uc.byAddr, to); !ok {
			return DepositSkipped, nil
		}
	}
	if existing, err := uc.deposits.FindByEvent(ctx, txHash, tr.LogIndex); err != nil {
		return "", err
	} else if existing != nil {
		return DepositSkipped, nil
	}
	if o, err := uc.orders.FindByTxEvent(ctx, txHash, tr.LogIndex); err != nil {
		return "", err
	} else if o != nil {
		return DepositSkipped, nil
	}

	amount, err := rawTokenToDecimal(tr.AmountRaw, uc.decimals)
	if err != nil || !amount.IsPositive() {
		amt := decimal.Zero
		if err == nil {
			amt = money.Round(amount)
		}
		return uc.saveAbnormal(ctx, tr, txHash, from, to, amt, "invalid amount")
	}
	amount = money.Round(amount)

	user, err := uc.users.FindByAddress(ctx, from)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return uc.saveAbnormal(ctx, tr, txHash, from, to, amount, "unknown sender")
		}
		return "", err
	}
	if uc.balances == nil {
		return "", fmt.Errorf("recharge credit: no balance repo")
	}

	credited := false
	err = uc.tx.InTx(ctx, func(ctx context.Context) error {
		if existing, err := uc.deposits.FindByEvent(ctx, txHash, tr.LogIndex); err != nil {
			return err
		} else if existing != nil {
			return nil
		}
		if err := uc.balances.AddRechargeBalance(ctx, user.ID, amount); err != nil {
			return err
		}
		if uc.ledger != nil {
			if err := uc.ledger.Create(ctx, &LedgerEntry{
				UserID:      user.ID,
				EntryType:   LedgerRecharge,
				Amount:      amount,
				BalanceKind: BalanceRecharge,
				Remark:      fmt.Sprintf("chain deposit %s#%d", txHash, tr.LogIndex),
			}); err != nil {
				return err
			}
		}
		if err := uc.deposits.Create(ctx, &ChainDeposit{
			TxHash:      txHash,
			LogIndex:    tr.LogIndex,
			FromAddr:    from,
			ToAddr:      firstNonEmptyAddr(to, firstReceive(uc.shares)),
			Amount:      amount,
			BlockNumber: tr.BlockNumber,
			Status:      DepositMatched,
			Remark:      "recharge",
		}); err != nil {
			return err
		}
		credited = true
		return nil
	})
	if err != nil {
		return "", err
	}
	if !credited {
		return DepositSkipped, nil
	}
	return DepositMatched, nil
}

// ListUser 当前用户充值记录。
func (uc *DepositUseCase) ListUser(ctx context.Context, userID uint64, page int) (*DepositPage, error) {
	if page < 1 {
		page = 1
	}
	u, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	rows, total, err := uc.deposits.ListByFrom(ctx, u.Address, page, DefaultRewardPageSize)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []*ChainDeposit{}
	}
	return &DepositPage{Items: rows, Total: total}, nil
}

// ListAdmin 管理端全部充值记录，可按发送地址筛。
func (uc *DepositUseCase) ListAdmin(ctx context.Context, address string, page int) (*DepositPage, error) {
	if page < 1 {
		page = 1
	}
	rows, total, err := uc.deposits.ListAdmin(ctx, strings.TrimSpace(address), page, DefaultRewardPageSize)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []*ChainDeposit{}
	}
	return &DepositPage{Items: rows, Total: total}, nil
}

func sortTransfers(rows []*ChainTransfer) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].BlockNumber != rows[j].BlockNumber {
			return rows[i].BlockNumber < rows[j].BlockNumber
		}
		if rows[i].LogIndex != rows[j].LogIndex {
			return rows[i].LogIndex < rows[j].LogIndex
		}
		return rows[i].TxHash < rows[j].TxHash
	})
}

func (uc *DepositUseCase) saveAbnormal(ctx context.Context, tr *ChainTransfer, txHash, from, to string, amount decimal.Decimal, remark string) (string, error) {
	err := uc.deposits.Create(ctx, &ChainDeposit{
		TxHash:      txHash,
		LogIndex:    tr.LogIndex,
		FromAddr:    from,
		ToAddr:      firstNonEmptyAddr(to, firstReceive(uc.shares)),
		Amount:      amount,
		BlockNumber: tr.BlockNumber,
		Status:      DepositAbnormal,
		Remark:      remark,
	})
	if err != nil {
		if existing, findErr := uc.deposits.FindByEvent(ctx, txHash, tr.LogIndex); findErr == nil && existing != nil {
			return DepositSkipped, nil
		}
		return "", err
	}
	return DepositAbnormal, nil
}

func firstNonEmptyAddr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func rawTokenToDecimal(raw string, decimals int32) (decimal.Decimal, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return decimal.Zero, fmt.Errorf("empty amount")
	}
	bi := new(big.Int)
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		body := strings.TrimPrefix(strings.TrimPrefix(raw, "0x"), "0X")
		if body == "" {
			return decimal.Zero, nil
		}
		if _, ok := bi.SetString(body, 16); !ok {
			return decimal.Zero, fmt.Errorf("bad hex amount")
		}
	} else {
		if _, ok := bi.SetString(raw, 10); !ok {
			return decimal.Zero, fmt.Errorf("bad amount")
		}
	}
	d := decimal.NewFromBigInt(bi, -decimals)
	return money.Round(d), nil
}

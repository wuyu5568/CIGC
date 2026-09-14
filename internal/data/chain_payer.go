package data

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/cigc/app/internal/biz"
	"github.com/cigc/app/internal/conf"
	"github.com/cigc/app/internal/pkg/ethtx"
	"github.com/cigc/app/internal/pkg/wallet"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/shopspring/decimal"
)

type chainPayer struct {
	rpc   *ethRPCClient
	key   *secp256k1.PrivateKey
	from  string
	token string
}

// NewChainPayer 热钱包 USDT 打款；未开打款或密钥/RPC 无效则 nil。
func NewChainPayer(app *conf.App) biz.ChainPayer {
	if app == nil || !app.PayoutEnabled {
		return nil
	}
	key, err := ethtx.ParsePrivateKey(app.HotWalletKey)
	if err != nil {
		return nil
	}
	reader := NewChainReader(app)
	rpc, ok := reader.(*ethRPCClient)
	if !ok || rpc == nil {
		return nil
	}
	token := wallet.NormalizeOrEmpty(app.UsdtAddress)
	if token == "" {
		return nil
	}
	return &chainPayer{
		rpc:   rpc,
		key:   key,
		from:  ethtx.AddressFromKey(key),
		token: token,
	}
}

func (p *chainPayer) TransferUSDT(ctx context.Context, to string, amount decimal.Decimal) (string, error) {
	if p == nil || p.rpc == nil || p.key == nil {
		return "", biz.ErrPayoutDisabled
	}
	to = wallet.NormalizeOrEmpty(to)
	if to == "" {
		return "", fmt.Errorf("invalid payout address")
	}
	rawAmt, err := ethtx.TokenRaw(amount, 18)
	if err != nil {
		return "", err
	}
	data, err := ethtx.ERC20TransferData(to, rawAmt)
	if err != nil {
		return "", err
	}
	chainID, err := p.rpc.chainID(ctx)
	if err != nil {
		return "", err
	}
	nonce, err := p.rpc.nonce(ctx, p.from)
	if err != nil {
		return "", err
	}
	gasPrice, err := p.rpc.gasPrice(ctx)
	if err != nil {
		return "", err
	}
	gas := uint64(80000)
	if g, err := p.rpc.estimateGas(ctx, p.from, p.token, data); err == nil && g > 21000 {
		gas = g + g/5
	}
	raw, err := ethtx.SignLegacyTx(p.key, chainID, nonce, gasPrice, new(big.Int).SetUint64(gas), p.token, big.NewInt(0), data)
	if err != nil {
		return "", err
	}
	var hash string
	if err := p.rpc.call(ctx, "eth_sendRawTransaction", []any{"0x" + hex.EncodeToString(raw)}, &hash); err != nil {
		return "", err
	}
	hash = strings.ToLower(strings.TrimSpace(hash))
	if hash == "" {
		return "", fmt.Errorf("empty tx hash")
	}
	return hash, nil
}

func (p *chainPayer) Receipt(ctx context.Context, txHash string) (ok, pending bool, err error) {
	if p == nil || p.rpc == nil {
		return false, false, biz.ErrPayoutDisabled
	}
	txHash = strings.ToLower(strings.TrimSpace(txHash))
	if txHash == "" {
		return false, false, fmt.Errorf("empty tx hash")
	}
	var receipt struct {
		Status string `json:"status"`
	}
	if err := p.rpc.call(ctx, "eth_getTransactionReceipt", []any{txHash}, &receipt); err != nil {
		if strings.Contains(err.Error(), "empty result") {
			return false, true, nil
		}
		return false, false, err
	}
	switch strings.ToLower(strings.TrimSpace(receipt.Status)) {
	case "0x1", "1":
		return true, false, nil
	case "0x0", "0":
		return false, false, nil
	default:
		return false, true, nil
	}
}

func (c *ethRPCClient) chainID(ctx context.Context) (*big.Int, error) {
	var hexNum string
	if err := c.call(ctx, "eth_chainId", []any{}, &hexNum); err != nil {
		return nil, err
	}
	n, err := parseHexUint64(hexNum)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetUint64(n), nil
}

func (c *ethRPCClient) nonce(ctx context.Context, from string) (*big.Int, error) {
	var hexNum string
	if err := c.call(ctx, "eth_getTransactionCount", []any{from, "pending"}, &hexNum); err != nil {
		return nil, err
	}
	n, err := parseHexUint64(hexNum)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetUint64(n), nil
}

func (c *ethRPCClient) gasPrice(ctx context.Context) (*big.Int, error) {
	var hexNum string
	if err := c.call(ctx, "eth_gasPrice", []any{}, &hexNum); err != nil {
		return nil, err
	}
	n := new(big.Int)
	s := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(hexNum), "0x"), "0X")
	if _, ok := n.SetString(s, 16); !ok {
		return nil, fmt.Errorf("bad gas price")
	}
	return n, nil
}

func (c *ethRPCClient) estimateGas(ctx context.Context, from, to string, data []byte) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var hexNum string
	err := c.call(ctx, "eth_estimateGas", []any{map[string]any{
		"from": from,
		"to":   to,
		"data": "0x" + hex.EncodeToString(data),
	}}, &hexNum)
	if err != nil {
		return 0, err
	}
	return parseHexUint64(hexNum)
}

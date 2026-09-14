package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/cigc/app/internal/biz"
	"github.com/cigc/app/internal/conf"
	"github.com/cigc/app/internal/pkg/wallet"
)

// Transfer topic0 = keccak256("Transfer(address,address,uint256)")
const transferTopic0 = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

type ethRPCClient struct {
	url    string
	client *http.Client
}

// NewChainReader 构造 JSON-RPC 链读端；RPC 为空则返回 nil（核销关闭）。
func NewChainReader(app *conf.App) biz.ChainReader {
	if app == nil || strings.TrimSpace(app.BscRPC) == "" {
		return nil
	}
	return &ethRPCClient{
		url: strings.TrimSpace(app.BscRPC),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *ethRPCClient) call(ctx context.Context, method string, params any, result any) error {
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if envelope.Error != nil {
		return fmt.Errorf("rpc %s: %s", method, envelope.Error.Message)
	}
	if len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		return fmt.Errorf("rpc %s: empty result", method)
	}
	return json.Unmarshal(envelope.Result, result)
}

func (c *ethRPCClient) BlockNumber(ctx context.Context) (uint64, error) {
	var hexNum string
	if err := c.call(ctx, "eth_blockNumber", []any{}, &hexNum); err != nil {
		return 0, err
	}
	return parseHexUint64(hexNum)
}

func (c *ethRPCClient) ListTransfers(ctx context.Context, token, to string, fromBlock, toBlock uint64) ([]*biz.ChainTransfer, error) {
	token = wallet.NormalizeOrEmpty(token)
	to = wallet.NormalizeReceiveAddress(to)
	if token == "" || to == "" {
		return nil, nil
	}
	toTopic := topicAddress(to)
	var logs []struct {
		Address         string   `json:"address"`
		Topics          []string `json:"topics"`
		Data            string   `json:"data"`
		BlockNumber     string   `json:"blockNumber"`
		TransactionHash string   `json:"transactionHash"`
		LogIndex        string   `json:"logIndex"`
	}
	err := c.call(ctx, "eth_getLogs", []any{map[string]any{
		"fromBlock": fmt.Sprintf("0x%x", fromBlock),
		"toBlock":   fmt.Sprintf("0x%x", toBlock),
		"address":   token,
		"topics":    []any{transferTopic0, nil, toTopic},
	}}, &logs)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.ChainTransfer, 0, len(logs))
	for _, lg := range logs {
		if len(lg.Topics) < 3 {
			continue
		}
		fromAddr := addressFromTopic(lg.Topics[1])
		toAddr := addressFromTopic(lg.Topics[2])
		bn, err := parseHexUint64(lg.BlockNumber)
		if err != nil {
			return nil, err
		}
		li, err := parseHexUint64(lg.LogIndex)
		if err != nil {
			return nil, err
		}
		out = append(out, &biz.ChainTransfer{
			TxHash:      strings.ToLower(lg.TransactionHash),
			LogIndex:    int(li),
			From:        fromAddr,
			To:          toAddr,
			AmountRaw:   lg.Data,
			BlockNumber: bn,
		})
	}
	return out, nil
}

func topicAddress(addr string) string {
	addr = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(addr)), "0x")
	if len(addr) < 40 {
		addr = strings.Repeat("0", 40-len(addr)) + addr
	}
	return "0x" + strings.Repeat("0", 24) + addr
}

func addressFromTopic(topic string) string {
	topic = strings.TrimPrefix(strings.ToLower(topic), "0x")
	if len(topic) < 40 {
		return ""
	}
	return "0x" + topic[len(topic)-40:]
}

func parseHexUint64(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if s == "" {
		return 0, nil
	}
	n := new(big.Int)
	if _, ok := n.SetString(s, 16); !ok {
		return 0, fmt.Errorf("bad hex uint: %s", s)
	}
	return n.Uint64(), nil
}

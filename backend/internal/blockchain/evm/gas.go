package evm

import (
	"context"
	"encoding/json"
	"fmt"
)

// GasPrice mengembalikan harga gas saat ini (wei).
func (c *Client) GasPrice(ctx context.Context) (uint64, error) {
	var raw json.RawMessage
	if err := c.call(ctx, "eth_gasPrice", []any{}, &raw); err != nil {
		return 0, err
	}
	value, err := parseHexInt(raw)
	if err != nil {
		return 0, err
	}
	return value.Uint64(), nil
}

// ChainID mengembalikan chain id jaringan.
func (c *Client) ChainID(ctx context.Context) (int64, error) {
	var raw json.RawMessage
	if err := c.call(ctx, "eth_chainId", []any{}, &raw); err != nil {
		return 0, err
	}
	value, err := parseHexInt(raw)
	if err != nil {
		return 0, err
	}
	return value.Int64(), nil
}

// Nonce mengembalikan nonce (jumlah transaksi terkirim) untuk alamat.
func (c *Client) Nonce(ctx context.Context, address string) (uint64, error) {
	var raw json.RawMessage
	if err := c.call(ctx, "eth_getTransactionCount", []any{address, "pending"}, &raw); err != nil {
		return 0, err
	}
	value, err := parseHexInt(raw)
	if err != nil {
		return 0, err
	}
	return value.Uint64(), nil
}

// EstimateGas memperkirakan gas untuk sebuah transaksi.
func (c *Client) EstimateGas(ctx context.Context, call map[string]string) (uint64, error) {
	var raw json.RawMessage
	if err := c.call(ctx, "eth_estimateGas", []any{call}, &raw); err != nil {
		return 0, err
	}
	value, err := parseHexInt(raw)
	if err != nil {
		return 0, err
	}
	return value.Uint64(), nil
}

// SendRawTransaction menyiarkan transaksi yang sudah ditandatangani dan
// mengembalikan hash transaksi.
func (c *Client) SendRawTransaction(ctx context.Context, rawHex string) (string, error) {
	if rawHex == "" {
		return "", fmt.Errorf("evm: raw transaction kosong")
	}
	if !hasHexPrefix(rawHex) {
		rawHex = "0x" + rawHex
	}
	var hash string
	if err := c.call(ctx, "eth_sendRawTransaction", []any{rawHex}, &hash); err != nil {
		return "", err
	}
	return hash, nil
}

func hasHexPrefix(value string) bool {
	return len(value) >= 2 && value[0] == '0' && (value[1] == 'x' || value[1] == 'X')
}

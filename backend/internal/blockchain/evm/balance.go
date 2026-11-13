package evm

import (
	"context"
	"fmt"
)

// Balance mengembalikan saldo native (dalam satuan terkecil).
// TODO: implementasikan JSON-RPC eth_getBalance.
func (c *Client) Balance(ctx context.Context, address string) (string, error) {
	if c.rpcURL == "" {
		return "", fmt.Errorf("evm: rpc url kosong")
	}
	return "0", nil
}

package evm

import (
	"context"
	"fmt"
)

// Token mendeskripsikan token ERC-20.
type Token struct {
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
}

// TokenBalance mengembalikan saldo token ERC-20.
// TODO: implementasikan eth_call ke contract balanceOf.
func (c *Client) TokenBalance(ctx context.Context, token Token, owner string) (string, error) {
	return "", fmt.Errorf("evm: token balance belum diimplementasikan")
}

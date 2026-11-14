package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Balance mengembalikan saldo native (dalam satuan terkecil, wei).
func (c *Client) Balance(ctx context.Context, address string) (string, error) {
	var raw json.RawMessage
	if err := c.call(ctx, "eth_getBalance", []any{address, "latest"}, &raw); err != nil {
		return "", err
	}
	value, err := parseHexInt(raw)
	if err != nil {
		return "", fmt.Errorf("evm: parse saldo: %w", err)
	}
	return value.String(), nil
}

// TokenBalance mengembalikan saldo token ERC-20 (dalam satuan terkecil).
func (c *Client) TokenBalance(ctx context.Context, token Token, owner string) (string, error) {
	call := map[string]string{
		"to":   token.Address,
		"data": "0x70a08231" + padAddress(owner),
	}
	var raw json.RawMessage
	if err := c.call(ctx, "eth_call", []any{call, "latest"}, &raw); err != nil {
		return "", err
	}
	value, err := parseHexInt(raw)
	if err != nil {
		return "", fmt.Errorf("evm: parse saldo token: %w", err)
	}
	return value.String(), nil
}

// BatchBalances mengambil saldo native + token banyak sekaligus lewat satu
// request batch: eth_getBalance untuk native dan eth_call balanceOf untuk token.
// Hasil dipetakan berdasarkan alamat (native) atau alamat token (ERC-20).
func (c *Client) BatchBalances(ctx context.Context, addresses []string, tokens []Token) (map[string]string, error) {
	out := make(map[string]string)
	if len(addresses) == 0 {
		return out, nil
	}

	type entry struct {
		address string
		token   *Token
	}
	var entries []entry
	var reqs []rpcRequest

	for _, address := range addresses {
		entries = append(entries, entry{address: address})
		reqs = append(reqs, rpcRequest{
			JSONRPC: "2.0",
			ID:      len(reqs) + 1,
			Method:  "eth_getBalance",
			Params:  []any{address, "latest"},
		})

		for i := range tokens {
			token := tokens[i]
			entries = append(entries, entry{address: address, token: &token})
			reqs = append(reqs, rpcRequest{
				JSONRPC: "2.0",
				ID:      len(reqs) + 1,
				Method:  "eth_call",
				Params: []any{
					map[string]string{
						"to":   token.Address,
						"data": "0x70a08231" + padAddress(address),
					},
					"latest",
				},
			})
		}
	}

	results, err := c.batch(ctx, reqs)
	if err != nil {
		return nil, err
	}

	for i, raw := range results {
		if len(raw) == 0 || i >= len(entries) {
			continue
		}
		value, err := parseHexInt(raw)
		if err != nil {
			continue
		}
		item := entries[i]
		if item.token == nil {
			out["native:"+strings.ToLower(item.address)] = value.String()
			continue
		}
		out[strings.ToLower(item.token.Address)+":"+strings.ToLower(item.address)] = value.String()
	}
	return out, nil
}

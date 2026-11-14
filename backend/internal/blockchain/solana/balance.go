package solana

import (
	"context"
	"fmt"
	"math/big"
)

// TokenBalances membaca saldo SPL token (satuan terkecil) untuk satu owner
// lewat getMultipleAccounts pada Associated Token Account tiap mint. RPC
// "indexed" seperti getTokenAccountsByOwner tidak dipakai supaya tetap jalan
// di endpoint publik gratis.
func (c *Client) TokenBalances(ctx context.Context, owner string, tokens []Token) (map[string]string, error) {
	out := make(map[string]string)
	if owner == "" || len(tokens) == 0 {
		return out, nil
	}

	refs := TokenAccountRefs(owner, tokens)
	if len(refs) == 0 {
		return out, nil
	}

	addresses := make([]string, 0, len(refs))
	for _, ref := range refs {
		addresses = append(addresses, ref.Address)
	}

	var result struct {
		Value []*accountValue `json:"value"`
	}
	params := []any{
		addresses,
		map[string]string{"encoding": "jsonParsed", "commitment": "confirmed"},
	}
	if err := c.call(ctx, "getMultipleAccounts", params, &result); err != nil {
		return nil, err
	}

	for i, account := range result.Value {
		if account == nil || i >= len(refs) {
			continue
		}
		mint, amount, _, ok := tokenInfo(account.Data)
		if !ok {
			continue
		}
		ref := refs[i]
		if mint != ref.Token.Mint {
			continue
		}
		current, _ := new(big.Int).SetString(out[mint], 10)
		if current == nil {
			current = new(big.Int)
		}
		add, ok := new(big.Int).SetString(amount, 10)
		if !ok {
			continue
		}
		out[mint] = current.Add(current, add).String()
	}
	return out, nil
}

// FormatUnits mengubah satuan terkecil menjadi string desimal.
func FormatUnits(value *big.Int, decimals uint8) string {
	return formatUnits(value, decimals)
}

// ParseUnits mengubah string desimal menjadi satuan terkecil.
func ParseUnits(amount string, decimals uint8) (*big.Int, error) {
	return parseUnits(amount, decimals)
}

func parseUnits(amount string, decimals uint8) (*big.Int, error) {
	amount = trimSpace(amount)
	if amount == "" {
		return nil, fmt.Errorf("solana: jumlah kosong")
	}

	value, ok := new(big.Rat).SetString(amount)
	if !ok || value.Sign() < 0 {
		return nil, fmt.Errorf("solana: jumlah tidak valid: %s", amount)
	}

	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	value.Mul(value, new(big.Rat).SetInt(scale))
	if !value.IsInt() {
		return nil, fmt.Errorf("solana: jumlah punya terlalu banyak desimal")
	}
	return new(big.Int).Set(value.Num()), nil
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

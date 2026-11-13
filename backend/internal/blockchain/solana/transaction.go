package solana

import (
	"context"
	"fmt"
)

// Transaction adalah representasi transaksi Solana.
type Transaction struct {
	Signature string `json:"signature"`
	Slot      uint64 `json:"slot"`
	Success   bool   `json:"success"`
}

// TransactionBySignature mengambil detail transaksi.
// TODO: implementasikan getTransaction.
func (c *Client) TransactionBySignature(ctx context.Context, signature string) (Transaction, error) {
	return Transaction{}, fmt.Errorf("solana: transaction belum diimplementasikan")
}

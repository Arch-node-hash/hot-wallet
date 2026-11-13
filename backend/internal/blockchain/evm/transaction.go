package evm

import (
	"context"
	"fmt"
)

// Transaction adalah representasi transaksi EVM.
type Transaction struct {
	Hash   string `json:"hash"`
	From   string `json:"from"`
	To     string `json:"to"`
	Value  string `json:"value"`
	Status uint64 `json:"status"`
}

// TransactionByHash mengambil detail transaksi.
// TODO: implementasikan eth_getTransactionByHash + eth_getTransactionReceipt.
func (c *Client) TransactionByHash(ctx context.Context, hash string) (Transaction, error) {
	return Transaction{}, fmt.Errorf("evm: transaction belum diimplementasikan")
}

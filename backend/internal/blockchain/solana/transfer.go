package solana

import (
	"context"
	"fmt"
)

// TransferRequest adalah parameter transfer SOL.
type TransferRequest struct {
	From   string
	To     string
	Amount uint64
}

// Transfer mengirim SOL.
// TODO: implementasikan pembuatan + signing transaksi.
func (c *Client) Transfer(ctx context.Context, req TransferRequest) (string, error) {
	return "", fmt.Errorf("solana: transfer belum diimplementasikan")
}

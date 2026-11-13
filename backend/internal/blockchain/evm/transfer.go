package evm

import (
	"context"
	"fmt"
)

// TransferRequest adalah parameter transfer native token.
type TransferRequest struct {
	From   string
	To     string
	Amount string
}

// Transfer mengirim native token.
// TODO: implementasikan signing + eth_sendRawTransaction.
func (c *Client) Transfer(ctx context.Context, req TransferRequest) (string, error) {
	return "", fmt.Errorf("evm: transfer belum diimplementasikan")
}

package evm

import (
	"context"
	"fmt"
)

// GasPrice mengembalikan harga gas saat ini.
// TODO: implementasikan eth_gasPrice.
func (c *Client) GasPrice(ctx context.Context) (uint64, error) {
	return 0, fmt.Errorf("evm: gas price belum diimplementasikan")
}

package network

import (
	"context"
	"time"
)

// RPC adalah kontrak minimum client JSON-RPC.
type RPC interface {
	Call(ctx context.Context, method string, params any, result any) error
}

// RPCConfig menyimpan konfigurasi koneksi RPC.
type RPCConfig struct {
	URL     string
	Timeout time.Duration
}

// TimeoutOrDefault mengembalikan timeout yang aman.
func (c RPCConfig) TimeoutOrDefault() time.Duration {
	if c.Timeout <= 0 {
		return 10 * time.Second
	}
	return c.Timeout
}

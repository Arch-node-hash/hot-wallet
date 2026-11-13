package solana

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Client adalah client minimum untuk jaringan Solana.
type Client struct {
	rpcURL string
	http   *http.Client
}

func NewClient(rpcURL string) *Client {
	return &Client{
		rpcURL: rpcURL,
		http:   &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) RPCURL() string {
	return c.rpcURL
}

// Balance mengembalikan saldo native dalam lamports.
func (c *Client) Balance(ctx context.Context, address string) (uint64, error) {
	var result struct {
		Value uint64 `json:"value"`
	}
	if err := c.call(ctx, "getBalance", []any{address}, &result); err != nil {
		return 0, err
	}
	return result.Value, nil
}

// BatchBalances mengambil saldo native banyak alamat sekaligus.
func (c *Client) BatchBalances(ctx context.Context, addresses []string) (map[string]uint64, error) {
	if len(addresses) == 0 {
		return map[string]uint64{}, nil
	}

	reqs := make([]rpcRequest, 0, len(addresses))
	for i, address := range addresses {
		reqs = append(reqs, rpcRequest{
			JSONRPC: "2.0",
			ID:      i + 1,
			Method:  "getBalance",
			Params:  []any{address},
		})
	}

	results, err := c.batch(ctx, reqs)
	if err != nil {
		return nil, err
	}

	out := make(map[string]uint64, len(addresses))
	for i, raw := range results {
		if len(raw) == 0 {
			continue
		}
		var value struct {
			Value uint64 `json:"value"`
		}
		if err := json.Unmarshal(raw, &value); err != nil {
			continue
		}
		out[addresses[i]] = value.Value
	}
	return out, nil
}

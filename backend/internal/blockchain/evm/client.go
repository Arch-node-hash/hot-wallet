package evm

import (
	"net/http"
	"time"
)

// Client adalah client minimum untuk jaringan EVM.
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

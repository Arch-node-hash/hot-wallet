package evm

// Client adalah client minimum untuk jaringan EVM.
type Client struct {
	rpcURL string
}

func NewClient(rpcURL string) *Client {
	return &Client{rpcURL: rpcURL}
}

func (c *Client) RPCURL() string {
	return c.rpcURL
}

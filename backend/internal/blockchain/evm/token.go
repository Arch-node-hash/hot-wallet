package evm

// Token mendeskripsikan token ERC-20.
type Token struct {
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
}

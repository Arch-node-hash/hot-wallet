package network

// Chain adalah jenis chain yang didukung.
type Chain string

const (
	ChainEVM    Chain = "evm"
	ChainSolana Chain = "solana"
)

// Network mendeskripsikan satu jaringan blockchain.
type Network struct {
	Name        string `json:"name"`
	Chain       Chain  `json:"chain"`
	ChainID     int64  `json:"chain_id,omitempty"`
	RPCURL      string `json:"rpc_url"`
	WSSURL      string `json:"wss_url,omitempty"`
	ExplorerURL string `json:"explorer_url,omitempty"`
	NativeSymbol   string `json:"native_symbol,omitempty"`
	NativeDecimals uint8  `json:"native_decimals,omitempty"`
}

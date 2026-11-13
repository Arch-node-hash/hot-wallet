package network

import "strings"

// Defaults mengembalikan identitas jaringan yang didukung tanpa endpoint.
// URL RPC/WSS tidak di-hardcode di sini: nilainya berasal dari env saat seed
// dan disimpan di tabel rpc_configs sehingga bisa diubah lewat Setting.
func Defaults() []Network {
	return []Network{
		{Name: "Ethereum Mainnet", Chain: ChainEVM, ChainID: 1, ExplorerURL: "https://etherscan.io", NativeSymbol: "ETH", NativeDecimals: 18},
		{Name: "BNB Chain", Chain: ChainEVM, ChainID: 56, ExplorerURL: "https://bscscan.com", NativeSymbol: "BNB", NativeDecimals: 18},
		{Name: "Arbitrum One", Chain: ChainEVM, ChainID: 42161, ExplorerURL: "https://arbiscan.io", NativeSymbol: "ETH", NativeDecimals: 18},
		{Name: "Robinhood Chain", Chain: ChainEVM, ChainID: 4663, ExplorerURL: "https://robinhoodchain.blockscout.com", NativeSymbol: "ETH", NativeDecimals: 18},
		{Name: "Solana Mainnet", Chain: ChainSolana, ChainID: 0, ExplorerURL: "https://solscan.io", NativeSymbol: "SOL", NativeDecimals: 9},
	}
}

// Canonical memetakan nama jaringan dari DB/env/UI ke kunci internal yang
// stabil. Contoh: "ethereum", "Ethereum Mainnet", "ETH" -> "ethereum".
func Canonical(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.ReplaceAll(n, "-", " ")
	n = strings.ReplaceAll(n, "_", " ")

	switch {
	case strings.Contains(n, "ethereum"), n == "eth":
		return "ethereum"
	case strings.Contains(n, "binance"), strings.Contains(n, "bnb"), strings.Contains(n, "bsc"):
		return "bsc"
	case strings.Contains(n, "arbitrum"):
		return "arbitrum"
	case strings.Contains(n, "robinhood"):
		return "robinhood"
	case strings.Contains(n, "solana"), n == "sol":
		return "solana"
	default:
		return ""
	}
}

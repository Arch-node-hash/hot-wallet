// Package tokenmeta menyimpan metadata token bawaan. Alamat token bukan
// konfigurasi koneksi, jadi aman didefinisikan di source; endpoint RPC/WSS
// tetap berasal dari env + tabel rpc_configs.
package tokenmeta

// Token adalah metadata satu token.
type Token struct {
	Network  string
	Address  string
	Symbol   string
	Name     string
	Decimals int
}

// Defaults mengembalikan token umum yang dipantau otomatis.
func Defaults() []Token {
	return []Token{
		// Ethereum Mainnet
		{"Ethereum Mainnet", "0xdAC17F958D2ee523a2206206994597C13D831ec7", "USDT", "Tether USD", 6},
		{"Ethereum Mainnet", "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", "USDC", "USD Coin", 6},
		{"Ethereum Mainnet", "0x6B175474E89094C44Da98b954EedeAC495271d0F", "DAI", "Dai Stablecoin", 18},
		{"Ethereum Mainnet", "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", "WETH", "Wrapped Ether", 18},

		// BNB Chain
		{"BNB Chain", "0x55d398326f99059fF775485246999027B3197955", "USDT", "Tether USD (BSC)", 18},
		{"BNB Chain", "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", "USDC", "USD Coin (BSC)", 18},
		{"BNB Chain", "0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56", "BUSD", "Binance USD", 18},

		// Arbitrum One
		{"Arbitrum One", "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9", "USDT", "Tether USD (Arbitrum)", 6},
		{"Arbitrum One", "0xaf88d065e77c8cC2239327C5EDb3A432268e5831", "USDC", "USD Coin (Arbitrum)", 6},
		{"Arbitrum One", "0x912CE59144191C1204E64559FE8253a0e49E6548", "ARB", "Arbitrum", 18},

		// Solana Mainnet (mint address)
		{"Solana Mainnet", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "USDT", "Tether USD (Solana)", 6},
		{"Solana Mainnet", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "USDC", "USD Coin (Solana)", 6},
	}
}

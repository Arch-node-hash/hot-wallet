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

// Defaults mengembalikan token yang dipantau otomatis.
//
// Selain saldo native, tiap jaringan memantau satu stablecoin: USDT di
// Ethereum/BNB/Arbitrum/Solana, dan USDG di Robinhood Chain (stablecoin resmi
// rantai tersebut; USDT/USDC belum punya kontrak resmi di sana).
func Defaults() []Token {
	return []Token{
		// Ethereum Mainnet
		{"Ethereum Mainnet", "0xdAC17F958D2ee523a2206206994597C13D831ec7", "USDT", "Tether USD", 6},

		// BNB Chain
		{"BNB Chain", "0x55d398326f99059fF775485246999027B3197955", "USDT", "Tether USD (BSC)", 18},

		// Arbitrum One
		{"Arbitrum One", "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9", "USDT", "Tether USD (Arbitrum)", 6},

		// Robinhood Chain: stablecoin resmi chain ini adalah USDG (Global
		// Dollar). Alamat resmi dari docs.robinhood.com/chain/contracts.
		{"Robinhood Chain", "0x5fc5360D0400a0Fd4f2af552ADD042D716F1d168", "USDG", "Global Dollar", 6},

		// Solana Mainnet (mint address)
		{"Solana Mainnet", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "USDT", "Tether USD (Solana)", 6},
	}
}

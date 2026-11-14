package network

// Tipe alamat yang dipakai UI untuk membedakan kelompok EVM dan Solana.
const (
	AddressTypeEVM    = "EVM Universal Address"
	AddressTypeSolana = "Solana Mainnet Base58 Address"
)

// Meta adalah metadata satu jaringan yang dipakai halaman Wallet: nama tab,
// standar token (badge), dan tipe alamat. Datanya statis karena tidak bergantung
// pada endpoint RPC/WSS.
type Meta struct {
	Key            string
	Name           string
	DisplayName    string
	Short          string
	Chain          Chain
	ChainID        int64
	NativeSymbol   string
	NativeDecimals uint8
	TokenStandard  string
	AddressType    string
	ExplorerURL    string
}

// AllMeta mengembalikan metadata semua jaringan yang didukung, urut sesuai
// urutan tab di UI (Ethereum, BNB, Arbitrum, Robinhood, Solana).
func AllMeta() []Meta {
	return []Meta{
		{
			Key: "ethereum", Name: "Ethereum Mainnet", DisplayName: "Ethereum", Short: "ETH",
			Chain: ChainEVM, ChainID: 1, NativeSymbol: "ETH", NativeDecimals: 18,
			TokenStandard: "ERC-20", AddressType: AddressTypeEVM,
			ExplorerURL: "https://etherscan.io",
		},
		{
			Key: "bsc", Name: "BNB Chain", DisplayName: "BNB Smart Chain", Short: "BNB",
			Chain: ChainEVM, ChainID: 56, NativeSymbol: "BNB", NativeDecimals: 18,
			TokenStandard: "BEP-20", AddressType: AddressTypeEVM,
			ExplorerURL: "https://bscscan.com",
		},
		{
			Key: "arbitrum", Name: "Arbitrum One", DisplayName: "Arbitrum One", Short: "ARB",
			Chain: ChainEVM, ChainID: 42161, NativeSymbol: "ETH", NativeDecimals: 18,
			TokenStandard: "Arbitrum", AddressType: AddressTypeEVM,
			ExplorerURL: "https://arbiscan.io",
		},
		{
			Key: "robinhood", Name: "Robinhood Chain", DisplayName: "Robinhood Chain", Short: "ROBINHOOD",
			Chain: ChainEVM, ChainID: 4663, NativeSymbol: "ETH", NativeDecimals: 18,
			TokenStandard: "RH-20", AddressType: AddressTypeEVM,
			ExplorerURL: "https://robinhoodchain.blockscout.com",
		},
		{
			Key: "solana", Name: "Solana Mainnet", DisplayName: "Solana", Short: "SOL",
			Chain: ChainSolana, ChainID: 0, NativeSymbol: "SOL", NativeDecimals: 9,
			TokenStandard: "SPL", AddressType: AddressTypeSolana,
			ExplorerURL: "https://solscan.io",
		},
	}
}

// MetaFor mencari metadata berdasarkan nama jaringan apa pun (nama panjang,
// key, atau singkatan). Nama yang tidak dikenal mengembalikan ok=false.
func MetaFor(name string) (Meta, bool) {
	key := Canonical(name)
	if key == "" {
		return Meta{}, false
	}
	for _, meta := range AllMeta() {
		if meta.Key == key {
			return meta, true
		}
	}
	return Meta{}, false
}

// EVMKeys mengembalikan key jaringan EVM dalam urutan tab.
func EVMKeys() []string {
	return []string{"ethereum", "bsc", "arbitrum", "robinhood"}
}

package models

import "time"

// APIResponse adalah struktur response standar
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Wallet models
type Wallet struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Address        string    `json:"address"`
	Network        string    `json:"network"`
	DerivationPath string    `json:"derivation_path"`
	Balance        string    `json:"balance"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type WalletBalance struct {
	WalletID     string `json:"wallet_id"`
	Network      string `json:"network"`
	TokenAddress string `json:"token_address,omitempty"`
	TokenSymbol  string `json:"token_symbol"`
	Balance      string `json:"balance"`
	ValueUSD     string `json:"value_usd"`
	UpdatedAt    string `json:"updated_at"`
}

type NetworkSummary struct {
	WalletID      string `json:"wallet_id"`
	Network       string `json:"network"`
	TotalBalance  string `json:"total_balance"`
	TotalValueUSD string `json:"total_value_usd"`
	TokenCount    int    `json:"token_count"`
	UpdatedAt     string `json:"updated_at"`
}

// Portfolio models
type PortfolioSummary struct {
	TotalAssets   string `json:"total_assets"`
	TotalValueUSD string `json:"total_value_usd"`
	NetworkCount  int    `json:"network_count"`
	TokenCount    int    `json:"token_count"`
	UpdatedAt     string `json:"updated_at"`
}

type PortfolioToken struct {
	WalletID     string `json:"wallet_id"`
	Network      string `json:"network"`
	TokenAddress string `json:"token_address,omitempty"`
	TokenSymbol  string `json:"token_symbol"`
	TokenName    string `json:"token_name,omitempty"`
	Balance      string `json:"balance"`
	PriceUSD     string `json:"price_usd"`
	ValueUSD     string `json:"value_usd"`
	UpdatedAt    string `json:"updated_at"`
}

type NetworkAsset struct {
	WalletID      string  `json:"wallet_id"`
	Network       string  `json:"network"`
	TotalTokens   int     `json:"total_tokens"`
	TotalValueUSD string  `json:"total_value_usd"`
	Percentage    float64 `json:"percentage"`
	UpdatedAt     string  `json:"updated_at"`
}

// Transaction models
type Transaction struct {
	ID           string    `json:"id"`
	WalletID     string    `json:"wallet_id"`
	Network      string    `json:"network"`
	TxHash       string    `json:"tx_hash"`
	FromAddress  string    `json:"from_address"`
	ToAddress    string    `json:"to_address"`
	Amount       string    `json:"amount"`
	TokenSymbol  string    `json:"token_symbol,omitempty"`
	TokenAddress string    `json:"token_address,omitempty"`
	Fee          string    `json:"fee,omitempty"`
	Status       string    `json:"status"`
	BlockNumber  *int64    `json:"block_number,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	CreatedAt    time.Time `json:"created_at"`
}

// Contact models
type Contact struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Network   string    `json:"network"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Setting models
type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RPCConfig struct {
	Network     string `json:"network"`
	RPCURL      string `json:"rpc_url"`
	WSSURL      string `json:"wss_url,omitempty"`
	ChainID     int64  `json:"chain_id,omitempty"`
	ExplorerURL string `json:"explorer_url,omitempty"`
	UpdatedAt   string `json:"updated_at"`
}

type SecuritySetting struct {
	PINHash          string `json:"pin_hash,omitempty"`
	BiometricEnabled bool   `json:"biometric_enabled"`
	AutoLockMinutes  int    `json:"auto_lock_minutes"`
	UpdatedAt        string `json:"updated_at"`
}

type BackupHistory struct {
	ID         string    `json:"id"`
	BackupType string    `json:"backup_type"`
	BackupDate string    `json:"backup_date"`
	FilePath   string    `json:"file_path,omitempty"`
	Encrypted  bool      `json:"encrypted"`
	Notes      string    `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// Request models
type CreateWalletRequest struct {
	Name           string `json:"name"`
	Network        string `json:"network"`
	DerivationPath string `json:"derivation_path,omitempty"`
}

type ImportWalletRequest struct {
	Name           string `json:"name"`
	PrivateKey     string `json:"private_key,omitempty"`
	Network        string `json:"network"`
	DerivationPath string `json:"derivation_path,omitempty"`
}

type SendRequest struct {
	WalletID     string `json:"wallet_id"`
	ToAddress    string `json:"to_address"`
	Amount       string `json:"amount"`
	TokenAddress string `json:"token_address,omitempty"`
	Network      string `json:"network"`
	GasLimit     string `json:"gas_limit,omitempty"`
	GasPrice     string `json:"gas_price,omitempty"`
}

type CreateContactRequest struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Network string `json:"network"`
	Notes   string `json:"notes,omitempty"`
}

type UpdateSecurityRequest struct {
	PIN              string `json:"pin,omitempty"`
	BiometricEnabled *bool  `json:"biometric_enabled,omitempty"`
	AutoLockMinutes  *int   `json:"auto_lock_minutes,omitempty"`
}

type BackupRequest struct {
	BackupType string `json:"backup_type"`
	Password   string `json:"password,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

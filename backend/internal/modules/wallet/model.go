package wallet

import (
	"hotwallet/backend/internal/modules/activity"
	"hotwallet/backend/internal/modules/contact"
	"hotwallet/backend/internal/modules/portfolio"
	"hotwallet/backend/internal/modules/transaction"
)

type WeeklyActivity struct {
	Day   string `json:"day"`
	Value int    `json:"value"`
	Label string `json:"label"`
}

type Progress struct {
	Percent    int `json:"percent"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Pending    int `json:"pending"`
}

// RawData adalah struktur data wallet dari koleksi "wallet".
type RawData struct {
	Stats          []portfolio.Stat `json:"stats"`
	WeeklyActivity []WeeklyActivity `json:"weekly_activity"`
	Progress       Progress         `json:"progress"`
}

// User adalah data user minimal untuk wallet
type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	Initials string `json:"initials"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

func (u User) GetID() string    { return u.ID }
func (u *User) SetID(id string) { u.ID = id }

// TimeEntry tetap dipertahankan dari struktur lama.
type TimeEntry struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	TaskID    string `json:"task_id"`
	Seconds   int    `json:"seconds"`
	Running   bool   `json:"running"`
	StartedAt string `json:"started_at"`
}

func (e TimeEntry) GetID() string    { return e.ID }
func (e *TimeEntry) SetID(id string) { e.ID = id }

// Response adalah agregasi data wallet.
type Response struct {
	User           User                  `json:"user"`
	Stats          []portfolio.Stat      `json:"stats"`
	WeeklyActivity []WeeklyActivity      `json:"weekly_activity"`
	Projects       []transaction.Project `json:"projects"`
	Team           []contact.Member      `json:"team"`
	Reminders      []activity.Reminder   `json:"reminders"`
	Progress       Progress              `json:"progress"`
	TimeEntry      TimeEntry             `json:"time_entry"`
}

// ---------------------------------------------------------------------------
// Wallet & saldo
// ---------------------------------------------------------------------------

type CreateWalletRequest struct {
	Name           string `json:"name"`
	Network        string `json:"network"`
	PrivateKey     string `json:"private_key,omitempty"`
	DerivationPath string `json:"derivation_path,omitempty"`
}

type UpdateWalletAddressRequest struct {
	Network        string `json:"network"`
	PrivateKey     string `json:"private_key"`
	DerivationPath string `json:"derivation_path,omitempty"`
}

type WalletResponse struct {
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	Address         string                   `json:"address"`
	Network         string                   `json:"network"`
	NetworkKey      string                   `json:"network_key,omitempty"`
	TokenStandard   string                   `json:"token_standard,omitempty"`
	AddressType     string                   `json:"address_type,omitempty"`
	AddressShort    string                   `json:"address_short,omitempty"`
	DerivationPath  string                   `json:"derivation_path"`
	Balance         string                   `json:"balance"`
	NetworkBalances map[string][]BalanceItem `json:"network_balances"`
	CreatedAt       string                   `json:"created_at"`
	UpdatedAt       string                   `json:"updated_at"`
}

type WalletDetailResponse struct {
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	Address         string                   `json:"address"`
	Network         string                   `json:"network"`
	NetworkKey      string                   `json:"network_key,omitempty"`
	TokenStandard   string                   `json:"token_standard,omitempty"`
	AddressType     string                   `json:"address_type,omitempty"`
	AddressShort    string                   `json:"address_short,omitempty"`
	NativeSymbol    string                   `json:"native_symbol,omitempty"`
	NativeBalance   string                   `json:"native_balance,omitempty"`
	USDTBalance     string                   `json:"usdt_balance,omitempty"`
	TotalValueUSD   string                   `json:"total_value_usd"`
	TotalValueIDR   string                   `json:"total_value_idr"`
	DerivationPath  string                   `json:"derivation_path"`
	Balance         string                   `json:"balance"`
	NetworkBalances map[string][]BalanceItem `json:"network_balances"`
	CreatedAt       string                   `json:"created_at"`
	UpdatedAt       string                   `json:"updated_at"`
}

type BalanceItem struct {
	TokenAddress string  `json:"token_address,omitempty"`
	TokenSymbol  string  `json:"token_symbol"`
	Name         string  `json:"name,omitempty"`
	Decimals     int     `json:"decimals,omitempty"`
	Balance      string  `json:"balance"`
	PriceUSD     string  `json:"price_usd"`
	ValueUSD     string  `json:"value_usd"`
	ValueIDR     string  `json:"value_idr"`
	Change24h    float64 `json:"change_24h"`
	IsNative     bool    `json:"is_native"`
}

// ---------------------------------------------------------------------------
// Halaman Wallet (overview, alamat, rotate, send)
// ---------------------------------------------------------------------------

// TokenBalanceView adalah satu baris token pada kartu jaringan.
type TokenBalanceView struct {
	TokenAddress string  `json:"token_address"`
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name,omitempty"`
	Decimals     int     `json:"decimals"`
	Balance      string  `json:"balance"`
	PriceUSD     string  `json:"price_usd"`
	ValueUSD     string  `json:"value_usd"`
	ValueIDR     string  `json:"value_idr"`
	Change24h    float64 `json:"change_24h"`
	IsNative     bool    `json:"is_native"`
}

// NetworkCard adalah data satu kartu jaringan pada tab ALL / tab jaringan.
type NetworkCard struct {
	NetworkKey    string             `json:"network_key"`
	Network       string             `json:"network"`
	DisplayName   string             `json:"display_name"`
	Short         string             `json:"short"`
	TokenStandard string             `json:"token_standard"`
	AddressType   string             `json:"address_type"`
	ExplorerURL   string             `json:"explorer_url,omitempty"`
	WalletID      string             `json:"wallet_id,omitempty"`
	Address       string             `json:"address,omitempty"`
	AddressShort  string             `json:"address_short,omitempty"`
	NativeSymbol  string             `json:"native_symbol"`
	NativeBalance string             `json:"native_balance"`
	USDTBalance   string             `json:"usdt_balance"`
	TotalValueUSD string             `json:"total_value_usd"`
	TotalValueIDR string             `json:"total_value_idr"`
	Change24h     float64            `json:"change_24h_percent"`
	HasWallet     bool               `json:"has_wallet"`
	Tokens        []TokenBalanceView `json:"tokens"`
}

// OverviewResponse adalah payload tab ALL.
type OverviewResponse struct {
	TotalValueUSD    string        `json:"total_value_usd"`
	TotalValueIDR    string        `json:"total_value_idr"`
	Change24hUSD     string        `json:"change_24h_usd"`
	Change24hPercent float64       `json:"change_24h_percent"`
	USDIDRRate       string        `json:"usd_idr_rate"`
	FXUpdatedAt      string        `json:"fx_updated_at,omitempty"`
	RefreshedAt      string        `json:"refreshed_at"`
	Networks         []NetworkCard `json:"networks"`
}

// AddressGroup adalah satu kolom pada modal alamat (EVM universal / Solana).
type AddressGroup struct {
	AddressType  string   `json:"address_type"`
	Address      string   `json:"address"`
	AddressShort string   `json:"address_short"`
	Networks     []string `json:"networks"`
	Shorts       []string `json:"shorts"`
	QRData       string   `json:"qr_data"`
}

// AddressesResponse adalah payload modal alamat (2 kolom).
type AddressesResponse struct {
	EVM    *AddressGroup `json:"evm"`
	Solana *AddressGroup `json:"solana"`
}

// NetworkTab adalah metadata tab di halaman Wallet.
type NetworkTab struct {
	NetworkKey    string `json:"network_key"`
	Network       string `json:"network"`
	DisplayName   string `json:"display_name"`
	Short         string `json:"short"`
	TokenStandard string `json:"token_standard"`
	AddressType   string `json:"address_type"`
	WalletID      string `json:"wallet_id,omitempty"`
	Address       string `json:"address,omitempty"`
	AddressShort  string `json:"address_short,omitempty"`
	HasWallet     bool   `json:"has_wallet"`
}

// RotatedWallet adalah wallet yang alamatnya ikut diperbarui saat rotate.
type RotatedWallet struct {
	WalletID        string `json:"wallet_id"`
	Network         string `json:"network"`
	OldAddress      string `json:"old_address"`
	NewAddress      string `json:"new_address"`
	DerivationIndex int    `json:"derivation_index"`
}

// RotateResponse adalah hasil "Perbaharui Address".
type RotateResponse struct {
	Chain           string          `json:"chain"`
	NetworkKey      string          `json:"network_key"`
	AddressType     string          `json:"address_type"`
	OldAddress      string          `json:"old_address"`
	NewAddress      string          `json:"new_address"`
	DerivationIndex int             `json:"derivation_index"`
	Note            string          `json:"note"`
	Wallets         []RotatedWallet `json:"wallets"`
}

// SendEstimateRequest adalah parameter estimasi biaya kirim.
type SendEstimateRequest struct {
	To           string `json:"to"`
	TokenAddress string `json:"token_address"`
	Amount       string `json:"amount"`
}

// SendEstimateResponse adalah hasil estimasi biaya.
type SendEstimateResponse struct {
	Network      string `json:"network"`
	NetworkKey   string `json:"network_key"`
	TokenSymbol  string `json:"token_symbol"`
	TokenAddress string `json:"token_address"`
	Amount       string `json:"amount"`
	NativeSymbol string `json:"native_symbol"`
	FeeNative    string `json:"fee_native"`
	FeeUSD       string `json:"fee_usd"`
	GasLimit     uint64 `json:"gas_limit,omitempty"`
	GasPrice     string `json:"gas_price,omitempty"`
	TotalUSD     string `json:"total_usd"`
}

// SendRequest adalah parameter kirim (langkah konfirmasi).
type SendRequest struct {
	To           string `json:"to"`
	TokenAddress string `json:"token_address"`
	Amount       string `json:"amount"`
	Pin          string `json:"pin,omitempty"`
}

// SendResponse adalah hasil broadcast transaksi.
type SendResponse struct {
	TxHash      string `json:"tx_hash"`
	Network     string `json:"network"`
	NetworkKey  string `json:"network_key"`
	TokenSymbol string `json:"token_symbol"`
	Amount      string `json:"amount"`
	To          string `json:"to"`
	ExplorerURL string `json:"explorer_url,omitempty"`
	Status      string `json:"status"`
}

// ---------------------------------------------------------------------------
// Diagnosa kesiapan kirim
// ---------------------------------------------------------------------------

type DiagnosticsResponse struct {
	CheckedAt string               `json:"checked_at"`
	Ready     int                  `json:"ready"`
	Total     int                  `json:"total"`
	Networks  []NetworkDiagnostics `json:"networks"`
}

// NetworkDiagnostics adalah status satu jaringan.
type NetworkDiagnostics struct {
	NetworkKey      string             `json:"network_key"`
	Network         string             `json:"network"`
	Transport       string             `json:"transport"`
	Address         string             `json:"address,omitempty"`
	Reachable       bool               `json:"reachable"`
	ChainID         int64              `json:"chain_id,omitempty"`
	ChainIDExpected int64              `json:"chain_id_expected,omitempty"`
	ChainIDOK       bool               `json:"chain_id_ok"`
	NativeSymbol    string             `json:"native_symbol"`
	NativeBalance   string             `json:"native_balance,omitempty"`
	Nonce           uint64             `json:"nonce,omitempty"`
	GasPriceWei     string             `json:"gas_price_wei,omitempty"`
	GasPriceGwei    string             `json:"gas_price_gwei,omitempty"`
	GasFeeNative    string             `json:"gas_fee_native,omitempty"`
	GasReady        bool               `json:"gas_ready"`
	Blockhash       string             `json:"blockhash,omitempty"`
	Version         string             `json:"version,omitempty"`
	Tokens          []TokenDiagnostics `json:"tokens"`
	Error           string             `json:"error,omitempty"`
}

// TokenDiagnostics adalah status satu kontrak token.
type TokenDiagnostics struct {
	Symbol   string `json:"symbol"`
	Address  string `json:"address"`
	Decimals int    `json:"decimals,omitempty"`
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
}

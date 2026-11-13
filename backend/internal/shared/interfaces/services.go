package interfaces

import (
	"context"

	"hotwallet/backend/internal/shared/models"
)

// WalletService interface untuk modul Wallet
type WalletService interface {
	// Daftar wallet
	ListWallets(ctx context.Context) ([]models.Wallet, error)
	
	// Detail wallet
	GetWallet(ctx context.Context, walletID string) (*models.Wallet, error)
	
	// Buat wallet baru
	CreateWallet(ctx context.Context, req models.CreateWalletRequest) (*models.Wallet, error)
	
	// Import wallet
	ImportWallet(ctx context.Context, req models.ImportWalletRequest) (*models.Wallet, error)
	
	// Hapus wallet
	DeleteWallet(ctx context.Context, walletID string) error
	
	// Get balance wallet per jaringan
	GetWalletBalance(ctx context.Context, walletID string) ([]models.WalletBalance, error)
	
	// Get network summary
	GetNetworkSummary(ctx context.Context, walletID string) ([]models.NetworkSummary, error)
	
	// Send transaction
	Send(ctx context.Context, req models.SendRequest) (string, error) // returns tx hash
	
	// Generate receive address
	GetReceiveAddress(ctx context.Context, walletID string) (string, error)
}

// PortfolioService interface untuk modul Portfolio
type PortfolioService interface {
	// Get total portfolio summary
	GetSummary(ctx context.Context) (*models.PortfolioSummary, error)
	
	// Get all tokens across all wallets
	GetTokens(ctx context.Context) ([]models.PortfolioToken, error)
	
	// Get assets breakdown per network
	GetNetworkAssets(ctx context.Context) ([]models.NetworkAsset, error)
	
	// Refresh portfolio data (fetch from blockchain)
	RefreshPortfolio(ctx context.Context) error
}

// TransactionService interface untuk modul Transaction
type TransactionService interface {
	// Get transaction history
	ListTransactions(ctx context.Context, walletID string) ([]models.Transaction, error)
	
	// Get transaction detail
	GetTransaction(ctx context.Context, txHash string) (*models.Transaction, error)
	
	// Get transaction status
	GetTransactionStatus(ctx context.Context, txHash string) (string, error)
	
	// Sync transactions from blockchain
	SyncTransactions(ctx context.Context, walletID string) error
}

// ContactService interface untuk modul Contact
type ContactService interface {
	// List contacts
	ListContacts(ctx context.Context) ([]models.Contact, error)
	
	// Get contact detail
	GetContact(ctx context.Context, contactID string) (*models.Contact, error)
	
	// Create contact
	CreateContact(ctx context.Context, req models.CreateContactRequest) (*models.Contact, error)
	
	// Update contact
	UpdateContact(ctx context.Context, contactID string, req models.CreateContactRequest) (*models.Contact, error)
	
	// Delete contact
	DeleteContact(ctx context.Context, contactID string) error
	
	// Search contacts
	SearchContacts(ctx context.Context, query string) ([]models.Contact, error)
}

// SettingService interface untuk modul Setting
type SettingService interface {
	// Get all settings
	GetSettings(ctx context.Context) ([]models.Setting, error)
	
	// Get specific setting
	GetSetting(ctx context.Context, key string) (*models.Setting, error)
	
	// Update setting
	UpdateSetting(ctx context.Context, key, value string) error
	
	// RPC Configuration
	ListRPCConfigs(ctx context.Context) ([]models.RPCConfig, error)
	UpdateRPCConfig(ctx context.Context, config models.RPCConfig) error
	
	// Security settings
	GetSecuritySettings(ctx context.Context) (*models.SecuritySetting, error)
	UpdateSecuritySettings(ctx context.Context, req models.UpdateSecurityRequest) error
	VerifyPIN(ctx context.Context, pin string) (bool, error)
	
	// Backup
	GetBackupHistory(ctx context.Context) ([]models.BackupHistory, error)
	CreateBackup(ctx context.Context, req models.BackupRequest) (*models.BackupHistory, error)
	RestoreBackup(ctx context.Context, backupID string, password string) error
}

// BlockchainService interface untuk komunikasi dengan blockchain
type BlockchainService interface {
	// Get balance for address
	GetBalance(ctx context.Context, network, address string) (string, error)
	
	// Get token balance
	GetTokenBalance(ctx context.Context, network, address, tokenAddress string) (string, error)
	
	// Send transaction
	SendTransaction(ctx context.Context, network string, fromPrivateKey string, toAddress string, amount string) (string, error)
	
	// Get transaction status
	GetTransactionStatus(ctx context.Context, network, txHash string) (string, error)
	
	// Get gas price
	GetGasPrice(ctx context.Context, network string) (string, error)
	
	// Estimate gas
	EstimateGas(ctx context.Context, network, from, to, amount string) (string, error)
}

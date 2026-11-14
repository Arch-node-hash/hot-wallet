package wallet

import (
	"context"
	"fmt"
	"strings"

	"hotwallet/backend/config"
	"hotwallet/backend/internal/blockchain/network"
	"hotwallet/backend/internal/blockchain/wallet"
	"hotwallet/backend/internal/infrastructure/store"
)

// Service mengelola business logic wallet
type Service struct {
	store *store.Store
	cfg   config.Config
}

func NewService(st *store.Store, cfg config.Config) *Service {
	return &Service{store: st, cfg: cfg}
}

// AutoCreateFromEnv membuat wallet otomatis dari private key di env
func (s *Service) AutoCreateFromEnv(ctx context.Context) error {
	return s.store.AutoCreateWalletFromEnv(ctx, s.cfg)
}

// ListWallets mengembalikan daftar semua wallet dengan saldo per jaringan
func (s *Service) ListWallets(ctx context.Context) ([]WalletResponse, error) {
	rows, err := s.store.ListWallets(ctx)
	if err != nil {
		return nil, err
	}

	var resp []WalletResponse
	for _, row := range rows {
		items, err := s.balanceItems(ctx, row)
		if err != nil {
			return nil, err
		}

		meta, _ := network.MetaFor(row.Network)
		resp = append(resp, WalletResponse{
			ID:              row.ID,
			Name:            row.Name,
			Address:         row.Address,
			Network:         row.Network,
			NetworkKey:      meta.Key,
			TokenStandard:   meta.TokenStandard,
			AddressType:     meta.AddressType,
			AddressShort:    shortAddress(row.Address),
			DerivationPath:  row.DerivationPath,
			Balance:         row.Balance,
			NetworkBalances: map[string][]BalanceItem{row.Network: items},
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		})
	}

	return resp, nil
}

// GetWallet mengembalikan detail wallet termasuk saldo per token per jaringan
func (s *Service) GetWallet(ctx context.Context, id string) (*WalletDetailResponse, error) {
	row, err := s.store.GetWallet(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, fmt.Errorf("wallet not found")
	}

	items, err := s.balanceItems(ctx, *row)
	if err != nil {
		return nil, err
	}

	meta, _ := network.MetaFor(row.Network)
	total := "0"
	native := "0"
	usdt := "0"
	for _, item := range items {
		total = addUSD(total, item.ValueUSD)
		if item.IsNative {
			native = item.Balance
		}
		if strings.EqualFold(item.TokenSymbol, "USDT") {
			usdt = item.Balance
		}
	}

	return &WalletDetailResponse{
		ID:              row.ID,
		Name:            row.Name,
		Address:         row.Address,
		Network:         row.Network,
		NetworkKey:      meta.Key,
		TokenStandard:   meta.TokenStandard,
		AddressType:     meta.AddressType,
		AddressShort:    shortAddress(row.Address),
		NativeSymbol:    meta.NativeSymbol,
		NativeBalance:   native,
		USDTBalance:     usdt,
		TotalValueUSD:   total,
		DerivationPath:  row.DerivationPath,
		Balance:         row.Balance,
		NetworkBalances: map[string][]BalanceItem{row.Network: items},
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

// balanceItems membaca saldo wallet dan menghitung nilai USD-nya memakai harga
// dari tabel token_prices (stablecoin fallback $1).
func (s *Service) balanceItems(ctx context.Context, row store.WalletRow) ([]BalanceItem, error) {
	balances, err := s.store.GetWalletBalances(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	prices, err := s.store.ListTokenPrices(ctx)
	if err != nil {
		return nil, err
	}
	tokens, err := s.store.ListTokens()
	if err != nil {
		return nil, err
	}

	idx := newPriceIndex(prices)
	meta, _ := network.MetaFor(row.Network)

	items := make([]BalanceItem, 0, len(balances))
	for _, b := range balances {
		price, change := idx.lookup(row.Network, b.TokenAddress, b.TokenSymbol)
		items = append(items, BalanceItem{
			TokenAddress: b.TokenAddress,
			TokenSymbol:  b.TokenSymbol,
			Name:         tokenName(tokens, row.Network, b.TokenAddress),
			Decimals:     tokenDecimals(tokens, row.Network, b.TokenAddress, meta.NativeDecimals),
			Balance:      b.Balance,
			PriceUSD:     price,
			ValueUSD:     mulUSD(b.Balance, price),
			Change24h:    change,
			IsNative:     b.TokenAddress == "",
		})
	}
	return items, nil
}

// CreateWallet membuat wallet baru dari private key atau generate baru
func (s *Service) CreateWallet(ctx context.Context, req CreateWalletRequest) (*WalletResponse, error) {
	var address, network, derivationPath string
	var err error

	if req.PrivateKey != "" {
		// Import dari private key
		switch req.Network {
		case "Ethereum Mainnet", "BNB Chain", "Arbitrum One", "Robinhood Chain":
			info, err := wallet.ParseEVMPrivateKey(req.PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("invalid EVM private key: %w", err)
			}
			address = info.Address
			network = req.Network
			derivationPath = req.DerivationPath
			if derivationPath == "" {
				derivationPath = "m/44'/60'/0'/0/0"
			}
		case "Solana Mainnet":
			info, err := wallet.ParseSolanaPrivateKey(req.PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("invalid Solana private key: %w", err)
			}
			address = info.Address
			network = req.Network
			derivationPath = req.DerivationPath
			if derivationPath == "" {
				derivationPath = "m/44'/501'/0'/0'"
			}
		default:
			return nil, fmt.Errorf("unsupported network: %s", req.Network)
		}
	} else {
		// Generate wallet baru dari seed phrase (belum diimplementasikan)
		return nil, fmt.Errorf("generate from seed not implemented yet, provide private key")
	}

	// Cek apakah address sudah ada di network yang sama. Satu private key EVM
	// dipakai untuk semua jaringan EVM, jadi address yang sama memang muncul
	// di beberapa wallet (satu per jaringan).
	existing, err := s.store.GetWalletByAddressAndNetwork(ctx, address, network)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("wallet with address %s already exists on %s", address, network)
	}

	// Generate ID
	id := generateWalletID(network, address)

	walletRow := &store.WalletRow{
		ID:             id,
		Name:           req.Name,
		Address:        address,
		Network:        network,
		DerivationPath: derivationPath,
		Balance:        "0",
	}

	if err := s.store.CreateWallet(ctx, walletRow); err != nil {
		return nil, err
	}

	return &WalletResponse{
		ID:              walletRow.ID,
		Name:            walletRow.Name,
		Address:         walletRow.Address,
		Network:         walletRow.Network,
		DerivationPath:  walletRow.DerivationPath,
		Balance:         walletRow.Balance,
		NetworkBalances: make(map[string][]BalanceItem),
		CreatedAt:       walletRow.CreatedAt,
		UpdatedAt:       walletRow.UpdatedAt,
	}, nil
}

// UpdateWalletAddress memperbarui alamat wallet (ganti private key)
func (s *Service) UpdateWalletAddress(ctx context.Context, id string, req UpdateWalletAddressRequest) (*WalletDetailResponse, error) {
	current, err := s.store.GetWallet(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("wallet not found")
	}

	if req.Network != "" && req.Network != current.Network {
		return nil, fmt.Errorf("network mismatch: wallet ini %s, bukan %s", current.Network, req.Network)
	}
	network := current.Network

	// Validasi private key baru dan turunkan alamatnya
	var address, derivationPath string
	switch network {
	case "Ethereum Mainnet", "BNB Chain", "Arbitrum One", "Robinhood Chain":
		info, err := wallet.ParseEVMPrivateKey(req.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid EVM private key: %w", err)
		}
		address = info.Address
		derivationPath = req.DerivationPath
		if derivationPath == "" {
			derivationPath = "m/44'/60'/0'/0/0"
		}
	case "Solana Mainnet":
		info, err := wallet.ParseSolanaPrivateKey(req.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid Solana private key: %w", err)
		}
		address = info.Address
		derivationPath = req.DerivationPath
		if derivationPath == "" {
			derivationPath = "m/44'/501'/0'/0'"
		}
	default:
		return nil, fmt.Errorf("unsupported network: %s", network)
	}

	// Cek apakah address baru sudah dipakai wallet lain di network yang sama.
	existing, err := s.store.GetWalletByAddressAndNetwork(ctx, address, network)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID != id {
		return nil, fmt.Errorf("address %s already used by another wallet on %s", address, network)
	}

	// Update wallet. Endpoint ini mengganti alamat dengan alamat dari private
	// key yang dikirim client, jadi selalu index 0.
	if err := s.store.UpdateWalletAddress(ctx, id, address, network, derivationPath, 0); err != nil {
		return nil, err
	}

	// Ambil wallet yang sudah diupdate
	return s.GetWallet(ctx, id)
}

// DeleteWallet menghapus wallet
func (s *Service) DeleteWallet(ctx context.Context, id string) error {
	return s.store.DeleteWallet(ctx, id)
}

// GenerateWalletID membuat ID wallet yang konsisten dengan wallet yang dibuat
// otomatis dari env: satu prefix per jaringan, alamat EVM tanpa 0x.
func generateWalletID(network, address string) string {
	prefix := "eth"
	switch network {
	case "BNB Chain":
		prefix = "bsc"
	case "Arbitrum One":
		prefix = "arb"
	case "Robinhood Chain":
		prefix = "rob"
	case "Solana Mainnet":
		prefix = "sol"
		if len(address) > 8 {
			address = address[:8]
		}
		return prefix + "-" + address
	}

	address = strings.TrimPrefix(address, "0x")
	if len(address) > 8 {
		address = address[:8]
	}
	return prefix + "-" + address
}

// Request/Response types
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
	Change24h    float64 `json:"change_24h"`
	IsNative     bool    `json:"is_native"`
}

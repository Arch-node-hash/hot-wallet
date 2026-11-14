package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"hotwallet/backend/config"
	"hotwallet/backend/internal/blockchain/wallet"
)

// WalletRow mewakili satu baris wallet dari database
type WalletRow struct {
	ID             string
	Name           string
	Address        string
	Network        string
	DerivationPath string
	Balance        string
	CreatedAt      string
	UpdatedAt      string
}

// ListWallets mengambil semua wallet
func (s *Store) ListWallets(ctx context.Context) ([]WalletRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, address, network, derivation_path, balance, created_at, updated_at
		FROM wallets
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("store: list wallets: %w", err)
	}
	defer rows.Close()

	var wallets []WalletRow
	for rows.Next() {
		var w WalletRow
		if err := rows.Scan(&w.ID, &w.Name, &w.Address, &w.Network, &w.DerivationPath, &w.Balance, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan wallet: %w", err)
		}
		wallets = append(wallets, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: rows wallet: %w", err)
	}
	return wallets, nil
}

// GetWallet mengambil wallet berdasarkan ID
func (s *Store) GetWallet(ctx context.Context, id string) (*WalletRow, error) {
	var w WalletRow
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, address, network, derivation_path, balance, created_at, updated_at
		FROM wallets
		WHERE id = ?
	`, id).Scan(&w.ID, &w.Name, &w.Address, &w.Network, &w.DerivationPath, &w.Balance, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: get wallet %s: %w", id, err)
	}
	return &w, nil
}

// GetWalletByAddressAndNetwork mengambil wallet berdasarkan alamat DAN network
func (s *Store) GetWalletByAddressAndNetwork(ctx context.Context, address, network string) (*WalletRow, error) {
	var w WalletRow
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, address, network, derivation_path, balance, created_at, updated_at
		FROM wallets
		WHERE address = ? AND network = ?
	`, address, network).Scan(&w.ID, &w.Name, &w.Address, &w.Network, &w.DerivationPath, &w.Balance, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: get wallet by address %s network %s: %w", address, network, err)
	}
	return &w, nil
}

// GetWalletByAddress mengambil wallet berdasarkan alamat saja (legacy - return first match)
func (s *Store) GetWalletByAddress(ctx context.Context, address string) (*WalletRow, error) {
	var w WalletRow
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, address, network, derivation_path, balance, created_at, updated_at
		FROM wallets
		WHERE address = ?
		LIMIT 1
	`, address).Scan(&w.ID, &w.Name, &w.Address, &w.Network, &w.DerivationPath, &w.Balance, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: get wallet by address %s: %w", address, err)
	}
	return &w, nil
}

// CreateWallet membuat wallet baru
func (s *Store) CreateWallet(ctx context.Context, w *WalletRow) error {
	if w.ID == "" {
		return fmt.Errorf("store: wallet ID cannot be empty")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	w.CreatedAt = now
	w.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO wallets (id, name, address, network, derivation_path, balance, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, w.ID, w.Name, w.Address, w.Network, w.DerivationPath, w.Balance, w.CreatedAt, w.UpdatedAt)
	if err != nil {
		return fmt.Errorf("store: create wallet: %w", err)
	}
	return nil
}

// UpdateWallet memperbarui wallet
func (s *Store) UpdateWallet(ctx context.Context, w *WalletRow) error {
	now := time.Now().UTC().Format(time.RFC3339)
	w.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		UPDATE wallets SET
			name = ?,
			address = ?,
			network = ?,
			derivation_path = ?,
			balance = ?,
			updated_at = ?
		WHERE id = ?
	`, w.Name, w.Address, w.Network, w.DerivationPath, w.Balance, w.UpdatedAt, w.ID)
	if err != nil {
		return fmt.Errorf("store: update wallet %s: %w", w.ID, err)
	}
	return nil
}

// DeleteWallet menghapus wallet (cascade ke balances, summaries, transactions)
func (s *Store) DeleteWallet(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM wallets WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete wallet %s: %w", id, err)
	}
	return nil
}

// AutoCreateWalletFromEnv membuat wallet otomatis dari private key di env.
// Satu ETH_PRIVATE_KEY dipakai untuk semua jaringan EVM (alamat sama, satu
// baris wallet per jaringan). Hanya membuat wallet yang belum ada.
func (s *Store) AutoCreateWalletFromEnv(ctx context.Context, cfg config.Config) error {
	var errs []error

	// Cek ETH_PRIVATE_KEY - 1 private key untuk SEMUA EVM chain (address sama)
	if cfg.ETHPrivateKey != "" {
		info, err := wallet.ParseEVMPrivateKey(cfg.ETHPrivateKey)
		if err != nil {
			errs = append(errs, fmt.Errorf("ETH_PRIVATE_KEY tidak valid: %w", err))
		} else {
			evmNetworks := []struct {
				Name           string
				DisplayName    string
				DerivationPath string
				IDPrefix       string
			}{
				{"Ethereum Mainnet", "Ethereum Wallet", "m/44'/60'/0'/0/0", "eth"},
				{"BNB Chain", "BNB Chain Wallet", "m/44'/60'/0'/0/0", "bsc"},
				{"Arbitrum One", "Arbitrum Wallet", "m/44'/60'/0'/0/0", "arb"},
				{"Robinhood Chain", "Robinhood Wallet", "m/44'/60'/0'/0/0", "rob"},
			}

			for _, net := range evmNetworks {
				// Cek apakah sudah ada wallet untuk network ini dengan address ini
				existing, _ := s.GetWalletByAddressAndNetwork(ctx, info.Address, net.Name)
				if existing == nil {
					walletRow := &WalletRow{
						ID:             net.IDPrefix + "-" + info.Address[2:10],
						Name:           net.DisplayName,
						Address:        info.Address,
						Network:        net.Name,
						DerivationPath: net.DerivationPath,
						Balance:        "0",
					}
					if err := s.CreateWallet(ctx, walletRow); err != nil {
						errs = append(errs, fmt.Errorf("auto create %s wallet: %w", net.Name, err))
					}
				}
			}
		}
	}

	// Cek SOLANA_PRIVATE_KEY
	if cfg.SolanaPrivateKey != "" {
		info, err := wallet.ParseSolanaPrivateKey(cfg.SolanaPrivateKey)
		if err != nil {
			errs = append(errs, fmt.Errorf("SOLANA_PRIVATE_KEY tidak valid: %w", err))
		} else {
			// Cek apakah sudah ada wallet Solana
			existing, _ := s.GetWalletByAddressAndNetwork(ctx, info.Address, "Solana Mainnet")
			if existing == nil {
				walletRow := &WalletRow{
					ID:             "sol-" + info.Address[:8],
					Name:           "Solana Wallet",
					Address:        info.Address,
					Network:        "Solana Mainnet",
					DerivationPath: "m/44'/501'/0'/0'",
					Balance:        "0",
				}
				if err := s.CreateWallet(ctx, walletRow); err != nil {
					errs = append(errs, fmt.Errorf("auto create Solana wallet: %w", err))
				}
			}
		}
	}

	return errors.Join(errs...)
}

// UpdateWalletAddress memperbarui alamat wallet (saat user ganti private key)
func (s *Store) UpdateWalletAddress(ctx context.Context, id, newAddress, newNetwork, newDerivationPath string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		UPDATE wallets SET
			address = ?,
			network = ?,
			derivation_path = ?,
			balance = '0',
			updated_at = ?
		WHERE id = ?
	`, newAddress, newNetwork, newDerivationPath, now, id)
	if err != nil {
		return fmt.Errorf("store: update wallet address %s: %w", id, err)
	}
	return nil
}

// UpdateWalletBalance memperbarui saldo native wallet
func (s *Store) UpdateWalletBalance(ctx context.Context, id, balance string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		UPDATE wallets SET balance = ?, updated_at = ? WHERE id = ?
	`, balance, now, id)
	if err != nil {
		return fmt.Errorf("store: update wallet balance %s: %w", id, err)
	}
	return nil
}

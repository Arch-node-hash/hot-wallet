package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
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

// envWalletSpec adalah definisi wallet yang dikelola dari private key env.
type envWalletSpec struct {
	Network        string
	DisplayName    string
	DerivationPath string
	IDPrefix       string
}

func evmEnvWallets() []envWalletSpec {
	return []envWalletSpec{
		{"Ethereum Mainnet", "Ethereum Wallet", "m/44'/60'/0'/0/0", "eth"},
		{"BNB Chain", "BNB Chain Wallet", "m/44'/60'/0'/0/0", "bsc"},
		{"Arbitrum One", "Arbitrum Wallet", "m/44'/60'/0'/0/0", "arb"},
		{"Robinhood Chain", "Robinhood Wallet", "m/44'/60'/0'/0/0", "rob"},
	}
}

func solanaEnvWallet() envWalletSpec {
	return envWalletSpec{"Solana Mainnet", "Solana Wallet", "m/44'/501'/0'/0'", "sol"}
}

// AutoCreateWalletFromEnv menyinkronkan wallet dengan private key di env.
// Satu ETH_PRIVATE_KEY dipakai untuk semua jaringan EVM. Kalau private key
// diganti, alamat wallet di jaringan tersebut ikut diperbarui.
func (s *Store) AutoCreateWalletFromEnv(ctx context.Context, cfg config.Config) error {
	var errs []error

	// Cek ETH_PRIVATE_KEY - 1 private key untuk SEMUA EVM chain (address sama)
	if cfg.ETHPrivateKey != "" {
		info, err := wallet.ParseEVMPrivateKey(cfg.ETHPrivateKey)
		if err != nil {
			errs = append(errs, fmt.Errorf("ETH_PRIVATE_KEY tidak valid: %w", err))
		} else {
			for _, spec := range evmEnvWallets() {
				if err := s.syncEnvWallet(ctx, info.Address, spec); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	// Cek SOLANA_PRIVATE_KEY
	if cfg.SolanaPrivateKey != "" {
		info, err := wallet.ParseSolanaPrivateKey(cfg.SolanaPrivateKey)
		if err != nil {
			errs = append(errs, fmt.Errorf("SOLANA_PRIVATE_KEY tidak valid: %w", err))
		} else if err := s.syncEnvWallet(ctx, info.Address, solanaEnvWallet()); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// syncEnvWallet memastikan wallet satu jaringan memakai alamat dari env.
//
// Deteksi perubahan key tidak perlu menyimpan key-nya: alamat adalah sidik
// jari deterministik dari private key. Alamat hasil derivasi dibandingkan
// dengan alamat yang tersimpan; kalau berbeda berarti key berubah.
//
// Aturan update:
//   - belum ada wallet di jaringan itu  -> buat baru
//   - ada tepat satu wallet             -> perbarui alamatnya
//   - ada lebih dari satu wallet        -> jangan sentuh, kembalikan error
//     supaya user memilih sendiri lewat PUT /address
func (s *Store) syncEnvWallet(ctx context.Context, address string, spec envWalletSpec) error {
	wallets, err := s.ListWallets(ctx)
	if err != nil {
		return err
	}

	var others []WalletRow
	for _, w := range wallets {
		if w.Network != spec.Network {
			continue
		}
		if strings.EqualFold(w.Address, address) {
			return nil // private key sama, tidak ada yang berubah
		}
		others = append(others, w)
	}

	switch len(others) {
	case 0:
		row := &WalletRow{
			ID:             spec.IDPrefix + "-" + walletIDSuffix(address),
			Name:           spec.DisplayName,
			Address:        address,
			Network:        spec.Network,
			DerivationPath: spec.DerivationPath,
			Balance:        "0",
		}
		if err := s.CreateWallet(ctx, row); err != nil {
			return fmt.Errorf("auto create %s wallet: %w", spec.Network, err)
		}
		log.Printf("wallet: %s dibuat dari env (%s)", row.ID, address)

	case 1:
		row := others[0]
		if err := s.UpdateWalletAddress(ctx, row.ID, address, spec.Network, spec.DerivationPath); err != nil {
			return fmt.Errorf("sync %s wallet: %w", spec.Network, err)
		}
		log.Printf("wallet: %s (%s) alamat diperbarui ke %s", row.ID, spec.Network, address)

	default:
		return fmt.Errorf(
			"%s punya %d wallet dan tidak ada yang cocok dengan private key env; "+
				"perbarui manual lewat PUT /api/v1/wallets/{id}/address",
			spec.Network, len(others),
		)
	}

	return nil
}

// walletIDSuffix mengambil 8 karakter alamat untuk ID wallet.
func walletIDSuffix(address string) string {
	address = strings.TrimPrefix(address, "0x")
	if len(address) > 8 {
		address = address[:8]
	}
	return address
}

// UpdateWalletAddress memperbarui alamat wallet (saat user ganti private key).
// Saldo lama dihapus karena milik alamat lama; watcher akan mengisi ulang.
func (s *Store) UpdateWalletAddress(ctx context.Context, id, newAddress, newNetwork, newDerivationPath string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: update wallet address %s: %w", id, err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE wallets SET
			address = ?,
			network = ?,
			derivation_path = ?,
			balance = '0',
			updated_at = ?
		WHERE id = ?
	`, newAddress, newNetwork, newDerivationPath, now, id); err != nil {
		return fmt.Errorf("store: update wallet address %s: %w", id, err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM wallet_balances WHERE wallet_id = ?`, id); err != nil {
		return fmt.Errorf("store: reset balances %s: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM network_summaries WHERE wallet_id = ?`, id); err != nil {
		return fmt.Errorf("store: reset summaries %s: %w", id, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit update wallet address %s: %w", id, err)
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

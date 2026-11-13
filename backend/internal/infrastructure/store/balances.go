package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// WalletTarget adalah wallet yang perlu dipantau watcher.
type WalletTarget struct {
	ID      string
	Address string
	Network string
}

// TokenRow adalah metadata token untuk pembacaan saldo.
type TokenRow struct {
	Network  string
	Address  string
	Symbol   string
	Name     string
	Decimals int
}

// BalanceRow adalah satu baris saldo (native memakai TokenAddress kosong).
type BalanceRow struct {
	WalletID     string
	Network      string
	TokenAddress string
	TokenSymbol  string
	Balance      string
	ValueUSD     string
}

// ListWalletTargets mengambil wallet yang punya alamat dan jaringan.
func (s *Store) ListWalletTargets() ([]WalletTarget, error) {
	rows, err := s.db.Query(`
		SELECT id, address, network
		FROM wallets
		WHERE address <> '' AND network <> ''
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("store: list wallet targets: %w", err)
	}
	defer rows.Close()

	var targets []WalletTarget
	for rows.Next() {
		var target WalletTarget
		if err := rows.Scan(&target.ID, &target.Address, &target.Network); err != nil {
			return nil, fmt.Errorf("store: scan wallet target: %w", err)
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: rows wallet target: %w", err)
	}
	return targets, nil
}

// ListTokens mengambil seluruh metadata token.
func (s *Store) ListTokens() ([]TokenRow, error) {
	rows, err := s.db.Query(`
		SELECT network, address, symbol, COALESCE(name, ''), decimals
		FROM tokens
		ORDER BY network, symbol
	`)
	if err != nil {
		return nil, fmt.Errorf("store: list tokens: %w", err)
	}
	defer rows.Close()

	var tokens []TokenRow
	for rows.Next() {
		var token TokenRow
		if err := rows.Scan(&token.Network, &token.Address, &token.Symbol, &token.Name, &token.Decimals); err != nil {
			return nil, fmt.Errorf("store: scan token: %w", err)
		}
		tokens = append(tokens, token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: rows token: %w", err)
	}
	return tokens, nil
}

// SeedTokens menambahkan metadata token tanpa menimpa data yang sudah ada.
func (s *Store) SeedTokens(rows []TokenRow) error {
	if len(rows) == 0 {
		return nil
	}
	for _, token := range rows {
		_, err := s.db.Exec(`
			INSERT OR IGNORE INTO tokens (network, address, symbol, name, decimals)
			VALUES (?, ?, ?, ?, ?)
		`, token.Network, token.Address, token.Symbol, token.Name, token.Decimals)
		if err != nil {
			return fmt.Errorf("store: seed token %s/%s: %w", token.Network, token.Symbol, err)
		}
	}
	return nil
}

// UpsertBalances menyimpan hasil pembacaan WebSocket ke database:
// wallet_balances, wallets.balance (native), dan network_summaries.
func (s *Store) UpsertBalances(ctx context.Context, rows []BalanceRow) error {
	if len(rows) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: balances tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	touched := make(map[string]struct{})

	for _, row := range rows {
		if row.WalletID == "" || row.Network == "" {
			continue
		}

		valueUSD := row.ValueUSD
		if valueUSD == "" {
			valueUSD = "0"
		}

		_, err := tx.Exec(`
			INSERT INTO wallet_balances
				(wallet_id, network, token_address, token_symbol, balance, value_usd, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(wallet_id, network, token_address) DO UPDATE SET
				token_symbol = excluded.token_symbol,
				balance = excluded.balance,
				value_usd = excluded.value_usd,
				updated_at = excluded.updated_at
		`, row.WalletID, row.Network, row.TokenAddress, row.TokenSymbol, row.Balance, valueUSD, now)
		if err != nil {
			return fmt.Errorf("store: upsert balance %s %s: %w", row.WalletID, row.Network, err)
		}

		if row.TokenAddress == "" {
			_, err = tx.Exec(`
				UPDATE wallets SET balance = ?, updated_at = ? WHERE id = ?
			`, row.Balance, now, row.WalletID)
			if err != nil {
				return fmt.Errorf("store: update wallet balance %s: %w", row.WalletID, err)
			}
		}

		touched[row.WalletID+"\x00"+row.Network] = struct{}{}
	}

	for key := range touched {
		var walletID, network string
		parts := splitKey(key)
		walletID, network = parts[0], parts[1]

		var native string
		var tokenCount int
		err := tx.QueryRow(`
			SELECT COALESCE(NULLIF((
				SELECT balance FROM wallet_balances
				WHERE wallet_id = ? AND network = ? AND token_address = ''
			), ''), '0')
		`, walletID, network).Scan(&native)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("store: native balance %s: %w", walletID, err)
		}

		err = tx.QueryRow(`
			SELECT COUNT(*) FROM wallet_balances WHERE wallet_id = ? AND network = ?
		`, walletID, network).Scan(&tokenCount)
		if err != nil {
			return fmt.Errorf("store: count balances %s: %w", walletID, err)
		}

		_, err = tx.Exec(`
			INSERT INTO network_summaries
				(wallet_id, network, total_balance, total_value_usd, token_count, updated_at)
			VALUES (?, ?, ?, '0', ?, ?)
			ON CONFLICT(wallet_id, network) DO UPDATE SET
				total_balance = excluded.total_balance,
				total_value_usd = excluded.total_value_usd,
				token_count = excluded.token_count,
				updated_at = excluded.updated_at
		`, walletID, network, native, tokenCount, now)
		if err != nil {
			return fmt.Errorf("store: upsert summary %s: %w", walletID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit balances: %w", err)
	}
	return nil
}

// GetWalletBalances membaca saldo tersimpan untuk satu wallet.
func (s *Store) GetWalletBalances(ctx context.Context, walletID string) ([]BalanceRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT wallet_id, network, COALESCE(token_address, ''), token_symbol, balance, value_usd
		FROM wallet_balances
		WHERE wallet_id = ?
		ORDER BY network, token_symbol
	`, walletID)
	if err != nil {
		return nil, fmt.Errorf("store: get balances: %w", err)
	}
	defer rows.Close()

	var result []BalanceRow
	for rows.Next() {
		var row BalanceRow
		if err := rows.Scan(&row.WalletID, &row.Network, &row.TokenAddress, &row.TokenSymbol, &row.Balance, &row.ValueUSD); err != nil {
			return nil, fmt.Errorf("store: scan balance: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: rows balance: %w", err)
	}
	return result, nil
}

func splitKey(key string) [2]string {
	for i := 0; i < len(key); i++ {
		if key[i] == 0 {
			return [2]string{key[:i], key[i+1:]}
		}
	}
	return [2]string{key, ""}
}

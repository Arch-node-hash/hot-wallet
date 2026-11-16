package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// FXRate adalah kurs satu pasangan mata uang, mis. USD -> IDR.
type FXRate struct {
	Base      string
	Quote     string
	Rate      string
	Source    string
	UpdatedAt string
}

// GetFXRate membaca kurs tersimpan. Bila belum ada, mengembalikan rate "0"
// tanpa error supaya pemanggil tidak perlu menangani kasus kosong.
func (s *Store) GetFXRate(ctx context.Context, base, quote string) (FXRate, error) {
	base = strings.ToUpper(strings.TrimSpace(base))
	quote = strings.ToUpper(strings.TrimSpace(quote))

	var rate FXRate
	err := s.db.QueryRowContext(ctx, `
		SELECT base, quote, rate, COALESCE(source, ''), COALESCE(updated_at, '')
		FROM fx_rates
		WHERE base = ? AND quote = ?
	`, base, quote).Scan(&rate.Base, &rate.Quote, &rate.Rate, &rate.Source, &rate.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return FXRate{Base: base, Quote: quote, Rate: "0"}, nil
	}
	if err != nil {
		return FXRate{}, fmt.Errorf("store: get fx rate %s/%s: %w", base, quote, err)
	}
	return rate, nil
}

// SaveFXRate menyimpan kurs terbaru (upsert).
func (s *Store) SaveFXRate(ctx context.Context, rate FXRate) error {
	base := strings.ToUpper(strings.TrimSpace(rate.Base))
	quote := strings.ToUpper(strings.TrimSpace(rate.Quote))
	if base == "" || quote == "" {
		return errors.New("store: fx rate tanpa base/quote")
	}
	if strings.TrimSpace(rate.Rate) == "" {
		return errors.New("store: fx rate kosong")
	}

	updatedAt := rate.UpdatedAt
	if updatedAt == "" {
		updatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO fx_rates (base, quote, rate, source, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(base, quote) DO UPDATE SET
			rate = excluded.rate,
			source = excluded.source,
			updated_at = excluded.updated_at
	`, base, quote, rate.Rate, rate.Source, updatedAt)
	if err != nil {
		return fmt.Errorf("store: save fx rate %s/%s: %w", base, quote, err)
	}
	return nil
}

// RevalueBalances menghitung ulang value_idr semua saldo dari value_usd ×
// kurs, lalu memperbarui total_value_idr di network_summaries. Dipanggil
// setiap kali kurs berubah.
func (s *Store) RevalueBalances(ctx context.Context, rate string) error {
	rate = strings.TrimSpace(rate)
	if rate == "" {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: revalue tx: %w", err)
	}
	defer tx.Rollback()

	type balance struct {
		walletID     string
		network      string
		tokenAddress string
	}
	type update struct {
		key   balance
		value string
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT wallet_id, network, COALESCE(token_address, ''), COALESCE(value_usd, '0')
		FROM wallet_balances
	`)
	if err != nil {
		return fmt.Errorf("store: revalue read: %w", err)
	}

	var updates []update
	totals := make(map[balance]string)
	for rows.Next() {
		var (
			walletID     string
			network      string
			tokenAddress string
			valueUSD     string
		)
		if err := rows.Scan(&walletID, &network, &tokenAddress, &valueUSD); err != nil {
			rows.Close()
			return fmt.Errorf("store: revalue scan: %w", err)
		}
		key := balance{walletID: walletID, network: network}
		valueIDR := IDRValue(valueUSD, rate)
		updates = append(updates, update{
			key:   balance{walletID: walletID, network: network, tokenAddress: tokenAddress},
			value: valueIDR,
		})
		totals[key] = roundIDR(addDecimal(totals[key], valueIDR))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: revalue rows: %w", err)
	}

	for _, item := range updates {
		_, err := tx.ExecContext(ctx, `
			UPDATE wallet_balances SET value_idr = ?
			WHERE wallet_id = ? AND network = ? AND COALESCE(token_address, '') = ?
		`, item.value, item.key.walletID, item.key.network, item.key.tokenAddress)
		if err != nil {
			return fmt.Errorf("store: revalue update: %w", err)
		}
	}

	for key, total := range totals {
		_, err := tx.ExecContext(ctx, `
			UPDATE network_summaries SET total_value_idr = ?, updated_at = ?
			WHERE wallet_id = ? AND network = ?
		`, total, time.Now().UTC().Format(time.RFC3339), key.walletID, key.network)
		if err != nil {
			return fmt.Errorf("store: revalue summary: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: revalue commit: %w", err)
	}
	return nil
}

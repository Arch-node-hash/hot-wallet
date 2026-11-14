package store

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// TokenPrice adalah harga satu token pada satu jaringan. Native token memakai
// TokenAddress kosong. Tabel ini diisi dari luar (mis. watcher harga) dan
// dibaca saat menghitung nilai USD; tidak ada API harga eksternal di sini.
type TokenPrice struct {
	Network      string
	TokenAddress string
	Symbol       string
	PriceUSD     string
	Change24h    float64
	UpdatedAt    string
}

// ListTokenPrices mengambil seluruh harga token.
func (s *Store) ListTokenPrices(ctx context.Context) ([]TokenPrice, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT network, COALESCE(token_address, ''), symbol, price_usd, change_24h, COALESCE(updated_at, '')
		FROM token_prices
		ORDER BY network, symbol
	`)
	if err != nil {
		return nil, fmt.Errorf("store: list token prices: %w", err)
	}
	defer rows.Close()

	var prices []TokenPrice
	for rows.Next() {
		var p TokenPrice
		if err := rows.Scan(&p.Network, &p.TokenAddress, &p.Symbol, &p.PriceUSD, &p.Change24h, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan token price: %w", err)
		}
		prices = append(prices, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: rows token price: %w", err)
	}
	return prices, nil
}

// UpsertTokenPrices menyimpan harga token (native maupun ERC-20/SPL).
func (s *Store) UpsertTokenPrices(ctx context.Context, rows []TokenPrice) error {
	if len(rows) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: token prices tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, row := range rows {
		if row.Network == "" {
			continue
		}
		price := row.PriceUSD
		if price == "" {
			price = "0"
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO token_prices (network, token_address, symbol, price_usd, change_24h, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(network, token_address) DO UPDATE SET
				symbol = excluded.symbol,
				price_usd = excluded.price_usd,
				change_24h = excluded.change_24h,
				updated_at = excluded.updated_at
		`, row.Network, strings.ToLower(row.TokenAddress), row.Symbol, price, row.Change24h, now)
		if err != nil {
			return fmt.Errorf("store: upsert token price %s/%s: %w", row.Network, row.Symbol, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit token prices: %w", err)
	}
	return nil
}

// Package fx mengambil kurs USD -> IDR dari penyedia publik secara berkala,
// menyimpannya di tabel fx_rates, lalu memicu perhitungan ulang nilai IDR.
//
// Kurs tidak di-hardcode: kalau rupiah menguat/melemah, nilai IDR ikut
// menyesuaikan. Bila pengambilan gagal, kurs terakhir di database tetap
// dipakai sehingga API tidak menampilkan nilai kosong.
package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"hotwallet/backend/internal/infrastructure/store"
)

// Config mengatur sumber dan jadwal pengambilan kurs.
type Config struct {
	URL      string
	Interval time.Duration
	Logf     func(string, ...any)
}

// Manager menjalankan pengambilan kurs berkala.
type Manager struct {
	store  *store.Store
	cfg    Config
	client *http.Client
}

// NewManager membuat manager dengan default yang aman.
func NewManager(st *store.Store, cfg Config) *Manager {
	if cfg.URL == "" {
		cfg.URL = "https://open.er-api.com/v6/latest/USD"
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Minute
	}
	if cfg.Logf == nil {
		cfg.Logf = log.Printf
	}
	return &Manager{
		store:  st,
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Run mengambil kurs saat mulai lalu setiap Interval sampai ctx selesai.
func (m *Manager) Run(ctx context.Context) {
	if err := m.Refresh(ctx); err != nil {
		m.cfg.Logf("fx: %v", err)
	}

	ticker := time.NewTicker(m.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.Refresh(ctx); err != nil {
				m.cfg.Logf("fx: %v", err)
			}
		}
	}
}

// Refresh mengambil kurs terbaru, menyimpannya, lalu menghitung ulang nilai
// IDR semua saldo. Bila gagal, kurs lama dibiarkan apa adanya.
func (m *Manager) Refresh(ctx context.Context) error {
	rate, err := m.fetchUSDIDR(ctx)
	if err != nil {
		return err
	}

	source := m.cfg.URL
	if parsed, err := url.Parse(m.cfg.URL); err == nil && parsed.Host != "" {
		source = parsed.Host
	}

	if err := m.store.SaveFXRate(ctx, store.FXRate{
		Base:      "USD",
		Quote:     "IDR",
		Rate:      rate,
		Source:    source,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return err
	}

	if err := m.store.RevalueBalances(ctx, rate); err != nil {
		return fmt.Errorf("hitung ulang idr: %w", err)
	}
	return nil
}

type rateResponse struct {
	Result   string             `json:"result"`
	Base     string             `json:"base"`
	BaseCode string             `json:"base_code"`
	Rates    map[string]float64 `json:"rates"`
}

func (m *Manager) fetchUSDIDR(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.cfg.URL, nil)
	if err != nil {
		return "", fmt.Errorf("buat request kurs: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ambil kurs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ambil kurs: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("baca kurs: %w", err)
	}

	var payload rateResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("parse kurs: %w", err)
	}
	if payload.Result != "" && strings.EqualFold(payload.Result, "error") {
		return "", fmt.Errorf("penyedia kurs mengembalikan error")
	}

	base := payload.BaseCode
	if base == "" {
		base = payload.Base
	}
	if base != "" && !strings.EqualFold(base, "USD") {
		return "", fmt.Errorf("kurs bukan berbasis USD: %s", base)
	}

	value, ok := payload.Rates["IDR"]
	if !ok || value <= 0 {
		return "", fmt.Errorf("kurs IDR tidak tersedia")
	}

	// 2 desimal cukup untuk rupiah dan menjaga format tetap konsisten.
	rate := new(big.Rat).SetFloat64(value)
	if rate == nil {
		return "", fmt.Errorf("kurs IDR tidak valid")
	}
	return new(big.Float).SetRat(rate).Text('f', 2), nil
}

package evm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"hotwallet/backend/internal/blockchain/wsrpc"
)

// Target adalah wallet yang dipantau saldonya.
type Target struct {
	WalletID string
	Address  string
	Network  string
}

// Balance adalah hasil pembacaan saldo yang siap disimpan ke database.
// Raw berisi satuan terkecil (wei), Amount sudah diformat ke desimal token.
type Balance struct {
	WalletID     string
	Network      string
	TokenAddress string
	TokenSymbol  string
	Decimals     uint8
	Raw          string
	Amount       string
}

// Watcher memantau saldo native + token ERC-20 lewat satu koneksi WebSocket:
// eth_subscribe(newHeads) sebagai trigger, lalu eth_getBalance/eth_call
// dikirim sebagai batch pada koneksi yang sama.
type Watcher struct {
	NetworkName    string
	WSSURL         string
	NativeSymbol   string
	NativeDecimals uint8

	Debounce     time.Duration
	RefreshEvery time.Duration

	Targets func() []Target
	Tokens  func() []Token

	// OnBalances dipanggil setiap kali saldo berhasil dibaca.
	OnBalances func(context.Context, []Balance) error

	Logf func(string, ...any)
}

// Run menjaga koneksi tetap hidup dengan reconnect + backoff.
func (w *Watcher) Run(ctx context.Context) error {
	if w.WSSURL == "" {
		return errors.New("evm: wss url kosong")
	}
	if w.Debounce <= 0 {
		w.Debounce = 4 * time.Second
	}
	if w.RefreshEvery <= 0 {
		w.RefreshEvery = 30 * time.Second
	}

	backoff := time.Second
	for {
		started := time.Now()
		err := w.session(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Since(started) > 30*time.Second {
			backoff = time.Second
		}

		w.logf("ws %s terputus: %v (reconnect dalam %s)", w.NetworkName, err, backoff)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}
}

func (w *Watcher) session(ctx context.Context) error {
	conn, err := wsrpc.Dial(ctx, w.WSSURL, 15*time.Second)
	if err != nil {
		return err
	}
	defer conn.CloseNow()

	sessionCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	readErr := make(chan error, 1)
	go func() {
		readErr <- conn.ReadLoop(sessionCtx)
	}()

	trigger := make(chan struct{}, 1)
	signal := func() {
		select {
		case trigger <- struct{}{}:
		default:
		}
	}

	if _, err := conn.Subscribe(sessionCtx, "eth_subscribe", []any{"newHeads"}, func(json.RawMessage) {
		signal()
	}); err != nil {
		return fmt.Errorf("subscribe newHeads: %w", err)
	}
	w.logf("ws %s tersambung, menunggu blok baru", w.NetworkName)

	go w.loop(sessionCtx, conn, trigger)

	select {
	case <-sessionCtx.Done():
		return sessionCtx.Err()
	case err := <-readErr:
		if err == nil {
			err = errors.New("koneksi ditutup")
		}
		return err
	}
}

// loop menjalankan refresh awal, refresh periodik, dan refresh yang dipicu
// blok baru dengan debounce (maksimal 1x per Debounce).
func (w *Watcher) loop(ctx context.Context, conn *wsrpc.Client, trigger <-chan struct{}) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	ticker := time.NewTicker(w.RefreshEvery)
	defer ticker.Stop()

	pending := false
	refresh := func() {
		if err := w.readBalances(ctx, conn); err != nil && ctx.Err() == nil {
			w.logf("ws %s refresh gagal: %v", w.NetworkName, err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-timer.C:
			pending = false
			refresh()

		case <-ticker.C:
			refresh()

		case <-trigger:
			if pending {
				continue
			}
			pending = true
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(w.Debounce)
		}
	}
}

func (w *Watcher) readBalances(ctx context.Context, conn *wsrpc.Client) error {
	targets := w.Targets()
	if len(targets) == 0 {
		return nil
	}
	tokens := w.Tokens()

	type job struct {
		target Target
		token  *Token
	}

	jobs := make([]job, 0, len(targets)*(1+len(tokens)))
	for _, target := range targets {
		jobs = append(jobs, job{target: target})
		for i := range tokens {
			token := tokens[i]
			jobs = append(jobs, job{target: target, token: &token})
		}
	}

	const maxBatch = 100
	var balances []Balance

	for start := 0; start < len(jobs); start += maxBatch {
		end := start + maxBatch
		if end > len(jobs) {
			end = len(jobs)
		}

		reqs := make([]wsrpc.Request, 0, end-start)
		for _, j := range jobs[start:end] {
			if j.token == nil {
				reqs = append(reqs, wsrpc.Request{
					Method: "eth_getBalance",
					Params: []any{j.target.Address, "latest"},
				})
				continue
			}
			reqs = append(reqs, wsrpc.Request{
				Method: "eth_call",
				Params: []any{
					map[string]string{
						"to":   j.token.Address,
						"data": "0x70a08231" + padAddress(j.target.Address),
					},
					"latest",
				},
			})
		}

		results, err := conn.Batch(ctx, reqs)
		if err != nil {
			return err
		}

		for i, raw := range results {
			if len(raw) == 0 {
				continue
			}
			j := jobs[start+i]
			value, err := parseHexInt(raw)
			if err != nil {
				w.logf("ws %s parse saldo gagal: %v", w.NetworkName, err)
				continue
			}

			balance := Balance{
				WalletID: j.target.WalletID,
				Network:  j.target.Network,
				Raw:      value.String(),
			}
			if j.token == nil {
				balance.TokenSymbol = w.NativeSymbol
				balance.Decimals = w.NativeDecimals
			} else {
				balance.TokenAddress = j.token.Address
				balance.TokenSymbol = j.token.Symbol
				balance.Decimals = j.token.Decimals
			}
			balance.Amount = FormatUnits(value, balance.Decimals)
			balances = append(balances, balance)
		}
	}

	if len(balances) > 0 && w.OnBalances != nil {
		return w.OnBalances(ctx, balances)
	}
	return nil
}

func (w *Watcher) logf(format string, args ...any) {
	if w.Logf != nil {
		w.Logf(format, args...)
	}
}

// FormatUnits mengubah satuan terkecil menjadi string desimal tanpa
// kehilangan presisi (mis. wei -> ETH).
func FormatUnits(value *big.Int, decimals uint8) string {
	if value == nil {
		return "0"
	}
	if decimals == 0 {
		return value.String()
	}

	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	quotient := new(big.Int)
	remainder := new(big.Int)
	quotient.QuoRem(value, base, remainder)

	if remainder.Sign() == 0 {
		return quotient.String()
	}

	fraction := remainder.String()
	if len(fraction) < int(decimals) {
		fraction = strings.Repeat("0", int(decimals)-len(fraction)) + fraction
	}
	fraction = strings.TrimRight(fraction, "0")
	if fraction == "" {
		return quotient.String()
	}
	return quotient.String() + "." + fraction
}

func parseHexInt(raw json.RawMessage) (*big.Int, error) {
	var hexValue string
	if err := json.Unmarshal(raw, &hexValue); err != nil {
		return nil, err
	}

	hexValue = strings.TrimPrefix(strings.TrimSpace(hexValue), "0x")
	if hexValue == "" {
		return big.NewInt(0), nil
	}

	value, ok := new(big.Int).SetString(hexValue, 16)
	if !ok {
		return nil, fmt.Errorf("nilai hex tidak valid: %s", hexValue)
	}
	return value, nil
}

func padAddress(address string) string {
	clean := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(address)), "0x")
	if len(clean) < 64 {
		clean = strings.Repeat("0", 64-len(clean)) + clean
	}
	return clean
}

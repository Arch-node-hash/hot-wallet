package solana

import (
	"context"
	"encoding/json"
	"errors"
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
type Balance struct {
	WalletID     string
	Network      string
	TokenAddress string
	TokenSymbol  string
	Decimals     uint8
	Raw          string
	Amount       string
}

// Watcher memantau saldo native dan SPL token lewat satu koneksi WebSocket:
// accountSubscribe untuk notifikasi, getMultipleAccounts untuk saldo awal dan
// refresh periodik. Tidak ada RPC "indexed" (getTokenAccountsByOwner) supaya
// tetap jalan di endpoint publik gratis.
type Watcher struct {
	NetworkName string
	WSSURL      string
	RPCURL      string

	NativeSymbol   string
	NativeDecimals uint8

	RefreshEvery time.Duration

	Targets func() []Target
	Tokens  func() []Token

	OnBalances func(context.Context, []Balance) error
	Logf       func(string, ...any)
}

// Run menjaga koneksi tetap hidup dengan reconnect + backoff.
func (w *Watcher) Run(ctx context.Context) error {
	if w.WSSURL == "" {
		return errors.New("solana: wss url kosong")
	}
	if w.RPCURL == "" {
		w.RPCURL = httpFromWS(w.WSSURL)
	}
	if w.RefreshEvery <= 0 {
		w.RefreshEvery = 60 * time.Second
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

	targets := w.Targets()
	tokens := w.Tokens()

	subscribed := 0
	for _, item := range targets {
		target := item
		params := []any{
			target.Address,
			map[string]any{"encoding": "jsonParsed", "commitment": "confirmed"},
		}
		if _, err := conn.Subscribe(sessionCtx, "accountSubscribe", params, func(raw json.RawMessage) {
			w.handleNative(sessionCtx, target, raw)
		}); err != nil {
			w.logf("ws %s subscribe %s gagal: %v", w.NetworkName, target.Address, err)
			continue
		}
		subscribed++

		for _, ref := range TokenAccountRefs(target.Address, tokens) {
			ref := ref
			if _, err := conn.Subscribe(sessionCtx, "accountSubscribe", []any{
				ref.Address,
				map[string]any{"encoding": "jsonParsed", "commitment": "confirmed"},
			}, func(raw json.RawMessage) {
				w.handleToken(sessionCtx, target, ref.Token, raw)
			}); err != nil {
				w.logf("ws %s subscribe token %s gagal: %v", w.NetworkName, ref.Address, err)
			}
		}
	}
	w.logf("ws %s tersambung (%d wallet, %d langganan)", w.NetworkName, len(targets), subscribed)

	go w.loop(sessionCtx)

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

func (w *Watcher) loop(ctx context.Context) {
	if err := w.refresh(ctx); err != nil && ctx.Err() == nil {
		w.logf("ws %s refresh awal gagal: %v", w.NetworkName, err)
	}

	ticker := time.NewTicker(w.RefreshEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.refresh(ctx); err != nil && ctx.Err() == nil {
				w.logf("ws %s refresh gagal: %v", w.NetworkName, err)
			}
		}
	}
}

type accountValue struct {
	Lamports uint64          `json:"lamports"`
	Data     json.RawMessage `json:"data"`
}

type parsedTokenInfo struct {
	Parsed struct {
		Info struct {
			Mint        string `json:"mint"`
			TokenAmount struct {
				Amount   string `json:"amount"`
				Decimals uint8  `json:"decimals"`
			} `json:"tokenAmount"`
		} `json:"info"`
	} `json:"parsed"`
}

// tokenInfo membaca data akun SPL token. Akun non-token (mis. wallet native)
// mengembalikan data berupa array base64 sehingga diabaikan.
func tokenInfo(raw json.RawMessage) (mint, amount string, decimals uint8, ok bool) {
	if len(raw) == 0 || raw[0] != '{' {
		return "", "", 0, false
	}
	var data parsedTokenInfo
	if err := json.Unmarshal(raw, &data); err != nil {
		return "", "", 0, false
	}
	info := data.Parsed.Info
	if info.Mint == "" || info.TokenAmount.Amount == "" {
		return "", "", 0, false
	}
	return info.Mint, info.TokenAmount.Amount, info.TokenAmount.Decimals, true
}

// refresh membaca saldo native + token sekaligus lewat batch
// getMultipleAccounts. Endpoint publik (PublicNode) tidak mengizinkan request
// JSON-RPC lewat WebSocket, jadi query memakai HTTP; langganan tetap lewat WS.
func (w *Watcher) refresh(ctx context.Context) error {
	targets := w.Targets()
	if len(targets) == 0 {
		return nil
	}
	tokens := w.Tokens()
	client := NewClient(w.RPCURL)

	type entry struct {
		target Target
		token  *Token // nil berarti saldo native
	}

	var entries []entry
	var addresses []string

	for _, target := range targets {
		entries = append(entries, entry{target: target})
		addresses = append(addresses, target.Address)

		for _, ref := range TokenAccountRefs(target.Address, tokens) {
			token := ref.Token
			entries = append(entries, entry{target: target, token: &token})
			addresses = append(addresses, ref.Address)
		}
	}

	nativeAmounts := make(map[string]*big.Int, len(targets))
	tokenAmounts := make(map[string]*big.Int, len(entries))
	tokenDecimals := make(map[string]uint8, len(entries))

	const maxAccounts = 100
	for start := 0; start < len(addresses); start += maxAccounts {
		end := start + maxAccounts
		if end > len(addresses) {
			end = len(addresses)
		}

		var result struct {
			Value []*accountValue `json:"value"`
		}
		params := []any{
			addresses[start:end],
			map[string]string{"encoding": "jsonParsed", "commitment": "confirmed"},
		}
		if err := client.call(ctx, "getMultipleAccounts", params, &result); err != nil {
			return err
		}

		for i, account := range result.Value {
			index := start + i
			if index >= len(entries) || account == nil {
				continue
			}
			item := entries[index]

			if item.token == nil {
				nativeAmounts[item.target.WalletID] = new(big.Int).SetUint64(account.Lamports)
				continue
			}

			_, amountText, decimals, ok := tokenInfo(account.Data)
			if !ok {
				continue
			}
			amount, ok := new(big.Int).SetString(amountText, 10)
			if !ok {
				continue
			}

			key := tokenKey(item.target.WalletID, item.token.Mint)
			if tokenAmounts[key] == nil {
				tokenAmounts[key] = new(big.Int)
			}
			tokenAmounts[key].Add(tokenAmounts[key], amount)
			if decimals > 0 {
				tokenDecimals[key] = decimals
			}
		}
	}

	balances := make([]Balance, 0, len(targets)*(1+len(tokens)))
	for _, target := range targets {
		lamports := nativeAmounts[target.WalletID]
		if lamports == nil {
			lamports = new(big.Int)
		}
		balances = append(balances, w.nativeBalance(target, lamports))

		for _, token := range tokens {
			key := tokenKey(target.WalletID, token.Mint)
			amount := tokenAmounts[key]
			if amount == nil {
				amount = new(big.Int)
			}
			decimals := token.Decimals
			if decimals == 0 {
				decimals = tokenDecimals[key]
			}
			balances = append(balances, Balance{
				WalletID:     target.WalletID,
				Network:      target.Network,
				TokenAddress: token.Mint,
				TokenSymbol:  token.Symbol,
				Decimals:     decimals,
				Raw:          amount.String(),
				Amount:       formatUnits(amount, decimals),
			})
		}
	}

	w.emit(ctx, balances)
	return nil
}

func (w *Watcher) handleNative(ctx context.Context, target Target, raw json.RawMessage) {
	var payload struct {
		Value struct {
			Lamports uint64 `json:"lamports"`
		} `json:"value"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return
	}

	w.emit(ctx, []Balance{
		w.nativeBalance(target, new(big.Int).SetUint64(payload.Value.Lamports)),
	})
}

func (w *Watcher) handleToken(ctx context.Context, target Target, token Token, raw json.RawMessage) {
	var payload struct {
		Value accountValue `json:"value"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return
	}

	_, amountText, parsedDecimals, ok := tokenInfo(payload.Value.Data)
	if !ok {
		return
	}
	amount, ok := new(big.Int).SetString(amountText, 10)
	if !ok {
		return
	}

	decimals := token.Decimals
	if decimals == 0 {
		decimals = parsedDecimals
	}

	w.emit(ctx, []Balance{{
		WalletID:     target.WalletID,
		Network:      target.Network,
		TokenAddress: token.Mint,
		TokenSymbol:  token.Symbol,
		Decimals:     decimals,
		Raw:          amount.String(),
		Amount:       formatUnits(amount, decimals),
	}})
}

func (w *Watcher) nativeBalance(target Target, lamports *big.Int) Balance {
	return Balance{
		WalletID:    target.WalletID,
		Network:     target.Network,
		TokenSymbol: w.NativeSymbol,
		Decimals:    w.NativeDecimals,
		Raw:         lamports.String(),
		Amount:      formatUnits(lamports, w.NativeDecimals),
	}
}

func (w *Watcher) emit(ctx context.Context, balances []Balance) {
	if len(balances) == 0 || w.OnBalances == nil {
		return
	}
	out := make([]Balance, len(balances))
	copy(out, balances)
	if err := w.OnBalances(ctx, out); err != nil {
		w.logf("ws %s simpan saldo gagal: %v", w.NetworkName, err)
	}
}

func (w *Watcher) logf(format string, args ...any) {
	if w.Logf != nil {
		w.Logf(format, args...)
	}
}

func tokenKey(walletID, mint string) string {
	return walletID + "\x00" + mint
}

// httpFromWS menurunkan URL HTTP dari URL WebSocket yang sama, dipakai bila
// endpoint HTTP untuk Solana belum diisi.
func httpFromWS(url string) string {
	switch {
	case strings.HasPrefix(url, "wss://"):
		return "https://" + strings.TrimPrefix(url, "wss://")
	case strings.HasPrefix(url, "ws://"):
		return "http://" + strings.TrimPrefix(url, "ws://")
	default:
		return url
	}
}

func formatUnits(value *big.Int, decimals uint8) string {
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

package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"hotwallet/backend/internal/blockchain/evm"
	"hotwallet/backend/internal/blockchain/network"
	"hotwallet/backend/internal/blockchain/solana"
	bwallet "hotwallet/backend/internal/blockchain/wallet"
	"hotwallet/backend/internal/blockchain/wsrpc"
	"hotwallet/backend/internal/infrastructure/store"
)

// Diagnostics memeriksa semua jaringan secara paralel: RPC hidup, chain id
// cocok, alamat wallet terbaca, saldo/nonce/gas, dan kontrak token.
func (s *Service) Diagnostics(ctx context.Context) (*DiagnosticsResponse, error) {
	rows, err := s.store.ListWallets(ctx)
	if err != nil {
		return nil, err
	}
	configs, err := s.store.ListRPCConfigs()
	if err != nil {
		return nil, err
	}
	tokens, err := s.store.ListTokens()
	if err != nil {
		return nil, err
	}

	rowByKey := make(map[string]store.WalletRow)
	for _, row := range rows {
		if key := network.Canonical(row.Network); key != "" {
			if _, ok := rowByKey[key]; !ok {
				rowByKey[key] = row
			}
		}
	}
	cfgByKey := make(map[string]store.RPCConfig)
	for _, cfg := range configs {
		if key := network.Canonical(cfg.Network); key != "" {
			cfgByKey[key] = cfg
		}
	}

	metas := network.AllMeta()
	results := make([]NetworkDiagnostics, len(metas))

	var wg sync.WaitGroup
	for i, meta := range metas {
		wg.Add(1)
		go func(i int, meta network.Meta) {
			defer wg.Done()
			results[i] = s.diagnoseNetwork(ctx, meta, rowByKey, cfgByKey, tokens)
		}(i, meta)
	}
	wg.Wait()

	resp := &DiagnosticsResponse{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Total:     len(results),
		Networks:  results,
	}
	for _, r := range results {
		if r.Reachable && (r.ChainIDExpected == 0 || r.ChainIDOK) {
			resp.Ready++
		}
	}
	return resp, nil
}

func (s *Service) diagnoseNetwork(
	ctx context.Context,
	meta network.Meta,
	rowByKey map[string]store.WalletRow,
	cfgByKey map[string]store.RPCConfig,
	tokens []store.TokenRow,
) NetworkDiagnostics {
	d := NetworkDiagnostics{
		NetworkKey:      meta.Key,
		Network:         meta.Name,
		NativeSymbol:    meta.NativeSymbol,
		ChainIDExpected: meta.ChainID,
	}

	if row, ok := rowByKey[meta.Key]; ok {
		d.Address = row.Address
	} else if info, err := s.privateKeyInfo(meta.Chain); err == nil {
		if address, _, err := bwallet.DeriveAddressFromPrivateKey(info, 0); err == nil {
			d.Address = address
		}
	}
	if d.Address == "" {
		d.Error = "tidak ada wallet/alamat (isi private key di env)"
		return d
	}

	cfg := cfgByKey[meta.Key]

	// Endpoint HTTP yang dipakai jalur kirim. Untuk Solana, turunkan dari WSS
	// bila rpc_url kosong (PublicNode menolak JSON-RPC lewat WebSocket).
	httpURL := strings.TrimSpace(cfg.RPCURL)
	derived := false
	if httpURL == "" && meta.Chain == network.ChainSolana && strings.TrimSpace(cfg.WSSURL) != "" {
		httpURL = solana.HTTPFromWS(cfg.WSSURL)
		derived = true
	}

	checkCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	var (
		caller  rpcCaller
		cleanup = func() {}
	)

	if meta.Chain == network.ChainSolana {
		if httpURL == "" {
			d.Error = "rpc_url dan wss_url belum diisi"
			return d
		}
		caller = newHTTPRPC(httpURL)
		d.Transport = "HTTP"
		if derived {
			d.Transport = "HTTP (dari WSS)"
		}
	} else {
		if strings.TrimSpace(cfg.WSSURL) != "" {
			if conn, err := wsrpc.Dial(checkCtx, cfg.WSSURL, 8*time.Second); err == nil {
				go conn.ReadLoop(checkCtx)
				caller = conn
				cleanup = func() { conn.CloseNow() }
				d.Transport = "WS"
			}
		}
		if caller == nil {
			if httpURL == "" {
				d.Error = "rpc_url dan wss_url belum diisi"
				return d
			}
			caller = newHTTPRPC(httpURL)
			d.Transport = "HTTP"
		}
	}
	defer cleanup()

	networkTokens := diagnosticsTokens(tokens, meta.Key)
	if meta.Chain == network.ChainSolana {
		fillSolanaDiagnostics(checkCtx, caller, &d, meta, networkTokens)
	} else {
		fillEVMDiagnostics(checkCtx, caller, &d, meta, networkTokens)
	}
	d.Reachable = d.Error == ""
	return d
}

func fillEVMDiagnostics(ctx context.Context, c rpcCaller, d *NetworkDiagnostics, meta network.Meta, tokens []store.TokenRow) {
	var chainHex string
	if err := c.Call(ctx, "eth_chainId", []any{}, &chainHex); err != nil {
		d.Error = "eth_chainId: " + err.Error()
		return
	}
	chainID, err := parseHex(chainHex)
	if err != nil {
		d.Error = err.Error()
		return
	}
	d.ChainID = chainID.Int64()
	d.ChainIDOK = d.ChainID == d.ChainIDExpected

	var balanceHex string
	if err := c.Call(ctx, "eth_getBalance", []any{d.Address, "latest"}, &balanceHex); err != nil {
		d.Error = "eth_getBalance: " + err.Error()
		return
	}
	balance, err := parseHex(balanceHex)
	if err != nil {
		d.Error = err.Error()
		return
	}
	d.NativeBalance = evm.FormatUnits(balance, meta.NativeDecimals)

	var nonceHex string
	if err := c.Call(ctx, "eth_getTransactionCount", []any{d.Address, "pending"}, &nonceHex); err == nil {
		if nonce, err := parseHex(nonceHex); err == nil {
			d.Nonce = nonce.Uint64()
		}
	}

	var gasHex string
	if err := c.Call(ctx, "eth_gasPrice", []any{}, &gasHex); err == nil {
		if gasPrice, err := parseHex(gasHex); err == nil {
			d.GasPriceWei = gasPrice.String()
			d.GasPriceGwei = evm.FormatUnits(gasPrice, 9)
			fee := new(big.Int).Mul(big.NewInt(21000), gasPrice)
			d.GasFeeNative = evm.FormatUnits(fee, meta.NativeDecimals)
			d.GasReady = balance.Cmp(fee) >= 0
		}
	}

	for _, token := range tokens {
		item := TokenDiagnostics{Symbol: token.Symbol, Address: token.Address}
		var out string
		if err := c.Call(ctx, "eth_call", []any{
			map[string]string{"to": token.Address, "data": "0x313ce567"},
			"latest",
		}, &out); err != nil {
			item.Error = err.Error()
		} else if strings.TrimSpace(out) == "" || strings.TrimSpace(out) == "0x" {
			item.Error = "kontrak tidak mengembalikan data"
		} else if decimals, err := parseHex(out); err != nil {
			item.Error = err.Error()
		} else {
			item.Decimals = int(decimals.Int64())
			item.OK = item.Decimals == token.Decimals
			if !item.OK {
				item.Error = fmt.Sprintf("decimals on-chain %d != DB %d", item.Decimals, token.Decimals)
			}
		}
		d.Tokens = append(d.Tokens, item)
	}
}

func fillSolanaDiagnostics(ctx context.Context, c rpcCaller, d *NetworkDiagnostics, meta network.Meta, tokens []store.TokenRow) {
	// Solana tidak memakai chain id ala EVM.
	d.ChainIDOK = true

	var balance struct {
		Value uint64 `json:"value"`
	}
	if err := c.Call(ctx, "getBalance", []any{d.Address}, &balance); err != nil {
		d.Error = "getBalance: " + err.Error()
		return
	}
	d.NativeBalance = solana.FormatUnits(new(big.Int).SetUint64(balance.Value), meta.NativeDecimals)
	d.GasFeeNative = solana.FormatUnits(big.NewInt(5000), meta.NativeDecimals)
	d.GasReady = balance.Value >= 5000

	var blockhash struct {
		Value struct {
			Blockhash string `json:"blockhash"`
		} `json:"value"`
	}
	if err := c.Call(ctx, "getLatestBlockhash", []any{map[string]string{"commitment": "confirmed"}}, &blockhash); err == nil {
		d.Blockhash = blockhash.Value.Blockhash
	}

	var version struct {
		SolanaCore string `json:"solana-core"`
	}
	if err := c.Call(ctx, "getVersion", []any{}, &version); err == nil {
		d.Version = version.SolanaCore
	}

	for _, token := range tokens {
		item := TokenDiagnostics{Symbol: token.Symbol, Address: token.Address}
		var info struct {
			Value json.RawMessage `json:"value"`
		}
		if err := c.Call(ctx, "getAccountInfo", []any{token.Address, map[string]any{"encoding": "base64"}}, &info); err != nil {
			item.Error = err.Error()
		} else if len(info.Value) == 0 || strings.TrimSpace(string(info.Value)) == "null" {
			item.Error = "mint tidak ditemukan"
		} else {
			item.OK = true
		}
		d.Tokens = append(d.Tokens, item)
	}
}

func diagnosticsTokens(tokens []store.TokenRow, key string) []store.TokenRow {
	var out []store.TokenRow
	for _, token := range tokens {
		if network.Canonical(token.Network) == key {
			out = append(out, token)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// JSON-RPC minimal (WebSocket atau HTTP) untuk diagnostik
// ---------------------------------------------------------------------------

type rpcCaller interface {
	Call(ctx context.Context, method string, params any, out any) error
}

type httpRPC struct {
	url    string
	client *http.Client
}

func newHTTPRPC(url string) *httpRPC {
	return &httpRPC{url: url, client: &http.Client{Timeout: 15 * time.Second}}
}

func (c *httpRPC) Call(ctx context.Context, method string, params any, out any) error {
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}

	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("decode %s: %w", method, err)
	}
	if envelope.Error != nil {
		return fmt.Errorf("rpc error %d: %s", envelope.Error.Code, envelope.Error.Message)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(envelope.Result, out)
}

func parseHex(value string) (*big.Int, error) {
	clean := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	if clean == "" {
		return big.NewInt(0), nil
	}
	n, ok := new(big.Int).SetString(clean, 16)
	if !ok {
		return nil, fmt.Errorf("nilai hex tidak valid: %q", value)
	}
	return n, nil
}

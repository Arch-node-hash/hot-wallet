// Command preflight memeriksa kesiapan jalur kirim saldo ke jaringan mainnet
// secara READ-ONLY: memastikan endpoint RPC/WSS hidup, chain id cocok, alamat
// wallet terbaca, saldo/nonce tersedia, harga gas masuk akal, dan kontrak token
// benar-benar ada. Perintah ini TIDAK menandatangani atau menyiarkan transaksi
// apa pun, dan tidak mengubah database.
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"hotwallet/backend/config"
	"hotwallet/backend/internal/blockchain/evm"
	"hotwallet/backend/internal/blockchain/network"
	"hotwallet/backend/internal/blockchain/solana"
	"hotwallet/backend/internal/blockchain/wallet"
	"hotwallet/backend/internal/blockchain/wsrpc"
	"hotwallet/backend/internal/infrastructure/store"
)

// rpcCaller adalah satu panggilan JSON-RPC, dipenuhi oleh koneksi WebSocket
// (wsrpc.Client) maupun HTTP (httpRPC) sehingga logika pengecekan sama.
type rpcCaller interface {
	Call(ctx context.Context, method string, params any, out any) error
}

// httpRPC adalah JSON-RPC minimal lewat HTTP (fallback bila WSS tidak ada).
type httpRPC struct {
	url    string
	client *http.Client
}

func newHTTPRPC(url string) *httpRPC {
	return &httpRPC{url: url, client: &http.Client{Timeout: 20 * time.Second}}
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

type tokenResult struct {
	Symbol   string
	Address  string
	Decimals int
	OK       bool
	Err      string
}

// walletInfo adalah data wallet minimum yang dibutuhkan preflight.
type walletInfo struct {
	ID              string
	Network         string
	Address         string
	DerivationIndex int
}

type report struct {
	Meta      network.Meta
	Transport string

	Address    string
	ChainID    int64
	ChainIDOK  bool
	RPCMissing bool

	NativeSymbol string
	NativeAmount string
	Nonce        uint64
	GasPriceWei  string
	GasNeeded    string
	GasReady     bool

	Blockhash string
	Version   string

	Tokens []tokenResult
	Err    error
}

func main() {
	filter := flag.String("network", "", "batasi ke jaringan tertentu (dipisah koma), mis. ethereum,solana")
	timeout := flag.Duration("timeout", 20*time.Second, "timeout per panggilan RPC")
	flag.Parse()

	cfg := config.Load()

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buka database: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	wallets, err := loadWallets(ctx, st)
	if err != nil {
		fmt.Fprintf(os.Stderr, "baca wallet: %v\n", err)
		os.Exit(1)
	}
	walletByNetwork := make(map[string]walletInfo)
	for _, w := range wallets {
		key := network.Canonical(w.Network)
		if key == "" {
			continue
		}
		if _, ok := walletByNetwork[key]; !ok {
			walletByNetwork[key] = w
		}
	}

	configs, err := st.ListRPCConfigs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "baca rpc_configs: %v\n", err)
		os.Exit(1)
	}
	configByKey := make(map[string]store.RPCConfig)
	for _, c := range configs {
		if key := network.Canonical(c.Network); key != "" {
			configByKey[key] = c
		}
	}

	tokens, _ := st.ListTokens()

	// Alamat cadangan dari env (index 0) bila belum ada baris wallet di DB.
	// Ini hanya membaca; tidak menulis wallet baru.
	evmInfo, _ := wallet.ParseEVMPrivateKey(cfg.ETHPrivateKey)
	solanaInfo, _ := wallet.ParseSolanaPrivateKey(cfg.SolanaPrivateKey)

	selected := parseFilter(*filter)

	fmt.Println("Hot Wallet — preflight mainnet (READ-ONLY, tidak mengirim transaksi)")
	fmt.Printf("Database : %s\n", cfg.DBPath)
	fmt.Printf("Mode     : WebSocket bila wss_url tersedia, selain itu HTTP\n")
	fmt.Println()

	var reports []report
	for _, meta := range network.AllMeta() {
		if len(selected) > 0 && !selected[meta.Key] {
			continue
		}

		rep := checkNetwork(ctx, *timeout, meta, walletByNetwork, configByKey, tokens, evmInfo, solanaInfo)
		reports = append(reports, rep)
		printReport(rep)
	}

	printSummary(reports)
}

func checkNetwork(
	ctx context.Context,
	timeout time.Duration,
	meta network.Meta,
	walletByNetwork map[string]walletInfo,
	configByKey map[string]store.RPCConfig,
	tokens []store.TokenRow,
	evmInfo, solanaInfo *wallet.PrivateKeyInfo,
) report {
	rep := report{Meta: meta}

	// Tentukan alamat wallet untuk jaringan ini.
	if w, ok := walletByNetwork[meta.Key]; ok {
		rep.Address = w.Address
	} else if meta.Chain == network.ChainSolana && solanaInfo != nil {
		if address, _, err := wallet.DeriveAddressFromPrivateKey(solanaInfo, 0); err == nil {
			rep.Address = address
		}
	} else if meta.Chain == network.ChainEVM && evmInfo != nil {
		if address, _, err := wallet.DeriveAddressFromPrivateKey(evmInfo, 0); err == nil {
			rep.Address = address
		}
	}
	if rep.Address == "" {
		rep.Err = fmt.Errorf("tidak ada wallet/alamat untuk %s (isi private key di .env)", meta.Key)
		return rep
	}

	rpcCfg := configByKey[meta.Key]

	// Endpoint HTTP yang benar-benar dipakai jalur kirim. Untuk Solana, bila
	// rpc_url kosong, turunkan dari wss_url (sama seperti resolveRPCURL di app).
	httpURL := strings.TrimSpace(rpcCfg.RPCURL)
	derived := false
	if httpURL == "" && meta.Chain == network.ChainSolana && strings.TrimSpace(rpcCfg.WSSURL) != "" {
		httpURL = solana.HTTPFromWS(rpcCfg.WSSURL)
		derived = true
	}
	rep.RPCMissing = httpURL == ""

	var (
		caller    rpcCaller
		transport string
		cleanup   = func() {}
		err       error
	)

	// PublicNode Solana menolak request JSON-RPC lewat WebSocket, jadi query
	// Solana selalu lewat HTTP. EVM memakai WebSocket bila tersedia.
	if meta.Chain == network.ChainSolana {
		if httpURL == "" {
			rep.Err = fmt.Errorf("rpc_url dan wss_url belum diisi")
			return rep
		}
		caller = newHTTPRPC(httpURL)
		transport = "HTTP"
		if derived {
			transport = "HTTP (dari WSS)"
		}
	} else {
		caller, transport, cleanup, err = dialRPC(ctx, timeout, rpcCfg)
		if err != nil {
			rep.Err = err
			return rep
		}
	}
	defer cleanup()
	rep.Transport = transport

	networkTokens := filterTokens(tokens, meta.Key)

	callCtx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	if meta.Chain == network.ChainSolana {
		return checkSolana(callCtx, caller, rep, networkTokens)
	}
	return checkEVM(callCtx, caller, rep, networkTokens)
}

// dialRPC membuka koneksi WebSocket bila wss_url diisi, selain itu HTTP.
func dialRPC(ctx context.Context, timeout time.Duration, cfg store.RPCConfig) (rpcCaller, string, func(), error) {
	if cfg.WSSURL != "" {
		conn, err := wsrpc.Dial(ctx, cfg.WSSURL, timeout)
		if err == nil {
			go conn.ReadLoop(ctx)
			return conn, "WS", func() { conn.CloseNow() }, nil
		}
		if cfg.RPCURL == "" {
			return nil, "", func() {}, fmt.Errorf("wss gagal (%v) dan rpc_url kosong", err)
		}
		// lanjut ke HTTP di bawah
	}
	if cfg.RPCURL == "" {
		return nil, "", func() {}, fmt.Errorf("rpc_url dan wss_url belum diisi")
	}
	return newHTTPRPC(cfg.RPCURL), "HTTP", func() {}, nil
}

func checkEVM(ctx context.Context, c rpcCaller, rep report, tokens []store.TokenRow) report {
	var chainHex string
	if err := c.Call(ctx, "eth_chainId", []any{}, &chainHex); err != nil {
		rep.Err = fmt.Errorf("eth_chainId: %w", err)
		return rep
	}
	chainID, err := parseHex(chainHex)
	if err != nil {
		rep.Err = err
		return rep
	}
	rep.ChainID = chainID.Int64()
	rep.ChainIDOK = rep.ChainID == rep.Meta.ChainID

	var balHex string
	if err := c.Call(ctx, "eth_getBalance", []any{rep.Address, "latest"}, &balHex); err != nil {
		rep.Err = fmt.Errorf("eth_getBalance: %w", err)
		return rep
	}
	balance, err := parseHex(balHex)
	if err != nil {
		rep.Err = err
		return rep
	}
	rep.NativeSymbol = rep.Meta.NativeSymbol
	rep.NativeAmount = evm.FormatUnits(balance, rep.Meta.NativeDecimals)

	var nonceHex string
	if err := c.Call(ctx, "eth_getTransactionCount", []any{rep.Address, "pending"}, &nonceHex); err == nil {
		if nonce, err := parseHex(nonceHex); err == nil {
			rep.Nonce = nonce.Uint64()
		}
	}

	var gasHex string
	if err := c.Call(ctx, "eth_gasPrice", []any{}, &gasHex); err == nil {
		if gasPrice, err := parseHex(gasHex); err == nil {
			rep.GasPriceWei = gasPrice.String()
			fee := new(big.Int).Mul(big.NewInt(21000), gasPrice)
			rep.GasNeeded = evm.FormatUnits(fee, rep.Meta.NativeDecimals)
			rep.GasReady = balance.Cmp(fee) >= 0
		}
	}

	// decimals() tiap token membuktikan kontrak benar-benar ada di mainnet.
	for _, t := range tokens {
		tr := tokenResult{Symbol: t.Symbol, Address: t.Address}
		var out string
		if err := c.Call(ctx, "eth_call", []any{
			map[string]string{"to": t.Address, "data": "0x313ce567"},
			"latest",
		}, &out); err != nil {
			tr.Err = err.Error()
		} else if strings.TrimSpace(out) == "" || strings.TrimSpace(out) == "0x" {
			tr.Err = "kontrak tidak mengembalikan data"
		} else if decimals, err := parseHex(out); err != nil {
			tr.Err = err.Error()
		} else {
			tr.Decimals = int(decimals.Int64())
			tr.OK = tr.Decimals == t.Decimals
			if !tr.OK {
				tr.Err = fmt.Sprintf("decimals on-chain %d != DB %d", tr.Decimals, t.Decimals)
			}
		}
		rep.Tokens = append(rep.Tokens, tr)
	}

	return rep
}

func checkSolana(ctx context.Context, c rpcCaller, rep report, tokens []store.TokenRow) report {
	var balance struct {
		Value uint64 `json:"value"`
	}
	if err := c.Call(ctx, "getBalance", []any{rep.Address}, &balance); err != nil {
		rep.Err = fmt.Errorf("getBalance: %w", err)
		return rep
	}
	rep.NativeSymbol = rep.Meta.NativeSymbol
	rep.NativeAmount = solana.FormatUnits(new(big.Int).SetUint64(balance.Value), rep.Meta.NativeDecimals)
	rep.GasNeeded = solana.FormatUnits(big.NewInt(5000), rep.Meta.NativeDecimals)
	rep.GasReady = balance.Value >= 5000

	var blockhash struct {
		Value struct {
			Blockhash string `json:"blockhash"`
		} `json:"value"`
	}
	if err := c.Call(ctx, "getLatestBlockhash", []any{map[string]string{"commitment": "confirmed"}}, &blockhash); err == nil {
		rep.Blockhash = blockhash.Value.Blockhash
	}

	var version struct {
		SolanaCore string `json:"solana-core"`
	}
	if err := c.Call(ctx, "getVersion", []any{}, &version); err == nil {
		rep.Version = version.SolanaCore
	}

	// Keberadaan mint membuktikan alamat token benar di mainnet.
	for _, t := range tokens {
		tr := tokenResult{Symbol: t.Symbol, Address: t.Address}
		var info struct {
			Value json.RawMessage `json:"value"`
		}
		if err := c.Call(ctx, "getAccountInfo", []any{t.Address, map[string]any{"encoding": "base64"}}, &info); err != nil {
			tr.Err = err.Error()
		} else if len(info.Value) == 0 || strings.TrimSpace(string(info.Value)) == "null" {
			tr.Err = "mint tidak ditemukan"
		} else {
			tr.OK = true
		}
		rep.Tokens = append(rep.Tokens, tr)
	}

	return rep
}

func printReport(rep report) {
	fmt.Printf("[%s] %s  (%s)\n", rep.Meta.Key, rep.Meta.Name, transportOrDash(rep.Transport))

	if rep.Err != nil {
		fmt.Printf("  status         : GAGAL — %v\n\n", rep.Err)
		return
	}

	fmt.Printf("  address        : %s\n", rep.Address)
	if rep.Meta.Chain == network.ChainEVM {
		if rep.ChainIDOK {
			fmt.Printf("  chain id       : %d (cocok)\n", rep.ChainID)
		} else {
			fmt.Printf("  chain id       : %d (TIDAK cocok, harusnya %d)\n", rep.ChainID, rep.Meta.ChainID)
		}
	} else if rep.Version != "" {
		fmt.Printf("  solana core    : %s\n", rep.Version)
	}
	fmt.Printf("  saldo native   : %s %s\n", rep.NativeAmount, rep.NativeSymbol)
	if rep.RPCMissing {
		fmt.Printf("  rpc_url        : KOSONG — jalur KIRIM (HTTP) tidak akan jalan\n")
	}
	if rep.Meta.Chain == network.ChainEVM {
		fmt.Printf("  nonce          : %d\n", rep.Nonce)
		fmt.Printf("  gas price      : %s gwei\n", gwei(rep.GasPriceWei))
	}
	if rep.Blockhash != "" {
		fmt.Printf("  blockhash      : %s\n", rep.Blockhash)
	}
	if rep.GasReady {
		fmt.Printf("  biaya gas      : ~%s %s (saldo cukup)\n", rep.GasNeeded, rep.NativeSymbol)
	} else {
		fmt.Printf("  biaya gas      : ~%s %s (SALDO KURANG untuk gas)\n", rep.GasNeeded, rep.NativeSymbol)
	}
	for _, t := range rep.Tokens {
		if t.OK {
			if t.Decimals > 0 {
				fmt.Printf("  token          : %s %s OK (decimals %d)\n", t.Symbol, shortAddress(t.Address), t.Decimals)
			} else {
				fmt.Printf("  token          : %s %s OK\n", t.Symbol, shortAddress(t.Address))
			}
		} else {
			fmt.Printf("  token          : %s %s GAGAL (%s)\n", t.Symbol, shortAddress(t.Address), t.Err)
		}
	}
	fmt.Println()
}

func printSummary(reports []report) {
	ready := 0
	for _, rep := range reports {
		if rep.Err == nil && (rep.Meta.Chain != network.ChainEVM || rep.ChainIDOK) {
			ready++
		}
	}
	fmt.Printf("Ringkasan: %d/%d jaringan siap menerima perintah kirim", ready, len(reports))
	if ready < len(reports) {
		fmt.Print(" — periksa baris GAGAL di atas")
	}
	fmt.Println()
	fmt.Println("Catatan: preflight ini hanya membaca. Bukti broadcast mainnet tetap")
	fmt.Println("membutuhkan satu transaksi mainnet nyata (nominal kecil).")
}

func parseFilter(value string) map[string]bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	out := make(map[string]bool)
	for _, part := range strings.Split(value, ",") {
		if key := network.Canonical(part); key != "" {
			out[key] = true
		}
	}
	return out
}

func filterTokens(tokens []store.TokenRow, key string) []store.TokenRow {
	var out []store.TokenRow
	for _, t := range tokens {
		if network.Canonical(t.Network) == key {
			out = append(out, t)
		}
	}
	return out
}

// loadWallets membaca wallets tanpa mengubah database. Kolom derivation_index
// hanya dipakai bila memang sudah ada (DB lama belum dimigrasi), sehingga
// preflight tetap bisa jalan sebagai pembaca murni.
func loadWallets(ctx context.Context, st *store.Store) ([]walletInfo, error) {
	hasIndex := columnExists(st.DB(), "wallets", "derivation_index")

	query := "SELECT id, network, address FROM wallets"
	if hasIndex {
		query = "SELECT id, network, address, derivation_index FROM wallets"
	}

	rows, err := st.DB().QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []walletInfo
	for rows.Next() {
		var w walletInfo
		if hasIndex {
			if err := rows.Scan(&w.ID, &w.Network, &w.Address, &w.DerivationIndex); err != nil {
				return nil, err
			}
		} else if err := rows.Scan(&w.ID, &w.Network, &w.Address); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func columnExists(db *sql.DB, table, column string) bool {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false
		}
		if name == column {
			return true
		}
	}
	return false
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

func gwei(wei string) string {
	if wei == "" {
		return "?"
	}
	n, ok := new(big.Int).SetString(wei, 10)
	if !ok {
		return wei
	}
	return evm.FormatUnits(n, 9)
}

func shortAddress(address string) string {
	if len(address) <= 12 {
		return address
	}
	return address[:6] + "..." + address[len(address)-4:]
}

func transportOrDash(transport string) string {
	if transport == "" {
		return "tidak tersambung"
	}
	return transport
}

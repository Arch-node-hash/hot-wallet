package wallet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"hotwallet/backend/internal/blockchain/evm"
	"hotwallet/backend/internal/blockchain/network"
	"hotwallet/backend/internal/blockchain/solana"
	"hotwallet/backend/internal/blockchain/wallet"
	"hotwallet/backend/internal/infrastructure/store"
)

// stablecoins selalu dihargai $1 walau tabel token_prices kosong.
var stablecoins = map[string]bool{
	"USDT": true, "USDC": true, "DAI": true, "BUSD": true,
	"TUSD": true, "FDUSD": true, "USDE": true, "USDD": true,
}

func canonical(name string) string { return network.Canonical(name) }

func isStablecoin(symbol string) bool {
	return stablecoins[strings.ToUpper(strings.TrimSpace(symbol))]
}

// ---------------------------------------------------------------------------
// View types
// ---------------------------------------------------------------------------

// TokenBalanceView adalah satu baris token pada kartu jaringan.
type TokenBalanceView struct {
	TokenAddress string  `json:"token_address"`
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name,omitempty"`
	Decimals     int     `json:"decimals"`
	Balance      string  `json:"balance"`
	PriceUSD     string  `json:"price_usd"`
	ValueUSD     string  `json:"value_usd"`
	Change24h    float64 `json:"change_24h"`
	IsNative     bool    `json:"is_native"`
}

// NetworkCard adalah data satu kartu jaringan pada tab ALL / tab jaringan.
type NetworkCard struct {
	NetworkKey    string             `json:"network_key"`
	Network       string             `json:"network"`
	DisplayName   string             `json:"display_name"`
	Short         string             `json:"short"`
	TokenStandard string             `json:"token_standard"`
	AddressType   string             `json:"address_type"`
	ExplorerURL   string             `json:"explorer_url,omitempty"`
	WalletID      string             `json:"wallet_id,omitempty"`
	Address       string             `json:"address,omitempty"`
	AddressShort  string             `json:"address_short,omitempty"`
	NativeSymbol  string             `json:"native_symbol"`
	NativeBalance string             `json:"native_balance"`
	USDTBalance   string             `json:"usdt_balance"`
	TotalValueUSD string             `json:"total_value_usd"`
	Change24h     float64            `json:"change_24h_percent"`
	HasWallet     bool               `json:"has_wallet"`
	Tokens        []TokenBalanceView `json:"tokens"`
}

// OverviewResponse adalah payload tab ALL.
type OverviewResponse struct {
	TotalValueUSD    string        `json:"total_value_usd"`
	Change24hUSD     string        `json:"change_24h_usd"`
	Change24hPercent float64       `json:"change_24h_percent"`
	RefreshedAt      string        `json:"refreshed_at"`
	Networks         []NetworkCard `json:"networks"`
}

// AddressGroup adalah satu kolom pada modal alamat (EVM universal / Solana).
type AddressGroup struct {
	AddressType  string   `json:"address_type"`
	Address      string   `json:"address"`
	AddressShort string   `json:"address_short"`
	Networks     []string `json:"networks"`
	Shorts       []string `json:"shorts"`
	QRData       string   `json:"qr_data"`
}

// AddressesResponse adalah payload modal alamat (2 kolom).
type AddressesResponse struct {
	EVM    *AddressGroup `json:"evm"`
	Solana *AddressGroup `json:"solana"`
}

// NetworkTab adalah metadata tab di halaman Wallet.
type NetworkTab struct {
	NetworkKey    string `json:"network_key"`
	Network       string `json:"network"`
	DisplayName   string `json:"display_name"`
	Short         string `json:"short"`
	TokenStandard string `json:"token_standard"`
	AddressType   string `json:"address_type"`
	WalletID      string `json:"wallet_id,omitempty"`
	Address       string `json:"address,omitempty"`
	AddressShort  string `json:"address_short,omitempty"`
	HasWallet     bool   `json:"has_wallet"`
}

// RotatedWallet adalah wallet yang alamatnya ikut diperbarui saat rotate.
type RotatedWallet struct {
	WalletID        string `json:"wallet_id"`
	Network         string `json:"network"`
	OldAddress      string `json:"old_address"`
	NewAddress      string `json:"new_address"`
	DerivationIndex int    `json:"derivation_index"`
}

// RotateResponse adalah hasil "Perbaharui Address".
type RotateResponse struct {
	Chain           string          `json:"chain"`
	NetworkKey      string          `json:"network_key"`
	AddressType     string          `json:"address_type"`
	OldAddress      string          `json:"old_address"`
	NewAddress      string          `json:"new_address"`
	DerivationIndex int             `json:"derivation_index"`
	Note            string          `json:"note"`
	Wallets         []RotatedWallet `json:"wallets"`
}

// SendEstimateRequest adalah parameter estimasi biaya kirim.
type SendEstimateRequest struct {
	To           string `json:"to"`
	TokenAddress string `json:"token_address"`
	Amount       string `json:"amount"`
}

// SendEstimateResponse adalah hasil estimasi biaya.
type SendEstimateResponse struct {
	Network      string `json:"network"`
	NetworkKey   string `json:"network_key"`
	TokenSymbol  string `json:"token_symbol"`
	TokenAddress string `json:"token_address"`
	Amount       string `json:"amount"`
	NativeSymbol string `json:"native_symbol"`
	FeeNative    string `json:"fee_native"`
	FeeUSD       string `json:"fee_usd"`
	GasLimit     uint64 `json:"gas_limit,omitempty"`
	GasPrice     string `json:"gas_price,omitempty"`
	TotalUSD     string `json:"total_usd"`
}

// SendRequest adalah parameter kirim (langkah konfirmasi).
type SendRequest struct {
	To           string `json:"to"`
	TokenAddress string `json:"token_address"`
	Amount       string `json:"amount"`
	Pin          string `json:"pin,omitempty"`
}

// SendResponse adalah hasil broadcast transaksi.
type SendResponse struct {
	TxHash      string `json:"tx_hash"`
	Network     string `json:"network"`
	NetworkKey  string `json:"network_key"`
	TokenSymbol string `json:"token_symbol"`
	Amount      string `json:"amount"`
	To          string `json:"to"`
	ExplorerURL string `json:"explorer_url,omitempty"`
	Status      string `json:"status"`
}

// ---------------------------------------------------------------------------
// Index harga & metadata token
// ---------------------------------------------------------------------------

type priceIndex struct {
	byAddress map[string]store.TokenPrice
	bySymbol  map[string]store.TokenPrice
}

func newPriceIndex(prices []store.TokenPrice) *priceIndex {
	idx := &priceIndex{
		byAddress: make(map[string]store.TokenPrice),
		bySymbol:  make(map[string]store.TokenPrice),
	}
	for _, p := range prices {
		key := canonical(p.Network)
		idx.byAddress[key+"\x00"+strings.ToLower(p.TokenAddress)] = p
		idx.bySymbol[key+"\x00"+strings.ToUpper(p.Symbol)] = p
	}
	return idx
}

// lookup mencari harga token: cocok alamat dulu, lalu simbol, lalu fallback
// stablecoin $1.
func (idx *priceIndex) lookup(networkName, tokenAddress, symbol string) (string, float64) {
	key := canonical(networkName)
	if p, ok := idx.byAddress[key+"\x00"+strings.ToLower(tokenAddress)]; ok {
		return normalizePrice(p.PriceUSD), p.Change24h
	}
	if p, ok := idx.bySymbol[key+"\x00"+strings.ToUpper(symbol)]; ok {
		return normalizePrice(p.PriceUSD), p.Change24h
	}
	if isStablecoin(symbol) {
		return "1", 0
	}
	return "0", 0
}

func normalizePrice(price string) string {
	if strings.TrimSpace(price) == "" {
		return "0"
	}
	return price
}

// ---------------------------------------------------------------------------
// Service: overview / addresses / networks
// ---------------------------------------------------------------------------

// Overview membangun data tab ALL: total USD, perubahan 24 jam, dan kartu
// tiap jaringan diurutkan dari nilai USD terbesar.
func (s *Service) Overview(ctx context.Context) (*OverviewResponse, error) {
	cards, err := s.buildNetworkCards(ctx)
	if err != nil {
		return nil, err
	}

	total := "0"
	weighted := 0.0
	for _, card := range cards {
		value := toFloat(card.TotalValueUSD)
		total = addUSD(total, card.TotalValueUSD)
		weighted += value * card.Change24h
	}

	totalFloat := toFloat(total)
	percent := 0.0
	if totalFloat > 0 {
		percent = weighted / totalFloat
	}
	changeUSD := mulUSD(total, strconv.FormatFloat(percent/100, 'f', 10, 64))

	return &OverviewResponse{
		TotalValueUSD:    total,
		Change24hUSD:     changeUSD,
		Change24hPercent: round2(percent),
		RefreshedAt:      nowRFC3339(),
		Networks:         cards,
	}, nil
}

func (s *Service) buildNetworkCards(ctx context.Context) ([]NetworkCard, error) {
	wallets, err := s.store.ListWallets(ctx)
	if err != nil {
		return nil, err
	}
	prices, err := s.store.ListTokenPrices(ctx)
	if err != nil {
		return nil, err
	}
	tokens, err := s.store.ListTokens()
	if err != nil {
		return nil, err
	}

	idx := newPriceIndex(prices)
	byNetwork := make(map[string]store.WalletRow)
	for _, w := range wallets {
		if key := canonical(w.Network); key != "" {
			byNetwork[key] = w
		}
	}

	cards := make([]NetworkCard, 0, len(network.AllMeta()))
	for _, meta := range network.AllMeta() {
		card := NetworkCard{
			NetworkKey:    meta.Key,
			Network:       meta.Name,
			DisplayName:   meta.DisplayName,
			Short:         meta.Short,
			TokenStandard: meta.TokenStandard,
			AddressType:   meta.AddressType,
			ExplorerURL:   meta.ExplorerURL,
			NativeSymbol:  meta.NativeSymbol,
			NativeBalance: "0",
			USDTBalance:   "0",
			TotalValueUSD: "0.00",
			Tokens:        []TokenBalanceView{},
		}

		row, ok := byNetwork[meta.Key]
		if !ok {
			cards = append(cards, card)
			continue
		}

		card.HasWallet = true
		card.WalletID = row.ID
		card.Address = row.Address
		card.AddressShort = shortAddress(row.Address)

		balances, err := s.store.GetWalletBalances(ctx, row.ID)
		if err != nil {
			return nil, err
		}

		totalValue := 0.0
		weighted := 0.0
		seenNative := false
		seenUSDT := false

		for _, b := range balances {
			price, change := idx.lookup(meta.Name, b.TokenAddress, b.TokenSymbol)
			value := mulUSD(b.Balance, price)

			view := TokenBalanceView{
				TokenAddress: b.TokenAddress,
				Symbol:       b.TokenSymbol,
				Decimals:     tokenDecimals(tokens, meta.Name, b.TokenAddress, meta.NativeDecimals),
				Balance:      b.Balance,
				PriceUSD:     price,
				ValueUSD:     value,
				Change24h:    change,
				IsNative:     b.TokenAddress == "",
			}
			if name := tokenName(tokens, meta.Name, b.TokenAddress); name != "" {
				view.Name = name
			}

			if b.TokenAddress == "" {
				card.NativeBalance = b.Balance
				seenNative = true
			}
			if strings.EqualFold(b.TokenSymbol, "USDT") {
				card.USDTBalance = b.Balance
				seenUSDT = true
			}

			card.Tokens = append(card.Tokens, view)
			totalValue += toFloat(value)
			weighted += toFloat(value) * change
		}

		if !seenNative {
			card.Tokens = append(card.Tokens, TokenBalanceView{
				Symbol:   meta.NativeSymbol,
				Decimals: int(meta.NativeDecimals),
				Balance:  "0",
				PriceUSD: priceFor(idx, meta.Name, "", meta.NativeSymbol),
				ValueUSD: "0",
				IsNative: true,
			})
		}
		if !seenUSDT {
			address := tokenAddressBySymbol(tokens, meta.Name, "USDT")
			card.Tokens = append(card.Tokens, TokenBalanceView{
				TokenAddress: address,
				Symbol:       "USDT",
				Decimals:     tokenDecimals(tokens, meta.Name, address, 6),
				Balance:      "0",
				PriceUSD:     "1",
				ValueUSD:     "0",
			})
		}

		sortTokenViews(card.Tokens)

		card.TotalValueUSD = format2(totalValue)
		if totalValue > 0 {
			card.Change24h = round2(weighted / totalValue)
		}
		cards = append(cards, card)
	}

	sort.SliceStable(cards, func(i, j int) bool {
		return toFloat(cards[i].TotalValueUSD) > toFloat(cards[j].TotalValueUSD)
	})
	return cards, nil
}

// Addresses mengembalikan dua kolom alamat (EVM universal + Solana) untuk modal.
func (s *Service) Addresses(ctx context.Context) (*AddressesResponse, error) {
	wallets, err := s.store.ListWallets(ctx)
	if err != nil {
		return nil, err
	}

	byNetwork := make(map[string]store.WalletRow)
	for _, w := range wallets {
		if key := canonical(w.Network); key != "" {
			byNetwork[key] = w
		}
	}

	resp := &AddressesResponse{}
	for _, meta := range network.AllMeta() {
		row, ok := byNetwork[meta.Key]
		if !ok {
			continue
		}

		group := &AddressGroup{AddressType: meta.AddressType}
		if meta.Chain == network.ChainSolana {
			if resp.Solana == nil {
				group.Address = row.Address
				group.AddressShort = shortAddress(row.Address)
				group.QRData = row.Address
				resp.Solana = group
			}
			resp.Solana.Networks = append(resp.Solana.Networks, meta.Name)
			resp.Solana.Shorts = append(resp.Solana.Shorts, meta.Short)
			continue
		}

		if resp.EVM == nil {
			group.Address = row.Address
			group.AddressShort = shortAddress(row.Address)
			group.QRData = row.Address
			resp.EVM = group
		}
		resp.EVM.Networks = append(resp.EVM.Networks, meta.Name)
		resp.EVM.Shorts = append(resp.EVM.Shorts, meta.Short)
	}
	return resp, nil
}

// Networks mengembalikan metadata tab jaringan beserta wallet yang ada.
func (s *Service) Networks(ctx context.Context) ([]NetworkTab, error) {
	wallets, err := s.store.ListWallets(ctx)
	if err != nil {
		return nil, err
	}

	byNetwork := make(map[string]store.WalletRow)
	for _, w := range wallets {
		if key := canonical(w.Network); key != "" {
			byNetwork[key] = w
		}
	}

	tabs := make([]NetworkTab, 0, len(network.AllMeta()))
	for _, meta := range network.AllMeta() {
		tab := NetworkTab{
			NetworkKey:    meta.Key,
			Network:       meta.Name,
			DisplayName:   meta.DisplayName,
			Short:         meta.Short,
			TokenStandard: meta.TokenStandard,
			AddressType:   meta.AddressType,
		}
		if row, ok := byNetwork[meta.Key]; ok {
			tab.HasWallet = true
			tab.WalletID = row.ID
			tab.Address = row.Address
			tab.AddressShort = shortAddress(row.Address)
		}
		tabs = append(tabs, tab)
	}
	return tabs, nil
}

// ---------------------------------------------------------------------------
// Rotate (Perbaharui Address)
// ---------------------------------------------------------------------------

// Rotate memperbarui alamat wallet ke alamat baru pada index HD berikutnya.
// Alamat lama tetap valid untuk menerima dana; wallet hanya menampilkan alamat
// baru. Untuk jaringan EVM, keempat wallet EVM dirotasi bersamaan karena
// memakai satu alamat universal.
func (s *Service) Rotate(ctx context.Context, id string) (*RotateResponse, error) {
	row, err := s.store.GetWallet(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, fmt.Errorf("wallet not found")
	}

	meta, ok := network.MetaFor(row.Network)
	if !ok {
		return nil, fmt.Errorf("unsupported network: %s", row.Network)
	}

	info, err := s.privateKeyInfo(meta.Chain)
	if err != nil {
		return nil, err
	}

	wallets, err := s.store.ListWallets(ctx)
	if err != nil {
		return nil, err
	}

	targets := make([]store.WalletRow, 0, 4)
	newIndex := uint32(row.DerivationIndex)
	if meta.Chain == network.ChainSolana {
		targets = append(targets, *row)
	} else {
		for _, w := range wallets {
			if w.Network == "" {
				continue
			}
			m, ok := network.MetaFor(w.Network)
			if !ok || m.Chain != network.ChainEVM {
				continue
			}
			targets = append(targets, w)
			if uint32(w.DerivationIndex) > newIndex {
				newIndex = uint32(w.DerivationIndex)
			}
		}
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("wallet not found")
	}
	newIndex++

	resp := &RotateResponse{
		NetworkKey:      meta.Key,
		AddressType:     meta.AddressType,
		OldAddress:      row.Address,
		DerivationIndex: int(newIndex),
		Note:            "Alamat lama tetap valid untuk menerima dana.",
	}
	if meta.Chain == network.ChainSolana {
		resp.Chain = string(network.ChainSolana)
	} else {
		resp.Chain = string(network.ChainEVM)
	}

	for _, target := range targets {
		address, path, err := wallet.DeriveAddressFromPrivateKey(info, newIndex)
		if err != nil {
			return nil, fmt.Errorf("rotate %s: %w", target.Network, err)
		}
		if err := s.store.UpdateWalletAddress(ctx, target.ID, address, target.Network, path, int(newIndex)); err != nil {
			return nil, fmt.Errorf("rotate %s: %w", target.Network, err)
		}
		resp.Wallets = append(resp.Wallets, RotatedWallet{
			WalletID:        target.ID,
			Network:         target.Network,
			OldAddress:      target.Address,
			NewAddress:      address,
			DerivationIndex: int(newIndex),
		})
		resp.NewAddress = address
	}
	return resp, nil
}

// ---------------------------------------------------------------------------
// Refresh (baca ulang saldo via HTTP RPC)
// ---------------------------------------------------------------------------

// RefreshAll membaca ulang saldo semua wallet via RPC HTTP, menyimpannya, lalu
// mengembalikan overview terbaru.
func (s *Service) RefreshAll(ctx context.Context) (*OverviewResponse, error) {
	if err := s.refresh(ctx, ""); err != nil {
		return nil, err
	}
	return s.Overview(ctx)
}

// RefreshWallet membaca ulang saldo satu wallet lalu mengembalikan detailnya.
func (s *Service) RefreshWallet(ctx context.Context, id string) (*WalletDetailResponse, error) {
	if err := s.refresh(ctx, id); err != nil {
		return nil, err
	}
	return s.GetWallet(ctx, id)
}

func (s *Service) refresh(ctx context.Context, id string) error {
	wallets, err := s.store.ListWallets(ctx)
	if err != nil {
		return err
	}

	var errs []error
	for _, row := range wallets {
		if id != "" && row.ID != id {
			continue
		}
		if err := s.refreshWallet(ctx, row); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", row.Network, err))
		}
	}
	if id != "" && len(errs) == 0 {
		// pastikan wallet dengan id tsb memang ada
		found := false
		for _, row := range wallets {
			if row.ID == id {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("wallet not found")
		}
	}
	return errors.Join(errs...)
}

func (s *Service) refreshWallet(ctx context.Context, row store.WalletRow) error {
	meta, ok := network.MetaFor(row.Network)
	if !ok {
		return fmt.Errorf("jaringan %s tidak dikenal", row.Network)
	}

	config, err := s.store.GetRPCConfig(row.Network)
	if err != nil {
		return err
	}
	if config == nil || config.RPCURL == "" {
		return fmt.Errorf("rpc_url untuk %s belum diisi", row.Network)
	}

	tokens, err := s.store.ListTokens()
	if err != nil {
		return err
	}
	prices, err := s.store.ListTokenPrices(ctx)
	if err != nil {
		return err
	}
	idx := newPriceIndex(prices)

	var rows []store.BalanceRow

	if meta.Chain == network.ChainSolana {
		client := solana.NewClient(config.RPCURL)
		solTokens := solanaTokensFor(tokens, meta.Name)

		lamports, err := client.Balance(ctx, row.Address)
		if err != nil {
			return err
		}
		nativeRaw := new(big.Int).SetUint64(lamports)
		rows = append(rows, store.BalanceRow{
			WalletID:    row.ID,
			Network:     meta.Name,
			TokenSymbol: meta.NativeSymbol,
			Balance:     solana.FormatUnits(nativeRaw, meta.NativeDecimals),
		})

		balances, err := client.TokenBalances(ctx, row.Address, solTokens)
		if err != nil {
			return err
		}
		for _, token := range solTokens {
			raw, _ := new(big.Int).SetString(balances[token.Mint], 10)
			if raw == nil {
				raw = new(big.Int)
			}
			rows = append(rows, store.BalanceRow{
				WalletID:     row.ID,
				Network:      meta.Name,
				TokenAddress: token.Mint,
				TokenSymbol:  token.Symbol,
				Balance:      solana.FormatUnits(raw, token.Decimals),
			})
		}
	} else {
		client := evm.NewClient(config.RPCURL)
		evmTokens := evmTokensFor(tokens, meta.Name)

		result, err := client.BatchBalances(ctx, []string{row.Address}, evmTokens)
		if err != nil {
			return err
		}
		lower := strings.ToLower(row.Address)

		nativeRaw, _ := new(big.Int).SetString(result["native:"+lower], 10)
		if nativeRaw == nil {
			nativeRaw = new(big.Int)
		}
		rows = append(rows, store.BalanceRow{
			WalletID:    row.ID,
			Network:     meta.Name,
			TokenSymbol: meta.NativeSymbol,
			Balance:     evm.FormatUnits(nativeRaw, meta.NativeDecimals),
		})

		for _, token := range evmTokens {
			key := strings.ToLower(token.Address) + ":" + lower
			raw, _ := new(big.Int).SetString(result[key], 10)
			if raw == nil {
				raw = new(big.Int)
			}
			rows = append(rows, store.BalanceRow{
				WalletID:     row.ID,
				Network:      meta.Name,
				TokenAddress: token.Address,
				TokenSymbol:  token.Symbol,
				Balance:      evm.FormatUnits(raw, token.Decimals),
			})
		}
	}

	for i := range rows {
		price, _ := idx.lookup(meta.Name, rows[i].TokenAddress, rows[i].TokenSymbol)
		rows[i].ValueUSD = mulUSD(rows[i].Balance, price)
	}
	return s.store.UpsertBalances(ctx, rows)
}

// ---------------------------------------------------------------------------
// Send
// ---------------------------------------------------------------------------

// SendEstimate menghitung estimasi biaya kirim sebelum konfirmasi.
func (s *Service) SendEstimate(ctx context.Context, id string, req SendEstimateRequest) (*SendEstimateResponse, error) {
	row, meta, _, err := s.sendContext(ctx, id)
	if err != nil {
		return nil, err
	}

	token, err := s.resolveSendToken(ctx, meta, req.TokenAddress)
	if err != nil {
		return nil, err
	}
	if err := validateAddress(meta.Chain, req.To); err != nil {
		return nil, err
	}

	amount, err := parseUnits(req.Amount, token.decimals)
	if err != nil {
		return nil, err
	}
	if amount.Sign() <= 0 {
		return nil, fmt.Errorf("jumlah harus lebih dari 0")
	}

	config, err := s.store.GetRPCConfig(row.Network)
	if err != nil {
		return nil, err
	}
	if config == nil || config.RPCURL == "" {
		return nil, fmt.Errorf("rpc_url untuk %s belum diisi", row.Network)
	}

	prices, _ := s.store.ListTokenPrices(ctx)
	idx := newPriceIndex(prices)
	nativePrice, _ := idx.lookup(meta.Name, "", meta.NativeSymbol)

	resp := &SendEstimateResponse{
		Network:      meta.Name,
		NetworkKey:   meta.Key,
		TokenSymbol:  token.symbol,
		TokenAddress: token.address,
		Amount:       req.Amount,
		NativeSymbol: meta.NativeSymbol,
		FeeNative:    "0",
		FeeUSD:       "0",
		TotalUSD:     mulUSD(req.Amount, priceFor(idx, meta.Name, token.address, token.symbol)),
	}

	if meta.Chain == network.ChainSolana {
		fee := big.NewInt(5000) // biaya dasar satu signature
		feeNative := solana.FormatUnits(fee, meta.NativeDecimals)
		resp.FeeNative = feeNative
		resp.FeeUSD = mulUSD(feeNative, nativePrice)
		return resp, nil
	}

	client := evm.NewClient(config.RPCURL)
	gasPrice, err := client.GasPrice(ctx)
	if err != nil {
		return nil, err
	}

	gasLimit := uint64(21000)
	call := map[string]string{
		"from":  row.Address,
		"to":    common.HexToAddress(req.To).Hex(),
		"value": "0x" + amount.Text(16),
	}
	if token.address == "" {
		call["to"] = common.HexToAddress(req.To).Hex()
	} else {
		data, err := evm.EncodeERC20Transfer(common.HexToAddress(req.To), amount)
		if err != nil {
			return nil, err
		}
		call["to"] = token.address
		call["value"] = "0x0"
		call["data"] = "0x" + hex.EncodeToString(data)
	}

	estimated, err := client.EstimateGas(ctx, call)
	if err == nil && estimated > 0 {
		gasLimit = estimated
	} else if token.address != "" {
		gasLimit = 65000
	}

	fee := new(big.Int).Mul(new(big.Int).SetUint64(gasLimit), new(big.Int).SetUint64(gasPrice))
	feeNative := evm.FormatUnits(fee, meta.NativeDecimals)

	resp.GasLimit = gasLimit
	resp.GasPrice = strconv.FormatUint(gasPrice, 10)
	resp.FeeNative = feeNative
	resp.FeeUSD = mulUSD(feeNative, nativePrice)
	return resp, nil
}

// Send membangun, menandatangani, dan menyiarkan transaksi transfer.
func (s *Service) Send(ctx context.Context, id string, req SendRequest) (*SendResponse, error) {
	row, meta, info, err := s.sendContext(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.checkPin(ctx, req.Pin); err != nil {
		return nil, err
	}

	token, err := s.resolveSendToken(ctx, meta, req.TokenAddress)
	if err != nil {
		return nil, err
	}
	if err := validateAddress(meta.Chain, req.To); err != nil {
		return nil, err
	}

	amount, err := parseUnits(req.Amount, token.decimals)
	if err != nil {
		return nil, err
	}
	if amount.Sign() <= 0 {
		return nil, fmt.Errorf("jumlah harus lebih dari 0")
	}

	key, err := wallet.PrivateKeyAtIndex(info, uint32(row.DerivationIndex))
	if err != nil {
		return nil, err
	}

	config, err := s.store.GetRPCConfig(row.Network)
	if err != nil {
		return nil, err
	}
	if config == nil || config.RPCURL == "" {
		return nil, fmt.Errorf("rpc_url untuk %s belum diisi", row.Network)
	}

	resp := &SendResponse{
		Network:     meta.Name,
		NetworkKey:  meta.Key,
		TokenSymbol: token.symbol,
		Amount:      req.Amount,
		To:          req.To,
		Status:      "pending",
	}

	if meta.Chain == network.ChainSolana {
		client := solana.NewClient(config.RPCURL)
		blockhash, err := client.LatestBlockhash(ctx)
		if err != nil {
			return nil, err
		}

		var txBase64 string
		if token.address == "" {
			txBase64, err = solana.SignNativeTransfer(key.RawBytes, row.Address, req.To, amount.Uint64(), blockhash)
		} else {
			source, err := solana.TokenAccountAddress(row.Address, token.address)
			if err != nil {
				return nil, err
			}
			dest, err := solana.TokenAccountAddress(req.To, token.address)
			if err != nil {
				return nil, err
			}
			txBase64, err = solana.SignSPLTransfer(key.RawBytes, row.Address, source, dest, amount.Uint64(), blockhash)
		}
		if err != nil {
			return nil, err
		}

		signature, err := client.SendTransaction(ctx, txBase64)
		if err != nil {
			return nil, err
		}
		resp.TxHash = signature
		resp.ExplorerURL = meta.ExplorerURL + "/tx/" + signature
		return resp, nil
	}

	client := evm.NewClient(config.RPCURL)
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	nonce, err := client.Nonce(ctx, row.Address)
	if err != nil {
		return nil, err
	}
	gasPrice, err := client.GasPrice(ctx)
	if err != nil {
		return nil, err
	}

	gasLimit := uint64(21000)
	var raw string
	if token.address == "" {
		raw, err = evm.SignTransfer(
			key.RawBytes,
			big.NewInt(chainID),
			nonce,
			common.HexToAddress(req.To),
			amount,
			gasLimit,
			new(big.Int).SetUint64(gasPrice),
		)
	} else {
		data, encErr := evm.EncodeERC20Transfer(common.HexToAddress(req.To), amount)
		if encErr != nil {
			return nil, encErr
		}
		estimated, estErr := client.EstimateGas(ctx, map[string]string{
			"from": row.Address,
			"to":   token.address,
			"data": "0x" + hex.EncodeToString(data),
		})
		if estErr == nil && estimated > 0 {
			gasLimit = estimated
		} else {
			gasLimit = 65000
		}
		raw, err = evm.SignERC20Transfer(
			key.RawBytes,
			big.NewInt(chainID),
			nonce,
			common.HexToAddress(token.address),
			common.HexToAddress(req.To),
			amount,
			gasLimit,
			new(big.Int).SetUint64(gasPrice),
		)
	}
	if err != nil {
		return nil, err
	}

	hash, err := client.SendRawTransaction(ctx, raw)
	if err != nil {
		return nil, err
	}
	resp.TxHash = hash
	resp.ExplorerURL = meta.ExplorerURL + "/tx/" + hash
	return resp, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type sendToken struct {
	address  string
	symbol   string
	decimals uint8
}

func (s *Service) sendContext(ctx context.Context, id string) (store.WalletRow, network.Meta, *wallet.PrivateKeyInfo, error) {
	row, err := s.store.GetWallet(ctx, id)
	if err != nil {
		return store.WalletRow{}, network.Meta{}, nil, err
	}
	if row == nil {
		return store.WalletRow{}, network.Meta{}, nil, fmt.Errorf("wallet not found")
	}
	meta, ok := network.MetaFor(row.Network)
	if !ok {
		return store.WalletRow{}, network.Meta{}, nil, fmt.Errorf("unsupported network: %s", row.Network)
	}
	info, err := s.privateKeyInfo(meta.Chain)
	if err != nil {
		return store.WalletRow{}, network.Meta{}, nil, err
	}
	return *row, meta, info, nil
}

func (s *Service) privateKeyInfo(chain network.Chain) (*wallet.PrivateKeyInfo, error) {
	if chain == network.ChainSolana {
		if strings.TrimSpace(s.cfg.SolanaPrivateKey) == "" {
			return nil, fmt.Errorf("SOLANA_PRIVATE_KEY belum diisi di env")
		}
		return wallet.ParseSolanaPrivateKey(s.cfg.SolanaPrivateKey)
	}
	if strings.TrimSpace(s.cfg.ETHPrivateKey) == "" {
		return nil, fmt.Errorf("ETH_PRIVATE_KEY belum diisi di env")
	}
	return wallet.ParseEVMPrivateKey(s.cfg.ETHPrivateKey)
}

func (s *Service) resolveSendToken(ctx context.Context, meta network.Meta, tokenAddress string) (sendToken, error) {
	if strings.TrimSpace(tokenAddress) == "" {
		return sendToken{address: "", symbol: meta.NativeSymbol, decimals: meta.NativeDecimals}, nil
	}

	tokens, err := s.store.ListTokens()
	if err != nil {
		return sendToken{}, err
	}
	for _, t := range tokens {
		if canonical(t.Network) != meta.Key {
			continue
		}
		if strings.EqualFold(t.Address, tokenAddress) {
			return sendToken{address: t.Address, symbol: t.Symbol, decimals: uint8(t.Decimals)}, nil
		}
	}
	return sendToken{}, fmt.Errorf("token %s tidak dikenal di %s", tokenAddress, meta.Name)
}

func (s *Service) checkPin(ctx context.Context, pin string) error {
	settings, err := s.store.GetSecuritySettings(ctx)
	if err != nil {
		return err
	}
	if !settings.PinEnabled {
		return nil
	}
	if strings.TrimSpace(pin) == "" {
		return fmt.Errorf("PIN diperlukan untuk mengirim")
	}
	if hashPIN(pin) != settings.PinHash {
		return fmt.Errorf("PIN salah")
	}
	return nil
}

func validateAddress(chain network.Chain, address string) error {
	if strings.TrimSpace(address) == "" {
		return fmt.Errorf("alamat tujuan wajib diisi")
	}
	if chain == network.ChainSolana {
		if !solana.ValidAddress(address) {
			return fmt.Errorf("alamat Solana tidak valid")
		}
		return nil
	}
	if !common.IsHexAddress(address) {
		return fmt.Errorf("alamat EVM tidak valid")
	}
	return nil
}

func solanaTokensFor(tokens []store.TokenRow, networkName string) []solana.Token {
	key := canonical(networkName)
	var out []solana.Token
	for _, t := range tokens {
		if canonical(t.Network) != key {
			continue
		}
		out = append(out, solana.Token{Mint: t.Address, Symbol: t.Symbol, Decimals: uint8(t.Decimals)})
	}
	return out
}

func evmTokensFor(tokens []store.TokenRow, networkName string) []evm.Token {
	key := canonical(networkName)
	var out []evm.Token
	for _, t := range tokens {
		if canonical(t.Network) != key {
			continue
		}
		out = append(out, evm.Token{Address: t.Address, Symbol: t.Symbol, Decimals: uint8(t.Decimals)})
	}
	return out
}

func tokenDecimals(tokens []store.TokenRow, networkName, tokenAddress string, fallback uint8) int {
	if tokenAddress == "" {
		return int(fallback)
	}
	key := canonical(networkName)
	for _, t := range tokens {
		if canonical(t.Network) == key && strings.EqualFold(t.Address, tokenAddress) {
			return t.Decimals
		}
	}
	return 0
}

func tokenName(tokens []store.TokenRow, networkName, tokenAddress string) string {
	key := canonical(networkName)
	for _, t := range tokens {
		if canonical(t.Network) == key && strings.EqualFold(t.Address, tokenAddress) {
			return t.Name
		}
	}
	return ""
}

func tokenAddressBySymbol(tokens []store.TokenRow, networkName, symbol string) string {
	key := canonical(networkName)
	for _, t := range tokens {
		if canonical(t.Network) == key && strings.EqualFold(t.Symbol, symbol) {
			return t.Address
		}
	}
	return ""
}

func priceFor(idx *priceIndex, networkName, tokenAddress, symbol string) string {
	price, _ := idx.lookup(networkName, tokenAddress, symbol)
	return price
}

func sortTokenViews(views []TokenBalanceView) {
	sort.SliceStable(views, func(i, j int) bool {
		pi, pj := tokenPriority(views[i]), tokenPriority(views[j])
		if pi != pj {
			return pi < pj
		}
		return toFloat(views[i].ValueUSD) > toFloat(views[j].ValueUSD)
	})
}

func tokenPriority(v TokenBalanceView) int {
	switch {
	case v.IsNative:
		return 0
	case strings.EqualFold(v.Symbol, "USDT"):
		return 1
	default:
		return 2
	}
}

func shortAddress(address string) string {
	address = strings.TrimSpace(address)
	if len(address) <= 12 {
		return address
	}
	return address[:6] + "..." + address[len(address)-4:]
}

func parseUnits(amount string, decimals uint8) (*big.Int, error) {
	value, ok := new(big.Rat).SetString(strings.TrimSpace(amount))
	if !ok || value.Sign() < 0 {
		return nil, fmt.Errorf("jumlah tidak valid: %s", amount)
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	value.Mul(value, new(big.Rat).SetInt(scale))
	if !value.IsInt() {
		return nil, fmt.Errorf("jumlah punya terlalu banyak desimal")
	}
	return new(big.Int).Set(value.Num()), nil
}

func mulUSD(a, b string) string {
	ra, ok1 := new(big.Rat).SetString(a)
	rb, ok2 := new(big.Rat).SetString(b)
	if !ok1 || !ok2 {
		return "0"
	}
	return format2Rat(new(big.Rat).Mul(ra, rb))
}

func addUSD(a, b string) string {
	ra, ok1 := new(big.Rat).SetString(a)
	if !ok1 {
		ra = new(big.Rat)
	}
	rb, ok2 := new(big.Rat).SetString(b)
	if !ok2 {
		rb = new(big.Rat)
	}
	return format2Rat(new(big.Rat).Add(ra, rb))
}

func format2Rat(r *big.Rat) string {
	return new(big.Float).SetRat(r).Text('f', 2)
}

func format2(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}

func round2(f float64) float64 {
	return mathRound(f*100) / 100
}

func mathRound(f float64) float64 {
	if f < 0 {
		return float64(int64(f - 0.5))
	}
	return float64(int64(f + 0.5))
}

func toFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

// hashPIN menyimpan PIN sebagai SHA-256 hex (bukan plaintext). Ini hanya
// pengaman dasar; PIN bukan pengganti enkripsi private key.
func hashPIN(pin string) string {
	sum := sha256.Sum256([]byte(pin))
	return hex.EncodeToString(sum[:])
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

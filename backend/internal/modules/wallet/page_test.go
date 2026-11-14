package wallet

import (
	"context"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"hotwallet/backend/config"
	"hotwallet/backend/internal/blockchain/network"
	"hotwallet/backend/internal/infrastructure/store"
)

func newPageTestService(t *testing.T, cfg config.Config) (*Service, *store.Store) {
	t.Helper()

	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	if err := store.InitSchema(st.DB()); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	return NewService(st, cfg), st
}

func randomEVMKey(t *testing.T) string {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return "0x" + hex.EncodeToString(crypto.FromECDSA(key))
}

const (
	ethUSDTAddress = "0xdAC17F958D2ee523a2206206994597C13D831ec7"
	bscUSDTAddress = "0x55d398326f99059fF775485246999027B3197955"
)

func TestOverviewAggregatesAndSortsCards(t *testing.T) {
	svc, st := newPageTestService(t, config.Config{})
	ctx := context.Background()

	if err := st.SeedTokens([]store.TokenRow{
		{Network: "Ethereum Mainnet", Address: ethUSDTAddress, Symbol: "USDT", Name: "Tether USD", Decimals: 6},
		{Network: "BNB Chain", Address: bscUSDTAddress, Symbol: "USDT", Name: "Tether USD (BSC)", Decimals: 18},
	}); err != nil {
		t.Fatalf("seed tokens: %v", err)
	}
	if err := st.UpsertTokenPrices(ctx, []store.TokenPrice{
		{Network: "Ethereum Mainnet", Symbol: "ETH", PriceUSD: "3000", Change24h: 2},
		{Network: "BNB Chain", Symbol: "BNB", PriceUSD: "500", Change24h: -1},
	}); err != nil {
		t.Fatalf("seed prices: %v", err)
	}

	if err := st.CreateWallet(ctx, &store.WalletRow{
		ID: "eth-test", Name: "Ethereum Wallet",
		Address: "0x1Be31A94361a391bBaFB2a4CCd704F57dc04d4bb", Network: "Ethereum Mainnet",
	}); err != nil {
		t.Fatalf("create eth: %v", err)
	}
	if err := st.CreateWallet(ctx, &store.WalletRow{
		ID: "bsc-test", Name: "BNB Wallet",
		Address: "0x1Be31A94361a391bBaFB2a4CCd704F57dc04d4bb", Network: "BNB Chain",
	}); err != nil {
		t.Fatalf("create bsc: %v", err)
	}

	// ETH: 2 * 3000 = 6000, USDT 1000 (fallback $1) -> 7000. BNB: 4 * 500 = 2000.
	if err := st.UpsertBalances(ctx, []store.BalanceRow{
		{WalletID: "eth-test", Network: "Ethereum Mainnet", TokenSymbol: "ETH", Balance: "2"},
		{WalletID: "eth-test", Network: "Ethereum Mainnet", TokenAddress: ethUSDTAddress, TokenSymbol: "USDT", Balance: "1000"},
		{WalletID: "bsc-test", Network: "BNB Chain", TokenSymbol: "BNB", Balance: "4"},
	}); err != nil {
		t.Fatalf("upsert balances: %v", err)
	}

	overview, err := svc.Overview(ctx)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}

	if len(overview.Networks) != len(network.AllMeta()) {
		t.Fatalf("jumlah kartu = %d, mau %d", len(overview.Networks), len(network.AllMeta()))
	}
	if overview.TotalValueUSD != "9000.00" {
		t.Fatalf("total = %s, mau 9000.00", overview.TotalValueUSD)
	}

	// Kartu diurutkan dari nilai USD terbesar: Ethereum (7000) lalu BNB (2000).
	if overview.Networks[0].NetworkKey != "ethereum" || overview.Networks[0].TotalValueUSD != "7000.00" {
		t.Fatalf("kartu pertama = %s/%s, mau ethereum/7000.00",
			overview.Networks[0].NetworkKey, overview.Networks[0].TotalValueUSD)
	}
	if overview.Networks[1].NetworkKey != "bsc" || overview.Networks[1].TotalValueUSD != "2000.00" {
		t.Fatalf("kartu kedua = %s/%s, mau bsc/2000.00",
			overview.Networks[1].NetworkKey, overview.Networks[1].TotalValueUSD)
	}

	eth := overview.Networks[0]
	if eth.USDTBalance != "1000" || eth.NativeBalance != "2" {
		t.Fatalf("saldo ETH: native=%s usdt=%s, mau 2 / 1000", eth.NativeBalance, eth.USDTBalance)
	}
	if !eth.HasWallet || eth.WalletID != "eth-test" || eth.TokenStandard != "ERC-20" {
		t.Fatalf("metadata kartu ETH tidak lengkap: %+v", eth)
	}
	// Native harus tampil sebelum USDT pada daftar token.
	if len(eth.Tokens) < 2 || !eth.Tokens[0].IsNative || !strings.EqualFold(eth.Tokens[1].Symbol, "USDT") {
		t.Fatalf("urutan token salah: %+v", eth.Tokens)
	}

	// Jaringan tanpa wallet tetap muncul sebagai kartu kosong.
	last := overview.Networks[len(overview.Networks)-1]
	if last.HasWallet || last.TotalValueUSD != "0.00" {
		t.Fatalf("kartu tanpa wallet = %+v, mau kosong", last)
	}
}

func TestAddressesGroupsEVMAndSolana(t *testing.T) {
	svc, st := newPageTestService(t, config.Config{ETHPrivateKey: randomEVMKey(t)})
	ctx := context.Background()

	if err := svc.AutoCreateFromEnv(ctx); err != nil {
		t.Fatalf("auto create: %v", err)
	}

	addresses, err := svc.Addresses(ctx)
	if err != nil {
		t.Fatalf("addresses: %v", err)
	}
	if addresses.EVM == nil {
		t.Fatal("grup EVM kosong")
	}
	if addresses.Solana != nil {
		t.Fatal("grup Solana seharusnya kosong tanpa SOLANA_PRIVATE_KEY")
	}
	if len(addresses.EVM.Networks) != len(network.EVMKeys()) {
		t.Fatalf("grup EVM punya %d jaringan, mau %d", len(addresses.EVM.Networks), len(network.EVMKeys()))
	}
	if addresses.EVM.QRData == "" || addresses.EVM.Address == "" {
		t.Fatalf("grup EVM tidak lengkap: %+v", addresses.EVM)
	}
	if addresses.EVM.AddressType != network.AddressTypeEVM {
		t.Fatalf("address type = %s, mau %s", addresses.EVM.AddressType, network.AddressTypeEVM)
	}

	rows, err := st.ListWallets(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, row := range rows {
		if !strings.EqualFold(row.Address, addresses.EVM.Address) {
			t.Fatalf("alamat EVM tidak seragam: %s != %s", row.Address, addresses.EVM.Address)
		}
	}
}

func TestRotateAdvancesAllEVMWallets(t *testing.T) {
	svc, st := newPageTestService(t, config.Config{ETHPrivateKey: randomEVMKey(t)})
	ctx := context.Background()

	if err := svc.AutoCreateFromEnv(ctx); err != nil {
		t.Fatalf("auto create: %v", err)
	}

	before, err := st.ListWallets(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(before) != 4 {
		t.Fatalf("jumlah wallet = %d, mau 4", len(before))
	}
	oldAddress := before[0].Address

	resp, err := svc.Rotate(ctx, before[0].ID)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if resp.NewAddress == "" || strings.EqualFold(resp.NewAddress, oldAddress) {
		t.Fatalf("alamat baru = %s, harus beda dari %s", resp.NewAddress, oldAddress)
	}
	if resp.DerivationIndex != 1 {
		t.Fatalf("derivation index = %d, mau 1", resp.DerivationIndex)
	}
	if len(resp.Wallets) != 4 {
		t.Fatalf("wallet yang dirotasi = %d, mau 4 (semua EVM)", len(resp.Wallets))
	}

	after, err := st.ListWallets(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, row := range after {
		if !strings.EqualFold(row.Address, resp.NewAddress) {
			t.Fatalf("%s: alamat = %s, mau %s", row.Network, row.Address, resp.NewAddress)
		}
		if row.DerivationIndex != 1 {
			t.Fatalf("%s: index = %d, mau 1", row.Network, row.DerivationIndex)
		}
	}

	// Rotate lagi harus menaikkan index dan mengubah alamat lagi.
	second, err := svc.Rotate(ctx, before[0].ID)
	if err != nil {
		t.Fatalf("rotate kedua: %v", err)
	}
	if second.DerivationIndex != 2 || strings.EqualFold(second.NewAddress, resp.NewAddress) {
		t.Fatalf("rotate kedua = index %d alamat %s, mau index 2 alamat berbeda",
			second.DerivationIndex, second.NewAddress)
	}
}

func TestNetworksReturnsTabsInUIOrder(t *testing.T) {
	svc, _ := newPageTestService(t, config.Config{ETHPrivateKey: randomEVMKey(t)})
	ctx := context.Background()

	if err := svc.AutoCreateFromEnv(ctx); err != nil {
		t.Fatalf("auto create: %v", err)
	}

	tabs, err := svc.Networks(ctx)
	if err != nil {
		t.Fatalf("networks: %v", err)
	}

	wantOrder := []string{"ethereum", "bsc", "arbitrum", "robinhood", "solana"}
	if len(tabs) != len(wantOrder) {
		t.Fatalf("jumlah tab = %d, mau %d", len(tabs), len(wantOrder))
	}
	for i, key := range wantOrder {
		if tabs[i].NetworkKey != key {
			t.Fatalf("tab %d = %s, mau %s", i, tabs[i].NetworkKey, key)
		}
	}
	if tabs[4].HasWallet {
		t.Fatal("tab Solana seharusnya belum punya wallet")
	}
	if tabs[0].TokenStandard != "ERC-20" || tabs[1].TokenStandard != "BEP-20" {
		t.Fatalf("badge standar token salah: %s / %s", tabs[0].TokenStandard, tabs[1].TokenStandard)
	}
}

package store

import (
	"context"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"hotwallet/backend/config"
	"hotwallet/backend/internal/blockchain/wallet"
)

func newWalletTestStore(t *testing.T) *Store {
	t.Helper()

	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	if err := InitSchema(st.DB()); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	return st
}

func randomPrivateKeyHex(t *testing.T) string {
	t.Helper()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return "0x" + hex.EncodeToString(crypto.FromECDSA(key))
}

func TestAutoCreateWalletFromEnvSyncsAfterKeyChange(t *testing.T) {
	st := newWalletTestStore(t)
	ctx := context.Background()
	cfg := config.Config{ETHPrivateKey: randomPrivateKeyHex(t)}

	if err := st.AutoCreateWalletFromEnv(ctx, cfg); err != nil {
		t.Fatalf("auto create: %v", err)
	}

	before, err := st.ListWallets(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(before) != 4 {
		t.Fatalf("jumlah wallet = %d, mau 4 (satu per jaringan EVM)", len(before))
	}

	oldAddress := before[0].Address
	for _, row := range before {
		if !strings.EqualFold(row.Address, oldAddress) {
			t.Fatalf("alamat EVM harus sama: %s != %s", row.Address, oldAddress)
		}
	}
	if err := st.UpdateWalletBalance(ctx, before[0].ID, "123"); err != nil {
		t.Fatalf("set balance: %v", err)
	}

	// Ganti private key, alamat harus ikut berubah tanpa menambah wallet baru.
	cfg.ETHPrivateKey = randomPrivateKeyHex(t)
	if err := st.AutoCreateWalletFromEnv(ctx, cfg); err != nil {
		t.Fatalf("sync key baru: %v", err)
	}

	after, err := st.ListWallets(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(after) != 4 {
		t.Fatalf("jumlah wallet = %d, mau tetap 4", len(after))
	}

	info, err := wallet.ParseEVMPrivateKey(cfg.ETHPrivateKey)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	for _, row := range after {
		if !strings.EqualFold(row.Address, info.Address) {
			t.Fatalf("%s: alamat = %s, mau %s", row.Network, row.Address, info.Address)
		}
		if row.Balance != "0" {
			t.Fatalf("%s: saldo lama harus direset, dapat %s", row.Network, row.Balance)
		}
	}
}

func TestAutoCreateWalletFromEnvSkipsNetworkWithMultipleWallets(t *testing.T) {
	st := newWalletTestStore(t)
	ctx := context.Background()
	cfg := config.Config{ETHPrivateKey: randomPrivateKeyHex(t)}

	if err := st.AutoCreateWalletFromEnv(ctx, cfg); err != nil {
		t.Fatalf("auto create: %v", err)
	}

	// Wallet kedua di BNB Chain (bukan dari env) membuat jaringan itu ambigu.
	manual := &WalletRow{
		ID:      "bsc-manual",
		Name:    "Manual",
		Address: "0x0000000000000000000000000000000000000001",
		Network: "BNB Chain",
		Balance: "0",
	}
	if err := st.CreateWallet(ctx, manual); err != nil {
		t.Fatalf("create manual: %v", err)
	}

	cfg.ETHPrivateKey = randomPrivateKeyHex(t)
	err := st.AutoCreateWalletFromEnv(ctx, cfg)
	if err == nil || !strings.Contains(err.Error(), "BNB Chain") {
		t.Fatalf("mau error soal BNB Chain, dapat: %v", err)
	}

	rows, err := st.ListWallets(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("jumlah wallet = %d, mau 5 (tidak ada wallet baru)", len(rows))
	}

	info, _ := wallet.ParseEVMPrivateKey(cfg.ETHPrivateKey)
	for _, row := range rows {
		if row.Network == "BNB Chain" && strings.EqualFold(row.Address, info.Address) {
			t.Fatal("wallet BNB ganda seharusnya tidak diubah otomatis")
		}
	}
}

package main

import (
	"context"
	"log"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"hotwallet/backend/config"
	"hotwallet/backend/internal/blockchain/network"
	"hotwallet/backend/internal/blockchain/tokenmeta"
	"hotwallet/backend/internal/blockchain/wallet"
	"hotwallet/backend/internal/blockchain/watcher"
	"hotwallet/backend/internal/infrastructure/router"
	"hotwallet/backend/internal/infrastructure/store"
)

func main() {
	cfg := config.Load()

	// Identitas wallet: seed phrase BIP39. Kalau belum ada di environment,
	// buat 12 kata acak dan simpan ke .env supaya alamat tetap sama setelah
	// restart. File .env tidak ikut git.
	if cfg.WalletMnemonic == "" {
		mnemonic, err := wallet.NewMnemonic(12)
		if err != nil {
			log.Fatalf("wallet: %v", err)
		}
		envPath := filepath.Join(cfg.Root, ".env")
		if err := config.SaveEnvValue(envPath, "WALLET_MNEMONIC", mnemonic); err != nil {
			log.Printf("warning: simpan seed phrase ke .env: %v", err)
		}
		cfg.WalletMnemonic = mnemonic
		log.Printf("seed phrase wallet dibuat dan disimpan di %s", envPath)
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer st.Close()

	// Inisialisasi schema database
	if err := store.InitSchema(st.DB()); err != nil {
		log.Fatalf("database schema: %v", err)
	}

	// Insert default settings
	if err := store.InsertDefaultSettings(st.DB()); err != nil {
		log.Printf("warning: default settings: %v", err)
	}

	// Seed endpoint RPC/WSS dari env. Nilai yang sudah diubah lewat aplikasi
	// (tabel rpc_configs) tidak akan ditimpa.
	if err := st.SeedRPCConfigs(rpcSeeds(cfg)); err != nil {
		log.Printf("warning: seed rpc: %v", err)
	}
	if err := st.SeedTokens(tokenSeeds()); err != nil {
		log.Printf("warning: seed token: %v", err)
	}

	app := router.New(cfg, st)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Watcher saldo real-time: satu koneksi WebSocket per jaringan, hasilnya
	// disimpan ke SQLite (wallet_balances, wallets.balance, network_summaries).
	if cfg.WSWatcherEnabled {
		manager := watcher.NewManager(st, watcher.Config{
			Debounce: cfg.WSDebounce,
			Refresh:  cfg.WSRefresh,
		})
		go func() {
			if err := manager.Run(ctx); err != nil && ctx.Err() == nil {
				log.Printf("watcher: %v", err)
			}
		}()
	}

	log.Printf("HotWallet API listening on http://localhost:%s", cfg.Port)
	log.Printf("  root    : %s", cfg.Root)
	log.Printf("  database: %s", cfg.DBPath)

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Listen(":" + cfg.Port)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			log.Fatalf("server: %v", err)
		}
	case <-ctx.Done():
		log.Println("shutting down...")
		if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}
}

// rpcSeeds membangun nilai awal rpc_configs dari env, tanpa meng-hardcode URL
// di source. Nama jaringan berasal dari network.Defaults().
func rpcSeeds(cfg config.Config) []store.RPCSeed {
	seeds := make([]store.RPCSeed, 0, len(network.Defaults()))
	for _, def := range network.Defaults() {
		key := network.Canonical(def.Name)
		seeds = append(seeds, store.RPCSeed{
			Network:     def.Name,
			RPCURL:      cfg.RPCEndpoints[key],
			WSSURL:      cfg.WSSEndpoints[key],
			ChainID:     def.ChainID,
			ExplorerURL: def.ExplorerURL,
		})
	}
	return seeds
}

func tokenSeeds() []store.TokenRow {
	defaults := tokenmeta.Defaults()
	rows := make([]store.TokenRow, 0, len(defaults))
	for _, token := range defaults {
		rows = append(rows, store.TokenRow{
			Network:  token.Network,
			Address:  token.Address,
			Symbol:   token.Symbol,
			Name:     token.Name,
			Decimals: token.Decimals,
		})
	}
	return rows
}

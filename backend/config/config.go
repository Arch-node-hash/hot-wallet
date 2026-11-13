package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config menyimpan konfigurasi dasar server.
// Tambahkan field baru di sini bila dibutuhkan.
type Config struct {
	Port    string
	DataDir string
	DBPath  string
	Root    string

	// Blockchain configuration
	DefaultNetwork string
	EnableTestnets bool

	// Security
	EncryptionKey string

	// Wallet identity (BIP39 seed phrase), diisi manual di .env.
	WalletMnemonic string

	// Backup
	BackupDir string

	// RPC endpoints (can be overridden by database)
	RPCEndpoints map[string]string
	WSSEndpoints map[string]string

	// Real-time balance watcher (WebSocket)
	WSWatcherEnabled bool
	WSDebounce       time.Duration
	WSRefresh        time.Duration
}

func Load() Config {
	root := findBackendRoot()

	// File .env lokal memudahkan development tanpa dependency tambahan.
	// Environment variable asli selalu menang.
	loadDotEnv(filepath.Join(root, ".env"))

	dataDir := resolve(root, env("DATA_DIR", "data"))
	dbPath := env("DB_PATH", "")
	if dbPath == "" {
		dbPath = filepath.Join(dataDir, "hotwallet.db")
	} else {
		dbPath = resolve(root, dbPath)
	}

	// Initialize RPC endpoints
	rpcEndpoints := map[string]string{
		"ethereum":  env("ETHEREUM_RPC", "https://ethereum-rpc.publicnode.com"),
		"bsc":       env("BSC_RPC", "https://bsc-rpc.publicnode.com"),
		"arbitrum":  env("ARBITRUM_RPC", "https://arbitrum-one-rpc.publicnode.com"),
		"robinhood": env("ROBINHOOD_RPC", "https://rpc.mainnet.chain.robinhood.com"),
		"solana":    env("SOLANA_RPC", ""),
		"polygon":   env("POLYGON_RPC", "https://polygon-rpc.com"),
	}

	// Endpoint WebSocket TIDAK di-hardcode di source. Isi lewat env (atau
	// rpc_configs di database). Kalau kosong, watcher untuk chain tsb tidak
	// dijalankan.
	wssEndpoints := map[string]string{}
	for _, key := range []string{"ethereum", "bsc", "arbitrum", "robinhood", "solana"} {
		if value := os.Getenv(strings.ToUpper(key) + "_WSS"); value != "" {
			wssEndpoints[key] = value
		}
	}

	return Config{
		Port:             env("PORT", "5009"),
		DataDir:          dataDir,
		DBPath:           filepath.Clean(dbPath),
		Root:             root,
		DefaultNetwork:   env("DEFAULT_NETWORK", "ethereum"),
		EnableTestnets:   env("ENABLE_TESTNETS", "false") == "true",
		EncryptionKey:    env("ENCRYPTION_KEY", ""),
		WalletMnemonic:   env("WALLET_MNEMONIC", ""),
		BackupDir:        filepath.Join(dataDir, "backups"),
		RPCEndpoints:     rpcEndpoints,
		WSSEndpoints:     wssEndpoints,
		WSWatcherEnabled: env("WS_WATCHER_ENABLED", "true") == "true",
		WSDebounce:       envDuration("WS_DEBOUNCE", 4*time.Second),
		WSRefresh:        envDuration("WS_REFRESH", 30*time.Second),
	}
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

// findBackendRoot mencari direktori yang berisi go.mod supaya server bisa
// dijalankan dari backend/, backend/cmd/api/, atau root project.
func findBackendRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	dir := cwd
	for {
		if isFile(filepath.Join(dir, "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	candidate := filepath.Join(cwd, "backend")
	if isFile(filepath.Join(candidate, "go.mod")) {
		return candidate
	}

	return cwd
}

func resolve(root, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Clean(filepath.Join(root, value))
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)

		if key == "" || os.Getenv(key) != "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return
		}
	}
}

package watcher

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"hotwallet/backend/internal/blockchain/evm"
	"hotwallet/backend/internal/blockchain/network"
	"hotwallet/backend/internal/blockchain/solana"
	"hotwallet/backend/internal/infrastructure/store"
)

// Config mengatur perilaku watcher.
type Config struct {
	Debounce time.Duration
	Refresh  time.Duration
	Logf     func(string, ...any)
}

// Manager menjalankan satu watcher WebSocket per jaringan yang punya wallet,
// lalu menyimpan seluruh saldo yang diterima ke database.
type Manager struct {
	store *store.Store
	cfg   Config

	mu         sync.RWMutex
	targets    []store.WalletTarget
	tokens     []store.TokenRow
	rpcConfigs map[string]store.RPCConfig

	defs   map[string]network.Network
	active map[string]context.CancelFunc
	sigs   map[string]string
	warned map[string]bool
}

func NewManager(st *store.Store, cfg Config) *Manager {
	defs := make(map[string]network.Network)
	for _, n := range network.Defaults() {
		if key := network.Canonical(n.Name); key != "" {
			defs[key] = n
		}
	}

	if cfg.Debounce <= 0 {
		cfg.Debounce = 4 * time.Second
	}
	if cfg.Refresh <= 0 {
		cfg.Refresh = 30 * time.Second
	}
	if cfg.Logf == nil {
		cfg.Logf = log.Printf
	}

	return &Manager{
		store:      st,
		cfg:        cfg,
		rpcConfigs: make(map[string]store.RPCConfig),
		defs:       defs,
		active:     make(map[string]context.CancelFunc),
		sigs:       make(map[string]string),
		warned:     make(map[string]bool),
	}
}

// Run memuat wallet secara berkala lalu menyalakan/mematikan watcher sesuai
// kebutuhan. Berhenti saat ctx selesai.
func (m *Manager) Run(ctx context.Context) error {
	m.reload()
	if err := m.sync(ctx); err != nil {
		m.logf("watcher: %v", err)
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.stopAll()
			return ctx.Err()
		case <-ticker.C:
			m.reload()
			if err := m.sync(ctx); err != nil {
				m.logf("watcher: %v", err)
			}
		}
	}
}

func (m *Manager) reload() {
	targets, err := m.store.ListWalletTargets()
	if err != nil {
		m.logf("watcher: baca wallet: %v", err)
		return
	}
	tokens, err := m.store.ListTokens()
	if err != nil {
		m.logf("watcher: baca token: %v", err)
		return
	}
	configs, err := m.store.ListRPCConfigs()
	if err != nil {
		m.logf("watcher: baca rpc_configs: %v", err)
		return
	}

	byKey := make(map[string]store.RPCConfig, len(configs))
	for _, config := range configs {
		key := network.Canonical(config.Network)
		if key == "" {
			continue
		}
		byKey[key] = config
	}

	m.mu.Lock()
	m.targets = targets
	m.tokens = tokens
	m.rpcConfigs = byKey
	m.mu.Unlock()
}

func (m *Manager) sync(ctx context.Context) error {
	needed := make(map[string]struct{})
	for _, target := range m.snapshotTargets() {
		if key := network.Canonical(target.Network); key != "" {
			needed[key] = struct{}{}
		}
	}

	// Matikan watcher yang sudah tidak dibutuhkan atau konfigurasinya berubah
	// (endpoint diedit lewat Setting, wallet/token ditambah/dihapus).
	for key, cancel := range m.active {
		if _, ok := needed[key]; !ok {
			cancel()
			delete(m.active, key)
			delete(m.sigs, key)
			delete(m.warned, key)
			continue
		}

		m.mu.RLock()
		config := m.rpcConfigs[key]
		m.mu.RUnlock()

		if config.WSSURL == "" || m.signature(key, config) != m.sigs[key] {
			cancel()
			delete(m.active, key)
			delete(m.sigs, key)
		}
	}

	for key := range needed {
		if _, ok := m.active[key]; ok {
			continue
		}

		m.mu.RLock()
		config := m.rpcConfigs[key]
		m.mu.RUnlock()

		if config.WSSURL == "" {
			if !m.warned[key] {
				m.logf("watcher: %s dilewati, wss_url belum diisi di rpc_configs/env", key)
				m.warned[key] = true
			}
			continue
		}

		child, cancel := context.WithCancel(ctx)
		m.active[key] = cancel
		m.sigs[key] = m.signature(key, config)
		delete(m.warned, key)
		m.start(child, key, config)
	}

	return nil
}

// signature merangkum konfigurasi yang membuat sebuah watcher perlu dimulai
// ulang: endpoint dan daftar wallet/token.
func (m *Manager) signature(key string, config store.RPCConfig) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var b strings.Builder
	b.WriteString(config.WSSURL)
	b.WriteByte('|')
	b.WriteString(config.RPCURL)

	for _, target := range m.targets {
		if network.Canonical(target.Network) != key {
			continue
		}
		b.WriteByte('|')
		b.WriteString(target.ID)
		b.WriteByte(':')
		b.WriteString(target.Address)
	}

	for _, token := range m.tokens {
		if network.Canonical(token.Network) != key {
			continue
		}
		b.WriteByte('|')
		b.WriteString(token.Address)
		b.WriteByte(':')
		b.WriteString(token.Symbol)
	}

	return b.String()
}

func (m *Manager) start(ctx context.Context, key string, config store.RPCConfig) {
	def, ok := m.defs[key]
	if !ok {
		m.logf("watcher: definisi jaringan %s tidak dikenal", key)
		return
	}

	if def.Chain == network.ChainSolana {
		w := &solana.Watcher{
			NetworkName:    def.Name,
			WSSURL:         config.WSSURL,
			RPCURL:         config.RPCURL,
			NativeSymbol:   def.NativeSymbol,
			NativeDecimals: def.NativeDecimals,
			RefreshEvery:   m.cfg.Refresh,
			Targets:        func() []solana.Target { return m.solanaTargets(key) },
			Tokens:         func() []solana.Token { return m.solanaTokens(key) },
			OnBalances:     m.persistSolana,
			Logf:           m.logf,
		}
		go func() {
			if err := w.Run(ctx); err != nil && ctx.Err() == nil {
				m.logf("watcher %s: %v", def.Name, err)
			}
		}()
		return
	}

	w := &evm.Watcher{
		NetworkName:    def.Name,
		WSSURL:         config.WSSURL,
		NativeSymbol:   def.NativeSymbol,
		NativeDecimals: def.NativeDecimals,
		Debounce:       m.cfg.Debounce,
		RefreshEvery:   m.cfg.Refresh,
		Targets:        func() []evm.Target { return m.evmTargets(key) },
		Tokens:         func() []evm.Token { return m.evmTokens(key) },
		OnBalances:     m.persistEVM,
		Logf:           m.logf,
	}
	go func() {
		if err := w.Run(ctx); err != nil && ctx.Err() == nil {
			m.logf("watcher %s: %v", def.Name, err)
		}
	}()
}

func (m *Manager) stopAll() {
	for key, cancel := range m.active {
		cancel()
		delete(m.active, key)
		delete(m.sigs, key)
		delete(m.warned, key)
	}
}

func (m *Manager) persistEVM(ctx context.Context, balances []evm.Balance) error {
	rows := make([]store.BalanceRow, 0, len(balances))
	for _, balance := range balances {
		rows = append(rows, store.BalanceRow{
			WalletID:     balance.WalletID,
			Network:      balance.Network,
			TokenAddress: balance.TokenAddress,
			TokenSymbol:  balance.TokenSymbol,
			Balance:      balance.Amount,
		})
	}
	return m.persist(ctx, rows)
}

func (m *Manager) persistSolana(ctx context.Context, balances []solana.Balance) error {
	rows := make([]store.BalanceRow, 0, len(balances))
	for _, balance := range balances {
		rows = append(rows, store.BalanceRow{
			WalletID:     balance.WalletID,
			Network:      balance.Network,
			TokenAddress: balance.TokenAddress,
			TokenSymbol:  balance.TokenSymbol,
			Balance:      balance.Amount,
		})
	}
	return m.persist(ctx, rows)
}

func (m *Manager) persist(ctx context.Context, rows []store.BalanceRow) error {
	if len(rows) == 0 {
		return nil
	}
	// Tulis tetap selesai walau koneksi WS sedang reconnect.
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	return m.store.UpsertBalances(writeCtx, rows)
}

func (m *Manager) snapshotTargets() []store.WalletTarget {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]store.WalletTarget, len(m.targets))
	copy(out, m.targets)
	return out
}

func (m *Manager) evmTargets(key string) []evm.Target {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []evm.Target
	for _, target := range m.targets {
		if network.Canonical(target.Network) != key {
			continue
		}
		out = append(out, evm.Target{
			WalletID: target.ID,
			Address:  target.Address,
			Network:  target.Network,
		})
	}
	return out
}

func (m *Manager) solanaTargets(key string) []solana.Target {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []solana.Target
	for _, target := range m.targets {
		if network.Canonical(target.Network) != key {
			continue
		}
		out = append(out, solana.Target{
			WalletID: target.ID,
			Address:  target.Address,
			Network:  target.Network,
		})
	}
	return out
}

func (m *Manager) evmTokens(key string) []evm.Token {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []evm.Token
	for _, token := range m.tokens {
		if network.Canonical(token.Network) != key {
			continue
		}
		out = append(out, evm.Token{
			Address:  token.Address,
			Symbol:   token.Symbol,
			Decimals: uint8(token.Decimals),
		})
	}
	return out
}

func (m *Manager) solanaTokens(key string) []solana.Token {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []solana.Token
	for _, token := range m.tokens {
		if network.Canonical(token.Network) != key {
			continue
		}
		out = append(out, solana.Token{
			Mint:     token.Address,
			Symbol:   token.Symbol,
			Decimals: uint8(token.Decimals),
		})
	}
	return out
}

func (m *Manager) logf(format string, args ...any) {
	if m.cfg.Logf != nil {
		m.cfg.Logf(format, args...)
	}
}

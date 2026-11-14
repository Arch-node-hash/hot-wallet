package store

import (
	"database/sql"
	"fmt"
	"time"

	"hotwallet/backend/internal/blockchain/network"
)

// RPCSeed adalah nilai awal untuk tabel rpc_configs (biasanya dari env).
type RPCSeed struct {
	Network     string
	RPCURL      string
	WSSURL      string
	ChainID     int64
	ExplorerURL string
}

// RPCConfig adalah konfigurasi endpoint satu jaringan.
type RPCConfig struct {
	Network     string
	RPCURL      string
	WSSURL      string
	ChainID     int64
	ExplorerURL string
	UpdatedAt   string
}

// SeedRPCConfigs mengisi rpc_configs tanpa menimpa nilai yang sudah diubah
// lewat aplikasi. Endpoint kosong di database akan diisi dari seed/env.
func (s *Store) SeedRPCConfigs(seeds []RPCSeed) error {
	if len(seeds) == 0 {
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, seed := range seeds {
		_, err := s.db.Exec(`
			INSERT INTO rpc_configs (network, rpc_url, wss_url, chain_id, explorer_url, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(network) DO UPDATE SET
				rpc_url = CASE
					WHEN rpc_configs.rpc_url = '' THEN excluded.rpc_url
					ELSE rpc_configs.rpc_url
				END,
				wss_url = CASE
					WHEN rpc_configs.wss_url IS NULL OR rpc_configs.wss_url = '' THEN excluded.wss_url
					ELSE rpc_configs.wss_url
				END,
				chain_id = CASE
					WHEN rpc_configs.chain_id IS NULL OR rpc_configs.chain_id = 0 THEN excluded.chain_id
					ELSE rpc_configs.chain_id
				END,
				explorer_url = CASE
					WHEN rpc_configs.explorer_url IS NULL OR rpc_configs.explorer_url = '' THEN excluded.explorer_url
					ELSE rpc_configs.explorer_url
				END
		`, seed.Network, seed.RPCURL, seed.WSSURL, seed.ChainID, seed.ExplorerURL, now)
		if err != nil {
			return fmt.Errorf("store: seed rpc %s: %w", seed.Network, err)
		}
	}
	return nil
}

// ListRPCConfigs mengembalikan seluruh konfigurasi RPC/WSS.
func (s *Store) ListRPCConfigs() ([]RPCConfig, error) {
	rows, err := s.db.Query(`
		SELECT network, rpc_url, wss_url, chain_id, explorer_url, updated_at
		FROM rpc_configs
		ORDER BY network
	`)
	if err != nil {
		return nil, fmt.Errorf("store: list rpc: %w", err)
	}
	defer rows.Close()

	var configs []RPCConfig
	for rows.Next() {
		var (
			config   RPCConfig
			wssURL   sql.NullString
			chainID  sql.NullInt64
			explorer sql.NullString
			updated  sql.NullString
		)
		if err := rows.Scan(&config.Network, &config.RPCURL, &wssURL, &chainID, &explorer, &updated); err != nil {
			return nil, fmt.Errorf("store: scan rpc: %w", err)
		}
		config.WSSURL = wssURL.String
		config.ChainID = chainID.Int64
		config.ExplorerURL = explorer.String
		config.UpdatedAt = updated.String
		configs = append(configs, config)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: rows rpc: %w", err)
	}
	return configs, nil
}

// GetRPCConfig mengambil konfigurasi satu jaringan berdasarkan nama apa pun.
func (s *Store) GetRPCConfig(network string) (*RPCConfig, error) {
	configs, err := s.ListRPCConfigs()
	if err != nil {
		return nil, err
	}
	key := canonicalNetwork(network)
	for i := range configs {
		if canonicalNetwork(configs[i].Network) == key {
			return &configs[i], nil
		}
	}
	return nil, nil
}

// canonicalNetwork adalah pembungkus network.Canonical agar file store lain
// bisa memakai normalisasi nama jaringan yang sama.
func canonicalNetwork(name string) string {
	return network.Canonical(name)
}

// UpdateRPCConfig menyimpan perubahan konfigurasi dari halaman Setting.
func (s *Store) UpdateRPCConfig(config RPCConfig) error {
	_, err := s.db.Exec(`
		UPDATE rpc_configs SET
			rpc_url = ?,
			wss_url = ?,
			chain_id = ?,
			explorer_url = ?,
			updated_at = ?
		WHERE network = ?
	`, config.RPCURL, config.WSSURL, config.ChainID, config.ExplorerURL,
		time.Now().UTC().Format(time.RFC3339), config.Network)
	if err != nil {
		return fmt.Errorf("store: update rpc %s: %w", config.Network, err)
	}
	return nil
}

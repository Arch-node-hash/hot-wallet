package network

import "fmt"

// Registry menyimpan daftar network yang tersedia.
type Registry struct {
	networks map[string]Network
}

func NewRegistry() *Registry {
	return &Registry{networks: make(map[string]Network)}
}

func (r *Registry) Register(n Network) error {
	if n.Name == "" {
		return fmt.Errorf("network: name kosong")
	}
	r.networks[n.Name] = n
	return nil
}

func (r *Registry) Get(name string) (Network, bool) {
	n, ok := r.networks[name]
	return n, ok
}

func (r *Registry) All() []Network {
	result := make([]Network, 0, len(r.networks))
	for _, n := range r.networks {
		result = append(result, n)
	}
	return result
}

// DefaultRegistry mengembalikan registry dengan identitas jaringan default.
// Endpoint RPC/WSS sengaja kosong: nilainya diambil dari konfigurasi env dan
// tabel rpc_configs (lihat internal/infrastructure/store/rpc.go).
func DefaultRegistry() *Registry {
	registry := NewRegistry()

	for _, n := range Defaults() {
		registry.Register(n)
	}

	// Devnet hanya untuk pengembangan; endpoint tetap dari konfigurasi.
	registry.Register(Network{
		Name:           "Solana Devnet",
		Chain:          ChainSolana,
		ChainID:        0,
		ExplorerURL:    "https://solscan.io/?cluster=devnet",
		NativeSymbol:   "SOL",
		NativeDecimals: 9,
	})

	return registry
}

// GetNetworkByChainID mencari network berdasarkan chain ID (untuk EVM).
func (r *Registry) GetNetworkByChainID(chainID int64) (Network, bool) {
	for _, n := range r.networks {
		if n.Chain == ChainEVM && n.ChainID == chainID {
			return n, true
		}
	}
	return Network{}, false
}

// GetNetworksByChain mengembalikan semua network untuk chain tertentu.
func (r *Registry) GetNetworksByChain(chain Chain) []Network {
	result := make([]Network, 0)
	for _, n := range r.networks {
		if n.Chain == chain {
			result = append(result, n)
		}
	}
	return result
}

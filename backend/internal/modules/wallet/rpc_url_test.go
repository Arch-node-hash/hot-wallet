package wallet

import (
	"testing"

	"hotwallet/backend/internal/blockchain/network"
	"hotwallet/backend/internal/infrastructure/store"
)

func TestResolveRPCURL(t *testing.T) {
	evmMeta, ok := network.MetaFor("ethereum")
	if !ok {
		t.Fatal("meta ethereum tidak ditemukan")
	}
	solMeta, ok := network.MetaFor("solana")
	if !ok {
		t.Fatal("meta solana tidak ditemukan")
	}

	cases := []struct {
		name   string
		config *store.RPCConfig
		meta   network.Meta
		want   string
	}{
		{
			name:   "config nil",
			config: nil,
			meta:   evmMeta,
			want:   "",
		},
		{
			name:   "evm pakai rpc_url",
			config: &store.RPCConfig{RPCURL: "https://eth.example"},
			meta:   evmMeta,
			want:   "https://eth.example",
		},
		{
			name:   "solana rpc_url kosong diturunkan dari wss",
			config: &store.RPCConfig{WSSURL: "wss://solana.example"},
			meta:   solMeta,
			want:   "https://solana.example",
		},
		{
			name:   "solana rpc_url diisi tetap dipakai",
			config: &store.RPCConfig{RPCURL: "https://solana.example", WSSURL: "wss://solana.example"},
			meta:   solMeta,
			want:   "https://solana.example",
		},
		{
			name:   "solana tanpa rpc_url dan wss",
			config: &store.RPCConfig{},
			meta:   solMeta,
			want:   "",
		},
		{
			name:   "evm tidak menurunkan dari wss",
			config: &store.RPCConfig{WSSURL: "wss://eth.example"},
			meta:   evmMeta,
			want:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveRPCURL(tc.config, tc.meta); got != tc.want {
				t.Fatalf("resolveRPCURL = %q, mau %q", got, tc.want)
			}
		})
	}
}

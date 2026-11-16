// Package pricing menggabungkan harga token dari tabel token_prices dengan
// fallback stablecoin $1. Dipakai oleh halaman Wallet dan watcher supaya
// perhitungan nilai USD di API dan di database selalu memakai aturan sama.
package pricing

import (
	"math/big"
	"strings"

	"hotwallet/backend/internal/blockchain/network"
	"hotwallet/backend/internal/infrastructure/store"
)

// stablecoins selalu dihargai $1 walau tabel token_prices kosong.
var stablecoins = map[string]bool{
	"USDT": true, "USDC": true, "DAI": true, "BUSD": true,
	"TUSD": true, "FDUSD": true, "USDE": true, "USDD": true,
	"USDG": true,
}

// IsStablecoin melaporkan apakah simbol token adalah stablecoin $1.
func IsStablecoin(symbol string) bool {
	return stablecoins[strings.ToUpper(strings.TrimSpace(symbol))]
}

// Normalize mengubah harga kosong menjadi "0".
func Normalize(price string) string {
	if strings.TrimSpace(price) == "" {
		return "0"
	}
	return price
}

// Index adalah pencarian harga per jaringan/alamat/simbol.
type Index struct {
	byAddress map[string]store.TokenPrice
	bySymbol  map[string]store.TokenPrice
}

// New membangun index dari daftar harga token.
func New(prices []store.TokenPrice) *Index {
	idx := &Index{
		byAddress: make(map[string]store.TokenPrice),
		bySymbol:  make(map[string]store.TokenPrice),
	}
	for _, p := range prices {
		key := network.Canonical(p.Network)
		idx.byAddress[key+"\x00"+strings.ToLower(p.TokenAddress)] = p
		idx.bySymbol[key+"\x00"+strings.ToUpper(p.Symbol)] = p
	}
	return idx
}

// Lookup mencari harga token: cocok alamat dulu, lalu simbol, lalu fallback
// stablecoin $1. Mengembalikan harga dan perubahan 24 jam (persen).
func (idx *Index) Lookup(networkName, tokenAddress, symbol string) (string, float64) {
	key := network.Canonical(networkName)
	if p, ok := idx.byAddress[key+"\x00"+strings.ToLower(tokenAddress)]; ok {
		return Normalize(p.PriceUSD), p.Change24h
	}
	if p, ok := idx.bySymbol[key+"\x00"+strings.ToUpper(symbol)]; ok {
		return Normalize(p.PriceUSD), p.Change24h
	}
	if IsStablecoin(symbol) {
		return "1", 0
	}
	return "0", 0
}

// Price hanya mengembalikan harga tanpa perubahan 24 jam.
func (idx *Index) Price(networkName, tokenAddress, symbol string) string {
	price, _ := idx.Lookup(networkName, tokenAddress, symbol)
	return price
}

// Value menghitung balance × harga memakai desimal (tanpa float).
func Value(balance, price string) string {
	rb, ok := new(big.Rat).SetString(strings.TrimSpace(balance))
	if !ok {
		return "0"
	}
	rp, ok := new(big.Rat).SetString(strings.TrimSpace(price))
	if !ok {
		return "0"
	}
	return new(big.Float).SetRat(new(big.Rat).Mul(rb, rp)).Text('f', 2)
}

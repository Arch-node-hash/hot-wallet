package store

import (
	"math/big"
	"strings"
)

// Helper desimal berbasis math/big supaya nilai uang tidak terkena error
// pembulatan float. Semua nilai disimpan sebagai string dengan 2 desimal.

func mulDecimal(a, b string) string {
	ra, ok := new(big.Rat).SetString(strings.TrimSpace(a))
	if !ok {
		return "0"
	}
	rb, ok := new(big.Rat).SetString(strings.TrimSpace(b))
	if !ok {
		return "0"
	}
	return formatDecimal(new(big.Rat).Mul(ra, rb))
}

func addDecimal(a, b string) string {
	ra, ok := new(big.Rat).SetString(strings.TrimSpace(a))
	if !ok {
		ra = new(big.Rat)
	}
	rb, ok := new(big.Rat).SetString(strings.TrimSpace(b))
	if !ok {
		rb = new(big.Rat)
	}
	return formatDecimal(new(big.Rat).Add(ra, rb))
}

func formatDecimal(r *big.Rat) string {
	return new(big.Float).SetRat(r).Text('f', 2)
}

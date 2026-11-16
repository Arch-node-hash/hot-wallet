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

// IDRValue menghitung nilai rupiah dari nilai USD × kurs, dibulatkan ke
// rupiah bulat (tanpa sen) supaya frontend tidak perlu memformat ulang.
func IDRValue(valueUSD, rate string) string {
	return roundIDR(mulDecimal(valueUSD, rate))
}

// roundIDR membulatkan desimal ke rupiah terdekat (setengah ke atas).
func roundIDR(value string) string {
	r, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok {
		return "0"
	}
	r.Add(r, big.NewRat(1, 2))
	return new(big.Int).Quo(r.Num(), r.Denom()).String()
}

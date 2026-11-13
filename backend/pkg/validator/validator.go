package validator

import (
	"fmt"
	"regexp"
	"strings"
)

var reEthAddress = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
var reEmail = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// WalletAddress memvalidasi format Ethereum address (0x + 40 hex chars).
func WalletAddress(addr string) error {
	if addr == "" {
		return nil // kosong diperbolehkan
	}
	if !reEthAddress.MatchString(addr) {
		return fmt.Errorf("wallet address tidak valid: harus format 0x diikuti 40 karakter hex")
	}
	return nil
}

// Email memvalidasi format email.
func Email(email string) error {
	if email == "" {
		return fmt.Errorf("email tidak boleh kosong")
	}
	if !reEmail.MatchString(email) {
		return fmt.Errorf("format email tidak valid")
	}
	return nil
}

// Required memvalidasi bahwa string tidak kosong.
func Required(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s tidak boleh kosong", field)
	}
	return nil
}

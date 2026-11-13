package wallet

import (
	"fmt"

	"github.com/tyler-smith/go-bip39"
)

// NewMnemonic membuat seed phrase BIP39 baru dengan jumlah kata tertentu
// (12, 15, 18, 21, atau 24).
func NewMnemonic(words int) (string, error) {
	switch words {
	case 12, 15, 18, 21, 24:
	default:
		return "", fmt.Errorf("wallet: jumlah kata harus 12, 15, 18, 21, atau 24")
	}

	entropy, err := bip39.NewEntropy(words * 32 / 3)
	if err != nil {
		return "", fmt.Errorf("wallet: generate entropy: %w", err)
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("wallet: generate mnemonic: %w", err)
	}
	return mnemonic, nil
}

// ValidateMnemonic memeriksa checksum dan daftar kata BIP39.
func ValidateMnemonic(mnemonic string) bool {
	return bip39.IsMnemonicValid(mnemonic)
}

// SeedFromMnemonic menurunkan seed 64 byte dari mnemonic + passphrase.
func SeedFromMnemonic(mnemonic, passphrase string) ([]byte, error) {
	if !ValidateMnemonic(mnemonic) {
		return nil, fmt.Errorf("wallet: seed phrase tidak valid")
	}

	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, passphrase)
	if err != nil {
		return nil, fmt.Errorf("wallet: seed: %w", err)
	}
	return seed, nil
}

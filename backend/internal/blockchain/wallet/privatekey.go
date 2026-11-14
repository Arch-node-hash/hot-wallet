package wallet

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
)

// PrivateKeyType menentukan jenis private key
type PrivateKeyType int

const (
	PrivateKeyTypeEVM PrivateKeyType = iota
	PrivateKeyTypeSolana
)

// PrivateKeyInfo menyimpan informasi private key yang sudah di-parse
type PrivateKeyInfo struct {
	Type       PrivateKeyType
	RawBytes   []byte
	Address    string
	PublicKey  []byte
	PrivateKey string // dalam format asli (hex/base58)
}

// ParseEVMPrivateKey mem-parse private key Ethereum (64 hex characters dengan 0x prefix)
func ParseEVMPrivateKey(privKeyStr string) (*PrivateKeyInfo, error) {
	// Hapus 0x prefix jika ada
	privKeyStr = strings.TrimPrefix(privKeyStr, "0x")

	// Decode hex
	privKeyBytes, err := hex.DecodeString(privKeyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}

	// Validasi panjang (32 bytes = 64 hex characters)
	if len(privKeyBytes) != 32 {
		return nil, fmt.Errorf("invalid length: got %d bytes, want 32", len(privKeyBytes))
	}

	// Konversi ke private key ECDSA
	privateKey, err := crypto.ToECDSA(privKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	// Dapatkan public key
	publicKey := privateKey.PublicKey

	// Generate address Ethereum
	address := crypto.PubkeyToAddress(publicKey).Hex()

	return &PrivateKeyInfo{
		Type:       PrivateKeyTypeEVM,
		RawBytes:   privKeyBytes,
		Address:    address,
		PublicKey:  crypto.FromECDSAPub(&publicKey),
		PrivateKey: "0x" + hex.EncodeToString(privKeyBytes),
	}, nil
}

// ParseSolanaPrivateKey mem-parse private key Solana. Format yang diterima:
// base58 32 byte (seed), base58 64 byte (keypair Phantom/solana-keygen),
// hex 64/128 karakter, atau JSON array byte (format `solana-keygen`).
func ParseSolanaPrivateKey(privKeyStr string) (*PrivateKeyInfo, error) {
	str := strings.TrimSpace(privKeyStr)

	var raw []byte

	switch {
	case strings.HasPrefix(str, "["):
		var nums []int
		if err := json.Unmarshal([]byte(str), &nums); err != nil {
			return nil, fmt.Errorf("invalid JSON array: %w", err)
		}
		raw = make([]byte, len(nums))
		for i, n := range nums {
			if n < 0 || n > 255 {
				return nil, fmt.Errorf("byte ke-%d di luar rentang 0-255", i)
			}
			raw[i] = byte(n)
		}
	case strings.HasPrefix(str, "0x"):
		decoded, err := hex.DecodeString(str[2:])
		if err != nil {
			return nil, fmt.Errorf("invalid hex: %w", err)
		}
		raw = decoded
	default:
		// Hex 64/128 karakter (32/64 byte) diperlakukan sebagai hex.
		if decoded, err := hex.DecodeString(str); err == nil && (len(decoded) == 32 || len(decoded) == 64) {
			raw = decoded
			break
		}

		decoded, err := base58.Decode(str)
		if err != nil {
			return nil, fmt.Errorf("invalid base58: %w", err)
		}
		raw = decoded
	}

	// Seed ed25519 selalu 32 byte. Keypair 64 byte = seed (32) + public key (32).
	var seed []byte
	switch len(raw) {
	case 32:
		seed = raw
	case 64:
		seed = raw[:32]
	default:
		return nil, fmt.Errorf(
			"invalid length: got %d bytes, want 32 (seed) atau 64 (keypair); "+
				"pakai base58, hex 64/128 karakter, atau JSON array",
			len(raw),
		)
	}

	// Buat private key ed25519
	privateKey := ed25519.NewKeyFromSeed(seed)

	// Dapatkan public key
	publicKey := privateKey.Public().(ed25519.PublicKey)

	// Generate address Solana (base58 encoded public key)
	address := base58.Encode(publicKey)

	return &PrivateKeyInfo{
		Type:       PrivateKeyTypeSolana,
		RawBytes:   seed,
		Address:    address,
		PublicKey:  publicKey,
		PrivateKey: privKeyStr,
	}, nil
}

// SignEVMTransaction menandatangani transaksi EVM dengan private key
func SignEVMTransaction(privKeyInfo *PrivateKeyInfo, txHash []byte) ([]byte, error) {
	if privKeyInfo.Type != PrivateKeyTypeEVM {
		return nil, fmt.Errorf("invalid key type for EVM signing")
	}

	privateKey, err := crypto.ToECDSA(privKeyInfo.RawBytes)
	if err != nil {
		return nil, err
	}

	// Sign hash dengan private key
	sig, err := crypto.Sign(txHash, privateKey)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

// SignSolanaTransaction menandatangani transaksi Solana dengan private key
func SignSolanaTransaction(privKeyInfo *PrivateKeyInfo, message []byte) ([]byte, error) {
	if privKeyInfo.Type != PrivateKeyTypeSolana {
		return nil, fmt.Errorf("invalid key type for Solana signing")
	}

	privateKey := ed25519.NewKeyFromSeed(privKeyInfo.RawBytes)

	// Sign message dengan private key ed25519
	sig := ed25519.Sign(privateKey, message)

	return sig, nil
}

// ValidatePrivateKey memvalidasi format private key
func ValidatePrivateKey(privKeyStr string, keyType PrivateKeyType) bool {
	switch keyType {
	case PrivateKeyTypeEVM:
		_, err := ParseEVMPrivateKey(privKeyStr)
		return err == nil
	case PrivateKeyTypeSolana:
		_, err := ParseSolanaPrivateKey(privKeyStr)
		return err == nil
	default:
		return false
	}
}

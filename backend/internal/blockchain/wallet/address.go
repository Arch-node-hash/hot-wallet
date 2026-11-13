package wallet

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"golang.org/x/crypto/sha3"
)

const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// EVMPublicKey mengubah private key secp256k1 menjadi public key uncompressed
// 65 byte (prefix 0x04).
func EVMPublicKey(privateKey []byte) ([]byte, error) {
	if len(privateKey) != 32 {
		return nil, fmt.Errorf("wallet: private key EVM harus 32 byte")
	}
	return secp256k1.PrivKeyFromBytes(privateKey).PubKey().SerializeUncompressed(), nil
}

// EVMAddressFromPrivateKey menghasilkan alamat Ethereum dengan checksum
// EIP-55.
func EVMAddressFromPrivateKey(privateKey []byte) (string, error) {
	publicKey, err := EVMPublicKey(privateKey)
	if err != nil {
		return "", err
	}
	return EVMAddress(publicKey), nil
}

// EVMAddress menghitung alamat Ethereum dari public key uncompressed.
func EVMAddress(publicKey []byte) string {
	if len(publicKey) == 65 {
		publicKey = publicKey[1:]
	}

	hash := sha3.NewLegacyKeccak256()
	hash.Write(publicKey)
	sum := hash.Sum(nil)

	return evmChecksum(sum[12:])
}

// SolanaAddress mengubah public key ed25519 menjadi alamat base58.
func SolanaAddress(publicKey []byte) string {
	return base58Encode(publicKey)
}

func evmChecksum(address []byte) string {
	lower := hex.EncodeToString(address)

	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(lower))
	sum := hash.Sum(nil)

	out := []byte(lower)
	for i := range out {
		nibble := sum[i/2]
		if i%2 == 0 {
			nibble >>= 4
		} else {
			nibble &= 0x0f
		}
		if nibble >= 8 && out[i] >= 'a' && out[i] <= 'f' {
			out[i] -= 'a' - 'A'
		}
	}
	return "0x" + string(out)
}

func base58Encode(input []byte) string {
	if len(input) == 0 {
		return ""
	}

	value := new(big.Int).SetBytes(input)
	base := big.NewInt(58)
	zero := big.NewInt(0)
	mod := new(big.Int)

	encoded := make([]byte, 0, len(input)*2)
	for value.Cmp(zero) > 0 {
		value.DivMod(value, base, mod)
		encoded = append(encoded, base58Alphabet[mod.Int64()])
	}

	for _, b := range input {
		if b != 0 {
			break
		}
		encoded = append(encoded, base58Alphabet[0])
	}

	for left, right := 0, len(encoded)-1; left < right; left, right = left+1, right-1 {
		encoded[left], encoded[right] = encoded[right], encoded[left]
	}
	return string(encoded)
}

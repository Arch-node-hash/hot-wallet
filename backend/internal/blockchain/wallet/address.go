package wallet

import "encoding/hex"

// Address adalah placeholder pembentukan alamat dari public key.
// Ganti dengan encoding chain yang dipakai (EVM/Solana).
func Address(publicKey []byte) string {
	return hex.EncodeToString(publicKey)
}

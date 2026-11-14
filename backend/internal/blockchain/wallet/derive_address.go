package wallet

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// addressSeedPrefix memisahkan seed derivasi alamat dari pemakaian lain, agar
// seed yang dihasilkan tidak pernah bertabrakan dengan seed BIP39 asli.
const addressSeedPrefix = "HotWallet/address/v1/"

// pathEVM0 adalah path yang ditampilkan untuk alamat index 0 EVM.
const pathEVM0 = "m/44'/60'/0'/0/0"

// pathSolana0 adalah path yang ditampilkan untuk alamat index 0 Solana.
const pathSolana0 = "m/44'/501'/0'/0'"

// AddressSeedFromPrivateKey menghasilkan seed deterministik 32 byte dari
// private key mentah. Seed ini dipakai untuk menurunkan alamat pada index > 0
// sehingga "Perbaharui Address" menghasilkan alamat baru tanpa perlu menyimpan
// private key tambahan: cukup menaikkan index HD.
func AddressSeedFromPrivateKey(info *PrivateKeyInfo) []byte {
	buf := make([]byte, 0, len(addressSeedPrefix)+len(info.RawBytes))
	buf = append(buf, addressSeedPrefix...)
	buf = append(buf, info.RawBytes...)
	sum := sha256.Sum256(buf)
	return sum[:]
}

// DeriveAddressFromPrivateKey menurunkan alamat pada index HD tertentu.
//
// Index 0 selalu alamat langsung dari private key (backward-compatible dengan
// wallet yang sudah ada dari env). Index > 0 diturunkan deterministik dari seed
// turunan key tersebut: BIP44 (secp256k1) untuk EVM dan SLIP-0010 (ed25519)
// untuk Solana.
//
// Mengembalikan alamat dan path derivasi yang ditampilkan ke user.
func DeriveAddressFromPrivateKey(info *PrivateKeyInfo, index uint32) (address, path string, err error) {
	if info == nil {
		return "", "", fmt.Errorf("wallet: private key kosong")
	}

	switch info.Type {
	case PrivateKeyTypeEVM:
		if index == 0 {
			return info.Address, pathEVM0, nil
		}
		master, err := NewMasterKey(AddressSeedFromPrivateKey(info))
		if err != nil {
			return "", "", err
		}
		derived, err := Derive(master, index)
		if err != nil {
			return "", "", err
		}
		return derived.Address, derived.Path, nil

	case PrivateKeyTypeSolana:
		if index == 0 {
			return info.Address, pathSolana0, nil
		}
		seed := AddressSeedFromPrivateKey(info)
		path := []uint32{
			hardened(purposeEVM),
			hardened(coinTypeSol),
			hardened(index),
			hardened(accountIndex0),
		}
		key, _ := deriveEd25519(seed, path)
		publicKey := ed25519.NewKeyFromSeed(key).Public().(ed25519.PublicKey)
		return SolanaAddress(publicKey), formatPath(path), nil

	default:
		return "", "", fmt.Errorf("wallet: tipe private key tidak dikenal")
	}
}

// PrivateKeyAtIndex mengembalikan PrivateKeyInfo yang siap dipakai menandatangani
// transaksi pada index tertentu. Index 0 mengembalikan key asli; index > 0
// menurunkan key dari seed deterministik.
func PrivateKeyAtIndex(info *PrivateKeyInfo, index uint32) (*PrivateKeyInfo, error) {
	if info == nil {
		return nil, fmt.Errorf("wallet: private key kosong")
	}
	if index == 0 {
		return info, nil
	}

	seed := AddressSeedFromPrivateKey(info)

	switch info.Type {
	case PrivateKeyTypeEVM:
		master, err := NewMasterKey(seed)
		if err != nil {
			return nil, err
		}
		derived, err := Derive(master, index)
		if err != nil {
			return nil, err
		}
		return &PrivateKeyInfo{
			Type:       PrivateKeyTypeEVM,
			RawBytes:   derived.Key,
			Address:    derived.Address,
			PublicKey:  derived.PublicKey,
			PrivateKey: "0x" + hex.EncodeToString(derived.Key),
		}, nil

	case PrivateKeyTypeSolana:
		path := []uint32{
			hardened(purposeEVM),
			hardened(coinTypeSol),
			hardened(index),
			hardened(accountIndex0),
		}
		key, _ := deriveEd25519(seed, path)
		privateKey := ed25519.NewKeyFromSeed(key)
		publicKey := privateKey.Public().(ed25519.PublicKey)
		return &PrivateKeyInfo{
			Type:       PrivateKeyTypeSolana,
			RawBytes:   key,
			Address:    SolanaAddress(publicKey),
			PublicKey:  append([]byte(nil), publicKey...),
			PrivateKey: info.PrivateKey,
		}, nil

	default:
		return nil, fmt.Errorf("wallet: tipe private key tidak dikenal")
	}
}

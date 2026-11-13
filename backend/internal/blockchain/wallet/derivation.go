package wallet

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"fmt"

	"github.com/tyler-smith/go-bip32"
)

const (
	purposeEVM    = 44
	coinTypeEVM   = 60
	coinTypeSol   = 501
	accountIndex0 = 0
	changeEVM     = 0
)

// DerivedKey adalah hasil turunan seed phrase untuk satu akun.
type DerivedKey struct {
	Index     uint32
	Path      string
	Key       []byte
	ChainCode []byte
	PublicKey []byte
	Address   string
}

// NewMasterKeyFromMnemonic membuat master key BIP39 dari seed phrase.
func NewMasterKeyFromMnemonic(mnemonic, passphrase string) (*MasterKey, error) {
	seed, err := SeedFromMnemonic(mnemonic, passphrase)
	if err != nil {
		return nil, err
	}
	return NewMasterKey(seed)
}

// Derive menurunkan akun EVM (BIP44) dari master key.
func Derive(master *MasterKey, index uint32) (DerivedKey, error) {
	if master == nil || len(master.Key) == 0 {
		return DerivedKey{}, fmt.Errorf("wallet: master key kosong")
	}

	path := []uint32{
		hardened(purposeEVM),
		hardened(coinTypeEVM),
		hardened(accountIndex0),
		changeEVM,
		index,
	}

	child, err := deriveBIP32(master.Key, path)
	if err != nil {
		return DerivedKey{}, err
	}

	address, err := EVMAddressFromPrivateKey(child.Key)
	if err != nil {
		return DerivedKey{}, err
	}
	publicKey, err := EVMPublicKey(child.Key)
	if err != nil {
		return DerivedKey{}, err
	}

	return DerivedKey{
		Index:     index,
		Path:      formatPath(path),
		Key:       append([]byte(nil), child.Key...),
		ChainCode: append([]byte(nil), child.ChainCode...),
		PublicKey: publicKey,
		Address:   address,
	}, nil
}

// DeriveEVM menurunkan akun EVM dari seed phrase pada path
// m/44'/60'/0'/0/{index}.
func DeriveEVM(mnemonic, passphrase string, index uint32) (DerivedKey, error) {
	master, err := NewMasterKeyFromMnemonic(mnemonic, passphrase)
	if err != nil {
		return DerivedKey{}, err
	}
	return Derive(master, index)
}

// DeriveSolana menurunkan akun Solana (SLIP-0010 ed25519) dari seed phrase
// pada path m/44'/501'/{index}'/0'.
func DeriveSolana(mnemonic, passphrase string, index uint32) (DerivedKey, error) {
	seed, err := SeedFromMnemonic(mnemonic, passphrase)
	if err != nil {
		return DerivedKey{}, err
	}

	path := []uint32{
		hardened(purposeEVM),
		hardened(coinTypeSol),
		hardened(index),
		hardened(accountIndex0),
	}

	key, chainCode := deriveEd25519(seed, path)
	privateKey := ed25519.NewKeyFromSeed(key)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return DerivedKey{
		Index:     index,
		Path:      formatPath(path),
		Key:       append([]byte(nil), key...),
		ChainCode: chainCode,
		PublicKey: append([]byte(nil), publicKey...),
		Address:   SolanaAddress(publicKey),
	}, nil
}

func deriveBIP32(seed []byte, path []uint32) (*bip32.Key, error) {
	key, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, fmt.Errorf("wallet: master BIP32: %w", err)
	}

	for _, index := range path {
		key, err = key.NewChildKey(index)
		if err != nil {
			return nil, fmt.Errorf("wallet: turunan BIP32: %w", err)
		}
	}
	return key, nil
}

// deriveEd25519 mengikuti SLIP-0010 untuk kurva ed25519 (semua langkah
// hardened).
func deriveEd25519(seed []byte, path []uint32) (key, chainCode []byte) {
	mac := hmac.New(sha512.New, []byte("ed25519 seed"))
	mac.Write(seed)
	sum := mac.Sum(nil)

	key = append([]byte(nil), sum[:32]...)
	chainCode = append([]byte(nil), sum[32:]...)

	for _, index := range path {
		var buf [37]byte
		copy(buf[1:], key)
		binary.BigEndian.PutUint32(buf[33:], hardened(index))

		mac = hmac.New(sha512.New, chainCode)
		mac.Write(buf[:])
		sum = mac.Sum(nil)

		key = append([]byte(nil), sum[:32]...)
		chainCode = append([]byte(nil), sum[32:]...)
	}
	return key, chainCode
}

func hardened(index uint32) uint32 {
	return index | bip32.FirstHardenedChild
}

func formatPath(path []uint32) string {
	out := "m"
	for _, index := range path {
		if index >= bip32.FirstHardenedChild {
			out += fmt.Sprintf("/%d'", index-bip32.FirstHardenedChild)
			continue
		}
		out += fmt.Sprintf("/%d", index)
	}
	return out
}

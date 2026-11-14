package wallet

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

func TestNewMnemonic(t *testing.T) {
	mnemonic, err := NewMnemonic(12)
	if err != nil {
		t.Fatalf("NewMnemonic: %v", err)
	}
	if got := len(strings.Fields(mnemonic)); got != 12 {
		t.Fatalf("jumlah kata = %d, mau 12", got)
	}
	if !ValidateMnemonic(mnemonic) {
		t.Fatal("mnemonic hasil generate tidak valid")
	}
}

func TestDeriveEVMKnownVector(t *testing.T) {
	// Hardhat account #0 untuk mnemonic standar.
	mnemonic := "test test test test test test test test test test test junk"
	const want = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

	got, err := DeriveEVM(mnemonic, "", 0)
	if err != nil {
		t.Fatalf("DeriveEVM: %v", err)
	}
	if got.Address != want {
		t.Fatalf("alamat EVM = %s, mau %s", got.Address, want)
	}
	if got.Path != "m/44'/60'/0'/0/0" {
		t.Fatalf("path = %s", got.Path)
	}
}

func TestDeriveEd25519Slip10Vector(t *testing.T) {
	// SLIP-0010 test vector 1: seed 000102...0f, path m/0'.
	seed, err := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	if err != nil {
		t.Fatal(err)
	}

	key, chain := deriveEd25519(seed, []uint32{0})

	const wantKey = "68e0fe46dfb67e368c75379acec591dad19df3cde26e63b93a8e704f1dade7a3"
	const wantChain = "8b59aa11380b624e81507a27fedda59fea6d0b779a778918a2fd3590e16e9c69"

	if got := hex.EncodeToString(key); got != wantKey {
		t.Fatalf("key = %s, mau %s", got, wantKey)
	}
	if got := hex.EncodeToString(chain); got != wantChain {
		t.Fatalf("chain code = %s, mau %s", got, wantChain)
	}
}

func TestDeriveSolana(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"

	account, err := DeriveSolana(mnemonic, "", 0)
	if err != nil {
		t.Fatalf("DeriveSolana: %v", err)
	}
	if len(account.PublicKey) != 32 {
		t.Fatalf("panjang public key = %d, mau 32", len(account.PublicKey))
	}
	if account.Address == "" {
		t.Fatal("alamat Solana kosong")
	}
	if account.Path != "m/44'/501'/0'/0'" {
		t.Fatalf("path = %s", account.Path)
	}
	if strings.Contains(account.Address, "0") || strings.Contains(account.Address, "O") {
		t.Fatalf("alamat base58 tidak boleh mengandung 0/O: %s", account.Address)
	}
}

func TestBase58Encode(t *testing.T) {
	if got := base58Encode([]byte{0, 0, 1}); got != "112" {
		t.Fatalf("base58 = %s, mau 112", got)
	}
}

func TestParseSolanaPrivateKeyFormats(t *testing.T) {
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = byte(i)
	}

	fromSeed, err := ParseSolanaPrivateKey(base58Encode(seed))
	if err != nil {
		t.Fatalf("base58 seed: %v", err)
	}

	// Keypair 64 byte = seed + public key (format export Phantom/solana-keygen).
	keypair := ed25519.NewKeyFromSeed(seed)
	fromKeypair, err := ParseSolanaPrivateKey(base58Encode(keypair))
	if err != nil {
		t.Fatalf("base58 keypair: %v", err)
	}
	if fromKeypair.Address != fromSeed.Address {
		t.Fatalf("alamat keypair %s != seed %s", fromKeypair.Address, fromSeed.Address)
	}

	fromHex, err := ParseSolanaPrivateKey("0x" + hex.EncodeToString(seed))
	if err != nil {
		t.Fatalf("hex seed: %v", err)
	}
	if fromHex.Address != fromSeed.Address {
		t.Fatalf("alamat hex %s != seed %s", fromHex.Address, fromSeed.Address)
	}

	nums := make([]string, len(keypair))
	for i, b := range keypair {
		nums[i] = fmt.Sprint(b)
	}
	fromJSON, err := ParseSolanaPrivateKey("[" + strings.Join(nums, ",") + "]")
	if err != nil {
		t.Fatalf("json keypair: %v", err)
	}
	if fromJSON.Address != fromSeed.Address {
		t.Fatalf("alamat json %s != seed %s", fromJSON.Address, fromSeed.Address)
	}

	if _, err := ParseSolanaPrivateKey("5JRaypasxMx1L97ZUX7YuKC5NpLJ7bRjv7Z5wP5JvF1qG"); err == nil {
		t.Fatal("key 33 byte seharusnya ditolak")
	}
}

package wallet

import "testing"

const testEVMKey = "0x0123456789012345678901234567890123456789012345678901234567890123"

func TestDeriveAddressFromPrivateKeyIndex0IsDirectKey(t *testing.T) {
	info, err := ParseEVMPrivateKey(testEVMKey)
	if err != nil {
		t.Fatalf("ParseEVMPrivateKey: %v", err)
	}

	address, path, err := DeriveAddressFromPrivateKey(info, 0)
	if err != nil {
		t.Fatalf("DeriveAddressFromPrivateKey: %v", err)
	}
	if address != info.Address {
		t.Fatalf("index 0 = %s, mau %s", address, info.Address)
	}
	if path != pathEVM0 {
		t.Fatalf("path = %s, mau %s", path, pathEVM0)
	}
}

func TestDeriveAddressFromPrivateKeyDeterministic(t *testing.T) {
	info, err := ParseEVMPrivateKey(testEVMKey)
	if err != nil {
		t.Fatal(err)
	}

	first, path1, err := DeriveAddressFromPrivateKey(info, 1)
	if err != nil {
		t.Fatal(err)
	}
	second, path2, err := DeriveAddressFromPrivateKey(info, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || path1 != path2 {
		t.Fatalf("derivasi index 1 tidak deterministik: %s/%s vs %s/%s", first, path1, second, path2)
	}

	third, _, err := DeriveAddressFromPrivateKey(info, 2)
	if err != nil {
		t.Fatal(err)
	}
	if third == first || third == info.Address {
		t.Fatalf("index berbeda harus menghasilkan alamat berbeda (%s)", third)
	}
	if path1 == path2 && path1 == pathEVM0 {
		t.Fatal("path index > 0 tidak boleh sama dengan index 0")
	}
}

func TestDeriveSolanaAddressFromPrivateKey(t *testing.T) {
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	info, err := ParseSolanaPrivateKey(base58Encode(seed))
	if err != nil {
		t.Fatal(err)
	}

	address0, path0, err := DeriveAddressFromPrivateKey(info, 0)
	if err != nil {
		t.Fatal(err)
	}
	if address0 != info.Address || path0 != pathSolana0 {
		t.Fatalf("index 0 Solana = %s/%s, mau %s/%s", address0, path0, info.Address, pathSolana0)
	}

	address1, _, err := DeriveAddressFromPrivateKey(info, 1)
	if err != nil {
		t.Fatal(err)
	}
	if address1 == address0 {
		t.Fatal("index 1 Solana harus menghasilkan alamat berbeda")
	}
	if len(address1) < 32 {
		t.Fatalf("alamat Solana terlalu pendek: %s", address1)
	}
}

func TestPrivateKeyAtIndexMatchesDerivedAddress(t *testing.T) {
	info, err := ParseEVMPrivateKey(testEVMKey)
	if err != nil {
		t.Fatal(err)
	}

	for _, index := range []uint32{0, 1, 5} {
		address, _, err := DeriveAddressFromPrivateKey(info, index)
		if err != nil {
			t.Fatal(err)
		}
		key, err := PrivateKeyAtIndex(info, index)
		if err != nil {
			t.Fatal(err)
		}
		if key.Address != address {
			t.Fatalf("index %d: PrivateKeyAtIndex %s != derived %s", index, key.Address, address)
		}
	}
}

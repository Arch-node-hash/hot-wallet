package solana

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"testing"
)

// shortVec membaca compact-u16 (shortvec) dari awal buf dan mengembalikan nilai
// serta sisa byte setelahnya.
func shortVec(t *testing.T, buf []byte) (int, []byte) {
	t.Helper()
	value := 0
	shift := 0
	for i, b := range buf {
		value |= int(b&0x7f) << shift
		if b&0x80 == 0 {
			return value, buf[i+1:]
		}
		shift += 7
		if shift > 14 {
			t.Fatalf("shortvec terlalu panjang")
		}
	}
	t.Fatalf("shortvec terpotong")
	return 0, nil
}

// decodedTx adalah hasil pembacaan transaksi Solana base64.
type decodedTx struct {
	signature []byte
	header    []byte
	keys      [][]byte
	blockhash []byte
	ixs       []instruction
	message   []byte
}

// decodeTx membaca transaksi base64 legacy Solana: shortvec jumlah signature,
// signature, lalu pesan (header, akun, blockhash, instruksi).
func decodeTx(t *testing.T, txBase64 string) decodedTx {
	t.Helper()

	raw, err := base64.StdEncoding.DecodeString(txBase64)
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}

	nSigs, rest := shortVec(t, raw)
	if nSigs != 1 {
		t.Fatalf("jumlah signature = %d, mau 1", nSigs)
	}
	if len(rest) < 64 {
		t.Fatalf("signature terpotong: %d byte", len(rest))
	}
	signature := rest[:64]
	message := rest[64:]

	if len(message) < 3 {
		t.Fatalf("pesan terpotong")
	}
	header := message[:3]
	body := message[3:]

	nKeys, body := shortVec(t, body)
	keys := make([][]byte, 0, nKeys)
	for i := 0; i < nKeys; i++ {
		if len(body) < 32 {
			t.Fatalf("akun ke-%d terpotong", i)
		}
		keys = append(keys, body[:32])
		body = body[32:]
	}

	if len(body) < 32 {
		t.Fatalf("blockhash terpotong")
	}
	blockhash := body[:32]
	body = body[32:]

	nIx, body := shortVec(t, body)
	ixs := make([]instruction, 0, nIx)
	for i := 0; i < nIx; i++ {
		if len(body) < 1 {
			t.Fatalf("program index instruksi ke-%d terpotong", i)
		}
		programIndex := body[0]
		body = body[1:]

		var nAccounts int
		nAccounts, body = shortVec(t, body)
		if len(body) < nAccounts {
			t.Fatalf("daftar akun instruksi ke-%d terpotong", i)
		}
		accounts := body[:nAccounts]
		body = body[nAccounts:]

		var nData int
		nData, body = shortVec(t, body)
		if len(body) < nData {
			t.Fatalf("data instruksi ke-%d terpotong", i)
		}
		data := body[:nData]
		body = body[nData:]

		ixs = append(ixs, instruction{programIndex: programIndex, accounts: accounts, data: data})
	}

	if len(body) != 0 {
		t.Fatalf("ada %d byte sisa yang tidak terpakai", len(body))
	}

	return decodedTx{signature: signature, header: header, keys: keys, blockhash: blockhash, ixs: ixs, message: message}
}

// randAddress menghasilkan alamat base58 32 byte acak (bukan kunci sungguhan,
// hanya untuk mengisi field alamat).
func randAddress(t *testing.T) (string, []byte) {
	t.Helper()
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return base58Encode(b), b
}

// TestSignNativeTransfer verifies the produced transaction is well-formed and
// its ed25519 signature genuinely validates against the sender public key.
func TestSignNativeTransfer(t *testing.T) {
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		t.Fatalf("rand: %v", err)
	}
	pub := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)

	from, fromBytes := randAddress(t)
	to, toBytes := randAddress(t)
	blockhash, blockhashBytes := randAddress(t)
	lamports := uint64(1_500_000_000)

	txBase64, err := SignNativeTransfer(seed, from, to, lamports, blockhash)
	if err != nil {
		t.Fatalf("SignNativeTransfer: %v", err)
	}

	tx := decodeTx(t, txBase64)

	if !ed25519.Verify(pub, tx.message, tx.signature) {
		t.Fatalf("signature native tidak valid untuk pesan yang dihasilkan")
	}
	if !bytes.Equal(tx.header, []byte{1, 0, 1}) {
		t.Fatalf("header = %v, mau [1 0 1]", tx.header)
	}
	if len(tx.keys) != 3 {
		t.Fatalf("jumlah akun = %d, mau 3", len(tx.keys))
	}
	if !bytes.Equal(tx.keys[0], fromBytes) || !bytes.Equal(tx.keys[1], toBytes) {
		t.Fatalf("urutan akun salah: %x, %x", tx.keys[0], tx.keys[1])
	}
	systemBytes, _ := base58Decode(systemProgram)
	if !bytes.Equal(tx.keys[2], systemBytes) {
		t.Fatalf("akun ke-3 bukan System Program: %x", tx.keys[2])
	}
	if !bytes.Equal(tx.blockhash, blockhashBytes) {
		t.Fatalf("blockhash tidak cocok")
	}
	if len(tx.ixs) != 1 {
		t.Fatalf("jumlah instruksi = %d, mau 1", len(tx.ixs))
	}
	ix := tx.ixs[0]
	if ix.programIndex != 2 {
		t.Fatalf("program index = %d, mau 2 (System Program)", ix.programIndex)
	}
	if !bytes.Equal(ix.accounts, []byte{0, 1}) {
		t.Fatalf("akun instruksi = %v, mau [0 1]", ix.accounts)
	}
	if len(ix.data) != 12 {
		t.Fatalf("panjang data = %d, mau 12", len(ix.data))
	}
	if got := binary.LittleEndian.Uint32(ix.data[0:4]); got != 2 {
		t.Fatalf("instruksi = %d, mau 2 (Transfer)", got)
	}
	if got := binary.LittleEndian.Uint64(ix.data[4:12]); got != lamports {
		t.Fatalf("lamports = %d, mau %d", got, lamports)
	}
}

// TestSignSPLTransfer verifies SPL transfer layout and signature validity.
func TestSignSPLTransfer(t *testing.T) {
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		t.Fatalf("rand: %v", err)
	}
	pub := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)

	owner, ownerBytes := randAddress(t)
	source, sourceBytes := randAddress(t)
	dest, destBytes := randAddress(t)
	blockhash, _ := randAddress(t)
	amount := uint64(42_000_000)

	txBase64, err := SignSPLTransfer(seed, owner, source, dest, amount, blockhash)
	if err != nil {
		t.Fatalf("SignSPLTransfer: %v", err)
	}

	tx := decodeTx(t, txBase64)

	if !ed25519.Verify(pub, tx.message, tx.signature) {
		t.Fatalf("signature SPL tidak valid untuk pesan yang dihasilkan")
	}
	if !bytes.Equal(tx.header, []byte{1, 1, 1}) {
		t.Fatalf("header = %v, mau [1 1 1]", tx.header)
	}
	if len(tx.keys) != 4 {
		t.Fatalf("jumlah akun = %d, mau 4", len(tx.keys))
	}
	if !bytes.Equal(tx.keys[0], ownerBytes) ||
		!bytes.Equal(tx.keys[1], sourceBytes) ||
		!bytes.Equal(tx.keys[2], destBytes) {
		t.Fatalf("urutan akun SPL salah")
	}
	tokenBytes, _ := base58Decode(tokenProgram)
	if !bytes.Equal(tx.keys[3], tokenBytes) {
		t.Fatalf("akun ke-4 bukan Token Program: %x", tx.keys[3])
	}
	if len(tx.ixs) != 1 {
		t.Fatalf("jumlah instruksi = %d, mau 1", len(tx.ixs))
	}
	ix := tx.ixs[0]
	if ix.programIndex != 3 {
		t.Fatalf("program index = %d, mau 3 (Token Program)", ix.programIndex)
	}
	if !bytes.Equal(ix.accounts, []byte{1, 2, 0}) {
		t.Fatalf("akun instruksi = %v, mau [1 2 0] (source, dest, owner)", ix.accounts)
	}
	if len(ix.data) != 9 {
		t.Fatalf("panjang data = %d, mau 9", len(ix.data))
	}
	if ix.data[0] != 3 {
		t.Fatalf("instruksi = %d, mau 3 (Transfer)", ix.data[0])
	}
	if got := binary.LittleEndian.Uint64(ix.data[1:9]); got != amount {
		t.Fatalf("amount = %d, mau %d", got, amount)
	}
}

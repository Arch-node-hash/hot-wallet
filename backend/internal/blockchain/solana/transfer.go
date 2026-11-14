package solana

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

const systemProgram = "11111111111111111111111111111111"

// TransferRequest adalah parameter transfer SOL.
type TransferRequest struct {
	From   string
	To     string
	Amount uint64
}

// Transfer mengirim SOL.
// TODO: implementasikan pembuatan + signing transaksi.
func (c *Client) Transfer(ctx context.Context, req TransferRequest) (string, error) {
	return "", fmt.Errorf("solana: transfer belum diimplementasikan")
}

// LatestBlockhash mengambil blockhash terbaru yang dipakai sebagai recent
// blockhash transaksi.
func (c *Client) LatestBlockhash(ctx context.Context) (string, error) {
	var result struct {
		Value struct {
			Blockhash string `json:"blockhash"`
		} `json:"value"`
	}
	if err := c.call(ctx, "getLatestBlockhash", []any{map[string]string{"commitment": "confirmed"}}, &result); err != nil {
		return "", err
	}
	if result.Value.Blockhash == "" {
		return "", fmt.Errorf("solana: blockhash kosong")
	}
	return result.Value.Blockhash, nil
}

// SendTransaction menyiarkan transaksi yang sudah ditandatangani dan
// mengembalikan signature transaksi.
func (c *Client) SendTransaction(ctx context.Context, txBase64 string) (string, error) {
	if txBase64 == "" {
		return "", fmt.Errorf("solana: transaksi kosong")
	}
	var signature string
	if err := c.call(ctx, "sendTransaction", []any{
		txBase64,
		map[string]any{"encoding": "base64", "preflightCommitment": "confirmed"},
	}, &signature); err != nil {
		return "", err
	}
	return signature, nil
}

// instruction adalah satu instruksi Solana.
type instruction struct {
	programIndex byte
	accounts     []byte
	data         []byte
}

// SignNativeTransfer membuat dan menandatangani transaksi transfer SOL.
func SignNativeTransfer(
	seed []byte,
	from, to string,
	lamports uint64,
	recentBlockhash string,
) (string, error) {
	if len(seed) != 32 {
		return "", fmt.Errorf("solana: seed harus 32 byte")
	}
	if lamports == 0 {
		return "", fmt.Errorf("solana: jumlah transfer kosong")
	}

	fromBytes, err := base58Decode(from)
	if err != nil || len(fromBytes) != 32 {
		return "", fmt.Errorf("solana: alamat pengirim tidak valid")
	}
	toBytes, err := base58Decode(to)
	if err != nil || len(toBytes) != 32 {
		return "", fmt.Errorf("solana: alamat tujuan tidak valid")
	}
	systemBytes, err := base58Decode(systemProgram)
	if err != nil {
		return "", err
	}

	keys := [][]byte{fromBytes, toBytes, systemBytes}
	header := []byte{1, 0, 1} // 1 signer, 0 readonly signed, 1 readonly unsigned

	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data[0:4], 2) // SystemInstruction::Transfer
	binary.LittleEndian.PutUint64(data[4:12], lamports)

	ix := instruction{programIndex: 2, accounts: []byte{0, 1}, data: data}

	message, err := buildMessage(header, keys, recentBlockhash, []instruction{ix})
	if err != nil {
		return "", err
	}

	return signTransaction(seed, message)
}

// SignSPLTransfer membuat dan menandatangani transaksi transfer SPL token.
func SignSPLTransfer(
	seed []byte,
	owner, source, dest string,
	amount uint64,
	recentBlockhash string,
) (string, error) {
	if len(seed) != 32 {
		return "", fmt.Errorf("solana: seed harus 32 byte")
	}
	if amount == 0 {
		return "", fmt.Errorf("solana: jumlah transfer kosong")
	}

	ownerBytes, err := base58Decode(owner)
	if err != nil || len(ownerBytes) != 32 {
		return "", fmt.Errorf("solana: alamat owner tidak valid")
	}
	sourceBytes, err := base58Decode(source)
	if err != nil || len(sourceBytes) != 32 {
		return "", fmt.Errorf("solana: alamat sumber tidak valid")
	}
	destBytes, err := base58Decode(dest)
	if err != nil || len(destBytes) != 32 {
		return "", fmt.Errorf("solana: alamat tujuan tidak valid")
	}
	tokenBytes, err := base58Decode(tokenProgram)
	if err != nil {
		return "", err
	}

	// Signer (owner) dulu, lalu akun writable non-signer, lalu readonly non-signer.
	keys := [][]byte{ownerBytes, sourceBytes, destBytes, tokenBytes}
	header := []byte{1, 1, 1} // 1 signer (readonly), 0 writable signed, 1 readonly unsigned

	data := make([]byte, 9)
	data[0] = 3 // TokenInstruction::Transfer
	binary.LittleEndian.PutUint64(data[1:9], amount)

	ix := instruction{programIndex: 3, accounts: []byte{1, 2, 0}, data: data}

	message, err := buildMessage(header, keys, recentBlockhash, []instruction{ix})
	if err != nil {
		return "", err
	}

	return signTransaction(seed, message)
}

// buildMessage menyusun pesan transaksi legacy Solana.
func buildMessage(header []byte, keys [][]byte, recentBlockhash string, ixs []instruction) ([]byte, error) {
	blockhash, err := base58Decode(recentBlockhash)
	if err != nil || len(blockhash) != 32 {
		return nil, fmt.Errorf("solana: recent blockhash tidak valid")
	}

	msg := make([]byte, 0, 256)
	msg = append(msg, header...)

	msg = append(msg, encodeShortVec(len(keys))...)
	for _, key := range keys {
		msg = append(msg, key...)
	}

	msg = append(msg, blockhash...)

	msg = append(msg, encodeShortVec(len(ixs))...)
	for _, ix := range ixs {
		msg = append(msg, ix.programIndex)
		msg = append(msg, encodeShortVec(len(ix.accounts))...)
		msg = append(msg, ix.accounts...)
		msg = append(msg, encodeShortVec(len(ix.data))...)
		msg = append(msg, ix.data...)
	}

	return msg, nil
}

// signTransaction menandatangani pesan dan membungkusnya menjadi transaksi
// base64 (satu signature + pesan).
func signTransaction(seed []byte, message []byte) (string, error) {
	privateKey := ed25519.NewKeyFromSeed(seed)
	signature := ed25519.Sign(privateKey, message)
	if len(signature) != 64 {
		return "", fmt.Errorf("solana: signature tidak valid")
	}

	tx := make([]byte, 0, 1+64+len(message))
	tx = append(tx, encodeShortVec(1)...)
	tx = append(tx, signature...)
	tx = append(tx, message...)

	return base64.StdEncoding.EncodeToString(tx), nil
}

// encodeShortVec menyusun compact-u16 (shortvec) sesuai spesifikasi Solana.
func encodeShortVec(n int) []byte {
	out := make([]byte, 0, 2)
	for {
		b := byte(n & 0x7f)
		n >>= 7
		if n == 0 {
			out = append(out, b)
			break
		}
		out = append(out, b|0x80)
	}
	return out
}

package evm

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func decodeRawTx(t *testing.T, raw string) *types.Transaction {
	t.Helper()

	b, err := hex.DecodeString(strings.TrimPrefix(raw, "0x"))
	if err != nil {
		t.Fatalf("decode hex: %v", err)
	}
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(b); err != nil {
		t.Fatalf("decode tx: %v", err)
	}
	return tx
}

// TestSignTransferIsValidAndRecoversSender memastikan transaksi native yang
// dihasilkan benar-benar valid: bisa di-decode ulang dan pengirimnya pulih
// (ecrecover) menjadi alamat milik private key yang menandatangani.
func TestSignTransferIsValidAndRecoversSender(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	priv := crypto.FromECDSA(key)
	wantFrom := crypto.PubkeyToAddress(key.PublicKey)

	chainID := big.NewInt(1)
	to := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	amount := big.NewInt(123456789)
	gasPrice := big.NewInt(5_000_000_000)

	raw, err := SignTransfer(priv, chainID, 7, to, amount, 21000, gasPrice)
	if err != nil {
		t.Fatalf("SignTransfer: %v", err)
	}

	tx := decodeRawTx(t, raw)

	sender, err := types.Sender(types.NewEIP155Signer(chainID), tx)
	if err != nil {
		t.Fatalf("recover sender: %v", err)
	}
	if sender != wantFrom {
		t.Fatalf("sender = %s, mau %s", sender.Hex(), wantFrom.Hex())
	}
	if tx.To() == nil || *tx.To() != to {
		t.Fatalf("to = %v, mau %s", tx.To(), to.Hex())
	}
	if tx.Value().Cmp(amount) != 0 {
		t.Fatalf("value = %s, mau %s", tx.Value(), amount)
	}
	if tx.Nonce() != 7 {
		t.Fatalf("nonce = %d, mau 7", tx.Nonce())
	}
	if tx.Gas() != 21000 {
		t.Fatalf("gas = %d, mau 21000", tx.Gas())
	}
	if tx.GasPrice().Cmp(gasPrice) != 0 {
		t.Fatalf("gasPrice = %s, mau %s", tx.GasPrice(), gasPrice)
	}
	if len(tx.Data()) != 0 {
		t.Fatalf("transfer native tidak boleh punya data, dapat %d byte", len(tx.Data()))
	}
}

// TestSignERC20TransferEncodesCalldata memastikan transfer ERC-20 menuju
// kontrak token dengan calldata transfer(address,uint256) yang benar.
func TestSignERC20TransferEncodesCalldata(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	priv := crypto.FromECDSA(key)

	chainID := big.NewInt(56)
	token := common.HexToAddress("0x55d398326f99059fF775485246999027B3197955")
	to := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	amount := big.NewInt(1_000_000)

	raw, err := SignERC20Transfer(priv, chainID, 0, token, to, amount, 65000, big.NewInt(1))
	if err != nil {
		t.Fatalf("SignERC20Transfer: %v", err)
	}

	tx := decodeRawTx(t, raw)
	if tx.To() == nil || *tx.To() != token {
		t.Fatalf("to = %v, mau kontrak token %s", tx.To(), token.Hex())
	}
	if tx.Value().Sign() != 0 {
		t.Fatalf("value ERC-20 harus 0, dapat %s", tx.Value())
	}

	data := tx.Data()
	if len(data) != 68 {
		t.Fatalf("panjang calldata = %d, mau 68", len(data))
	}
	if hex.EncodeToString(data[:4]) != erc20TransferSelector {
		t.Fatalf("selector = %s, mau %s", hex.EncodeToString(data[:4]), erc20TransferSelector)
	}
	if !bytes.Equal(data[4:36], common.LeftPadBytes(to.Bytes(), 32)) {
		t.Fatalf("arg alamat tujuan salah: %x", data[4:36])
	}
	if !bytes.Equal(data[36:68], common.LeftPadBytes(amount.Bytes(), 32)) {
		t.Fatalf("arg jumlah salah: %x", data[36:68])
	}
}

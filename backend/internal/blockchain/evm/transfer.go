package evm

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// erc20TransferSelector adalah 4 byte pertama dari keccak256("transfer(address,uint256)").
const erc20TransferSelector = "a9059cbb"

// TransferRequest adalah parameter transfer native token.
type TransferRequest struct {
	From   string
	To     string
	Amount string
}

// Transfer mengirim native token.
// TODO: implementasikan signing + eth_sendRawTransaction.
func (c *Client) Transfer(ctx context.Context, req TransferRequest) (string, error) {
	return "", fmt.Errorf("evm: transfer belum diimplementasikan")
}

// SignTransfer membuat dan menandatangani transaksi transfer native (legacy)
// lalu mengembalikan raw transaction siap dikirim ke eth_sendRawTransaction.
func SignTransfer(
	privKey []byte,
	chainID *big.Int,
	nonce uint64,
	to common.Address,
	amount *big.Int,
	gasLimit uint64,
	gasPrice *big.Int,
) (string, error) {
	return signLegacy(privKey, chainID, nonce, to, amount, gasLimit, gasPrice, nil)
}

// SignERC20Transfer membuat dan menandatangani transaksi pemanggilan
// transfer(address,uint256) pada kontrak token ERC-20.
func SignERC20Transfer(
	privKey []byte,
	chainID *big.Int,
	nonce uint64,
	token common.Address,
	to common.Address,
	amount *big.Int,
	gasLimit uint64,
	gasPrice *big.Int,
) (string, error) {
	data, err := encodeERC20Transfer(to, amount)
	if err != nil {
		return "", err
	}
	return signLegacy(privKey, chainID, nonce, token, big.NewInt(0), gasLimit, gasPrice, data)
}

func signLegacy(
	privKey []byte,
	chainID *big.Int,
	nonce uint64,
	to common.Address,
	amount *big.Int,
	gasLimit uint64,
	gasPrice *big.Int,
	data []byte,
) (string, error) {
	if len(privKey) != 32 {
		return "", fmt.Errorf("evm: private key harus 32 byte")
	}
	if chainID == nil || chainID.Sign() == 0 {
		return "", fmt.Errorf("evm: chain id tidak valid")
	}
	if gasPrice == nil || gasPrice.Sign() <= 0 {
		return "", fmt.Errorf("evm: gas price tidak valid")
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    amount,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})

	key, err := crypto.ToECDSA(privKey)
	if err != nil {
		return "", fmt.Errorf("evm: private key tidak valid: %w", err)
	}

	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		return "", fmt.Errorf("evm: sign tx: %w", err)
	}

	raw, err := signed.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("evm: encode tx: %w", err)
	}
	return "0x" + hex.EncodeToString(raw), nil
}

// EncodeERC20Transfer menyusun calldata transfer(address,uint256) yang siap
// dipakai untuk eth_call/eth_estimateGas maupun transaksi ERC-20.
func EncodeERC20Transfer(to common.Address, amount *big.Int) ([]byte, error) {
	return encodeERC20Transfer(to, amount)
}

// encodeERC20Transfer menyusun calldata transfer(address,uint256).
func encodeERC20Transfer(to common.Address, amount *big.Int) ([]byte, error) {
	if amount == nil {
		return nil, fmt.Errorf("evm: jumlah transfer kosong")
	}
	if amount.Sign() < 0 {
		return nil, fmt.Errorf("evm: jumlah transfer negatif")
	}

	selector, err := hex.DecodeString(erc20TransferSelector)
	if err != nil {
		return nil, err
	}

	data := make([]byte, 0, 4+32+32)
	data = append(data, selector...)
	data = append(data, padLeft32(to.Bytes())...)
	data = append(data, padLeft32(amount.Bytes())...)
	return data, nil
}

// padLeft32 mengubah byte menjadi 32 byte dengan padding nol di kiri.
func padLeft32(b []byte) []byte {
	out := make([]byte, 32)
	if len(b) >= 32 {
		copy(out, b[len(b)-32:])
		return out
	}
	copy(out[32-len(b):], b)
	return out
}

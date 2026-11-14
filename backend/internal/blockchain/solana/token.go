package solana

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
)

const (
	tokenProgram     = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	token2022Program = "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb"
	ataProgram       = "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL"
)

// Token mendeskripsikan SPL token.
type Token struct {
	Mint     string `json:"mint"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
}

// TokenAccountRef adalah alamat Associated Token Account untuk satu mint.
type TokenAccountRef struct {
	Token   Token
	Address string
}

// TokenAccountRefs menurunkan alamat Associated Token Account (ATA) untuk
// setiap token yang dipantau. Pendekatan ini dipakai supaya saldo token bisa
// dibaca dengan getMultipleAccounts biasa, tanpa RPC "indexed" seperti
// getTokenAccountsByOwner yang diblokir di endpoint publik gratis.
func TokenAccountRefs(owner string, tokens []Token) []TokenAccountRef {
	ownerBytes, err := base58Decode(owner)
	if err != nil || len(ownerBytes) != 32 {
		return nil
	}

	programs := []string{tokenProgram, token2022Program}
	out := make([]TokenAccountRef, 0, len(tokens)*len(programs))

	for _, token := range tokens {
		mintBytes, err := base58Decode(token.Mint)
		if err != nil || len(mintBytes) != 32 {
			continue
		}
		for _, program := range programs {
			programBytes, err := base58Decode(program)
			if err != nil {
				continue
			}
			address, err := findProgramAddress([][]byte{ownerBytes, programBytes, mintBytes}, ataProgram)
			if err != nil {
				continue
			}
			out = append(out, TokenAccountRef{Token: token, Address: address})
		}
	}

	return out
}

// TokenAccountAddress menurunkan Associated Token Account (ATA) standar
// (Token Program) untuk satu mint. Dipakai saat mengirim SPL token.
func TokenAccountAddress(owner, mint string) (string, error) {
	ownerBytes, err := base58Decode(owner)
	if err != nil || len(ownerBytes) != 32 {
		return "", fmt.Errorf("solana: alamat owner tidak valid")
	}
	mintBytes, err := base58Decode(mint)
	if err != nil || len(mintBytes) != 32 {
		return "", fmt.Errorf("solana: mint tidak valid")
	}
	programBytes, err := base58Decode(tokenProgram)
	if err != nil {
		return "", err
	}
	return findProgramAddress([][]byte{ownerBytes, programBytes, mintBytes}, ataProgram)
}

// ValidAddress melaporkan apakah string adalah alamat Solana base58 32 byte.
func ValidAddress(address string) bool {
	decoded, err := base58Decode(address)
	return err == nil && len(decoded) == 32
}

// findProgramAddress menurunkan Program Derived Address (PDA): hash dari
// seeds + program id + "ProgramDerivedAddress", dengan bump yang menghasilkan
// titik di luar kurva ed25519.
func findProgramAddress(seeds [][]byte, programID string) (string, error) {
	programBytes, err := base58Decode(programID)
	if err != nil {
		return "", err
	}

	for bump := 255; bump >= 0; bump-- {
		hash := sha256.New()
		for _, seed := range seeds {
			hash.Write(seed)
		}
		hash.Write([]byte{byte(bump)})
		hash.Write(programBytes)
		hash.Write([]byte("ProgramDerivedAddress"))
		sum := hash.Sum(nil)

		if !isEdwardsPoint(sum) {
			return base58Encode(sum), nil
		}
	}
	return "", errors.New("solana: pda tidak ditemukan")
}

var (
	edP = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(19))
	edD = func() *big.Int {
		inv := new(big.Int).ModInverse(big.NewInt(121666), edP)
		if inv == nil {
			return big.NewInt(0)
		}
		d := new(big.Int).Mul(big.NewInt(-121665), inv)
		return d.Mod(d, edP)
	}()
)

// isEdwardsPoint melaporkan apakah 32 byte terkompresi merupakan titik valid
// pada kurva ed25519. Dipakai untuk memastikan hasil hash bukan titik kurva.
func isEdwardsPoint(compressed []byte) bool {
	if len(compressed) != 32 {
		return false
	}

	// y disimpan little-endian dengan bit tanda x di posisi tertinggi.
	be := make([]byte, 32)
	for i := 0; i < 32; i++ {
		be[i] = compressed[31-i]
	}
	be[0] &= 0x7f
	y := new(big.Int).SetBytes(be)
	if y.Cmp(edP) >= 0 {
		return false
	}

	y2 := new(big.Int).Mul(y, y)
	y2.Mod(y2, edP)

	// x^2 = (y^2 - 1) / (d*y^2 + 1) mod p
	num := new(big.Int).Sub(y2, big.NewInt(1))
	num.Mod(num, edP)

	den := new(big.Int).Mul(edD, y2)
	den.Add(den, big.NewInt(1))
	den.Mod(den, edP)

	denInv := new(big.Int).ModInverse(den, edP)
	if denInv == nil {
		return false
	}

	x2 := new(big.Int).Mul(num, denInv)
	x2.Mod(x2, edP)
	if x2.Sign() == 0 {
		return true
	}

	// x^2 punya akar kuadrat bila x2^((p-1)/2) == 1 (simbol Legendre).
	exp := new(big.Int).Rsh(new(big.Int).Sub(edP, big.NewInt(1)), 1)
	legendre := new(big.Int).Exp(x2, exp, edP)
	return legendre.Cmp(big.NewInt(1)) == 0
}

const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func base58Encode(input []byte) string {
	x := new(big.Int).SetBytes(input)
	base := big.NewInt(58)
	mod := new(big.Int)

	encoded := make([]byte, 0, 44)
	for x.Sign() > 0 {
		x.DivMod(x, base, mod)
		encoded = append(encoded, base58Alphabet[mod.Int64()])
	}
	for _, b := range input {
		if b != 0 {
			break
		}
		encoded = append(encoded, base58Alphabet[0])
	}

	for i, j := 0, len(encoded)-1; i < j; i, j = i+1, j-1 {
		encoded[i], encoded[j] = encoded[j], encoded[i]
	}
	return string(encoded)
}

func base58Decode(input string) ([]byte, error) {
	x := big.NewInt(0)
	base := big.NewInt(58)
	for _, r := range input {
		idx := -1
		for i := 0; i < len(base58Alphabet); i++ {
			if rune(base58Alphabet[i]) == r {
				idx = i
				break
			}
		}
		if idx < 0 {
			return nil, fmt.Errorf("solana: karakter base58 tidak valid: %q", r)
		}
		x.Mul(x, base)
		x.Add(x, big.NewInt(int64(idx)))
	}

	decoded := x.Bytes()
	leading := 0
	for leading < len(input) && input[leading] == base58Alphabet[0] {
		leading++
	}
	if leading > 0 {
		padded := make([]byte, leading+len(decoded))
		copy(padded[leading:], decoded)
		return padded, nil
	}
	return decoded, nil
}

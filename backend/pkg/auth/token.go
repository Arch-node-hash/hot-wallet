package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// ErrInvalidToken dikembalikan saat token tidak valid atau kedaluwarsa.
var ErrInvalidToken = errors.New("token tidak valid atau sudah kedaluwarsa")

type tokenHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// Claims adalah payload token unlock wallet.
type Claims struct {
	Subject   string `json:"sub"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// Generate membuat JWT HS256 untuk wallet yang sudah di-unlock.
func Generate(secret []byte, subject string, ttl time.Duration) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("jwt: secret kosong")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	now := time.Now()
	headerJSON, err := json.Marshal(tokenHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(Claims{
		Subject:   subject,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
	})
	if err != nil {
		return "", err
	}

	signingInput := encode(headerJSON) + "." + encode(claimsJSON)
	signature := sign(secret, signingInput)
	return signingInput + "." + encode(signature), nil
}

// Verify memvalidasi signature, header, dan masa berlaku token.
func Verify(secret []byte, token string) (*Claims, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}
	expected := sign(secret, parts[0]+"."+parts[1])
	if !hmac.Equal(signature, expected) {
		return nil, ErrInvalidToken
	}

	var header tokenHeader
	if err := decode(parts[0], &header); err != nil || header.Alg != "HS256" {
		return nil, ErrInvalidToken
	}

	var claims Claims
	if err := decode(parts[1], &claims); err != nil {
		return nil, ErrInvalidToken
	}
	if claims.Subject == "" || claims.ExpiresAt == 0 || time.Now().Unix() >= claims.ExpiresAt {
		return nil, ErrInvalidToken
	}
	return &claims, nil
}

// Bearer mengambil token dari header Authorization.
func Bearer(headerValue string) string {
	parts := strings.Fields(strings.TrimSpace(headerValue))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func sign(secret []byte, input string) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(input))
	return mac.Sum(nil)
}

func encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func decode(encoded string, dst any) error {
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

const hashRounds = 120_000

// HashPIN mengenkripsi PIN wallet menjadi hash SHA-256 berulang.
func HashPIN(pin string) string {
	sum := []byte(pin)
	for i := 0; i < hashRounds; i++ {
		digest := sha256.Sum256(sum)
		sum = digest[:]
	}
	return hex.EncodeToString(sum)
}

// VerifyPIN membandingkan PIN dengan hash secara constant time.
func VerifyPIN(pin, hash string) bool {
	if hash == "" {
		return false
	}
	computed := HashPIN(pin)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) == 1
}

// RandomHex menghasilkan n byte acak dalam bentuk hex.
func RandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

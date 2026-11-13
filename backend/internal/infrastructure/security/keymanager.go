package security

import "fmt"

// KeyManager membungkus operasi enkripsi dengan satu kunci.
type KeyManager struct {
	key []byte
}

// NewKeyManager menerima kunci AES 16, 24, atau 32 byte.
func NewKeyManager(key []byte) (*KeyManager, error) {
	switch len(key) {
	case 16, 24, 32:
	default:
		return nil, fmt.Errorf("security: panjang kunci harus 16, 24, atau 32 byte")
	}

	copied := make([]byte, len(key))
	copy(copied, key)
	return &KeyManager{key: copied}, nil
}

func (m *KeyManager) Encrypt(plaintext []byte) ([]byte, error) {
	return Encrypt(m.key, plaintext)
}

func (m *KeyManager) Decrypt(ciphertext []byte) ([]byte, error) {
	return Decrypt(m.key, ciphertext)
}

// Zero menghapus kunci dari memori.
func (m *KeyManager) Zero() {
	for i := range m.key {
		m.key[i] = 0
	}
}

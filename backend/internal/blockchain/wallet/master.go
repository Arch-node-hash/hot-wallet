package wallet

import (
	"crypto/rand"
	"fmt"
)

// MasterKey adalah kunci utama untuk menurunkan kunci wallet.
type MasterKey struct {
	Key []byte
}

// NewMasterKey membuat master key dari seed. Jika seed kosong, seed acak
// 32 byte akan dibuat.
func NewMasterKey(seed []byte) (*MasterKey, error) {
	if len(seed) == 0 {
		seed = make([]byte, 32)
		if _, err := rand.Read(seed); err != nil {
			return nil, fmt.Errorf("wallet: generate seed: %w", err)
		}
	}

	key := make([]byte, len(seed))
	copy(key, seed)
	return &MasterKey{Key: key}, nil
}

// Zero menghapus isi master key dari memori.
func (m *MasterKey) Zero() {
	for i := range m.Key {
		m.Key[i] = 0
	}
}

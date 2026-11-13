package wallet

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// DerivedKey adalah hasil turunan master key.
type DerivedKey struct {
	Index uint32
	Key   []byte
}

// Derive adalah placeholder deterministik. Ganti dengan BIP32/BIP44 sesuai
// kebutuhan chain sebelum dipakai di production.
func Derive(master *MasterKey, index uint32) (DerivedKey, error) {
	if master == nil || len(master.Key) == 0 {
		return DerivedKey{}, fmt.Errorf("wallet: master key kosong")
	}

	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], index)

	mac := hmac.New(sha256.New, master.Key)
	mac.Write(buf[:])

	return DerivedKey{
		Index: index,
		Key:   mac.Sum(nil),
	}, nil
}

package wallet

import (
	"fmt"
	"os"
	"path/filepath"
)

// Keystore menyimpan material kunci terenkripsi di disk.
type Keystore struct {
	dir string
}

func NewKeystore(dir string) *Keystore {
	return &Keystore{dir: dir}
}

func (k *Keystore) Save(id string, data []byte) error {
	if err := os.MkdirAll(k.dir, 0o700); err != nil {
		return fmt.Errorf("keystore: create dir: %w", err)
	}
	target := filepath.Join(k.dir, id+".key")
	if err := os.WriteFile(target, data, 0o600); err != nil {
		return fmt.Errorf("keystore: save %s: %w", id, err)
	}
	return nil
}

func (k *Keystore) Load(id string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(k.dir, id+".key"))
	if err != nil {
		return nil, fmt.Errorf("keystore: load %s: %w", id, err)
	}
	return data, nil
}

func (k *Keystore) Delete(id string) error {
	if err := os.Remove(filepath.Join(k.dir, id+".key")); err != nil {
		return fmt.Errorf("keystore: delete %s: %w", id, err)
	}
	return nil
}

package contact

import (
	"context"
	"path/filepath"
	"testing"

	"hotwallet/backend/internal/infrastructure/store"
)

func newTestService(t *testing.T) (*Service, *store.Store) {
	t.Helper()

	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	if err := store.InitSchema(st.DB()); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	return NewService(st), st
}

func TestListReturnsContactsSortedByName(t *testing.T) {
	svc, st := newTestService(t)
	ctx := context.Background()

	rows := []struct {
		id, name, address, network string
	}{
		{"c2", "Budi", "0x0000000000000000000000000000000000000002", "Ethereum Mainnet"},
		{"c1", "Ani", "0x0000000000000000000000000000000000000001", "BNB Chain"},
	}
	for _, r := range rows {
		if _, err := st.DB().ExecContext(ctx,
			`INSERT INTO contacts (id, name, address, network) VALUES (?, ?, ?, ?)`,
			r.id, r.name, r.address, r.network); err != nil {
			t.Fatalf("insert contact: %v", err)
		}
	}

	contacts, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(contacts) != 2 {
		t.Fatalf("jumlah kontak = %d, mau 2", len(contacts))
	}
	if contacts[0].Name != "Ani" || contacts[1].Name != "Budi" {
		t.Fatalf("kontak harus urut nama: %s, %s", contacts[0].Name, contacts[1].Name)
	}
	if contacts[0].ID != "c1" || contacts[0].Network != "BNB Chain" {
		t.Fatalf("data kontak tidak lengkap: %+v", contacts[0])
	}
}

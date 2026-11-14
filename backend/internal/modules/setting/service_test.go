package setting

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

func TestSecurityPinRoundTrip(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	initial, err := svc.GetSecurity(ctx)
	if err != nil {
		t.Fatalf("get security: %v", err)
	}
	if initial.PinEnabled {
		t.Fatal("PIN seharusnya belum aktif")
	}
	if ok, _ := svc.VerifyPin(ctx, "apa saja"); !ok {
		t.Fatal("tanpa PIN aktif, verifikasi harus lolos")
	}

	pin := "1234"
	updated, err := svc.UpdateSecurity(ctx, SecurityUpdate{Pin: &pin})
	if err != nil {
		t.Fatalf("set pin: %v", err)
	}
	if !updated.PinEnabled {
		t.Fatal("PIN seharusnya aktif setelah diset")
	}

	if ok, err := svc.VerifyPin(ctx, "1234"); err != nil || !ok {
		t.Fatalf("PIN benar ditolak: ok=%v err=%v", ok, err)
	}
	if ok, _ := svc.VerifyPin(ctx, "0000"); ok {
		t.Fatal("PIN salah seharusnya ditolak")
	}

	// PIN terlalu pendek harus ditolak.
	short := "12"
	if _, err := svc.UpdateSecurity(ctx, SecurityUpdate{Pin: &short}); err == nil {
		t.Fatal("PIN pendek seharusnya error")
	}

	removed, err := svc.UpdateSecurity(ctx, SecurityUpdate{RemovePin: true})
	if err != nil {
		t.Fatalf("remove pin: %v", err)
	}
	if removed.PinEnabled {
		t.Fatal("PIN seharusnya nonaktif setelah dihapus")
	}
	if ok, _ := svc.VerifyPin(ctx, "bebas"); !ok {
		t.Fatal("setelah PIN dihapus, verifikasi harus lolos")
	}
}

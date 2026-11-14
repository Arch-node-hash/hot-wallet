package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SecuritySettings adalah pengaturan PIN/biometric yang dipakai halaman
// Setting dan langkah konfirmasi Send.
type SecuritySettings struct {
	PinEnabled       bool
	PinHash          string
	BiometricEnabled bool
	AutoLockMinutes  int
	UpdatedAt        string
}

// GetSecuritySettings mengambil baris pengaturan keamanan tunggal. Baris
// dibuat default (PIN nonaktif) bila belum ada.
func (s *Store) GetSecuritySettings(ctx context.Context) (*SecuritySettings, error) {
	var (
		pinHash   sql.NullString
		biometric int
		autoLock  int
		updated   sql.NullString
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT pin_hash, biometric_enabled, auto_lock_minutes, updated_at
		FROM security_settings
		ORDER BY id
		LIMIT 1
	`).Scan(&pinHash, &biometric, &autoLock, &updated)
	if err != nil {
		if err == sql.ErrNoRows {
			return &SecuritySettings{PinEnabled: false, AutoLockMinutes: 5}, nil
		}
		return nil, fmt.Errorf("store: get security settings: %w", err)
	}

	return &SecuritySettings{
		PinEnabled:       pinHash.Valid && pinHash.String != "",
		PinHash:          pinHash.String,
		BiometricEnabled: biometric == 1,
		AutoLockMinutes:  autoLock,
		UpdatedAt:        updated.String,
	}, nil
}

// SaveSecuritySettings menyimpan pengaturan keamanan (satu baris).
func (s *Store) SaveSecuritySettings(ctx context.Context, settings SecuritySettings) error {
	now := time.Now().UTC().Format(time.RFC3339)
	biometric := 0
	if settings.BiometricEnabled {
		biometric = 1
	}

	var pinHash any
	if settings.PinHash != "" {
		pinHash = settings.PinHash
	}

	res, err := s.db.ExecContext(ctx, `
		UPDATE security_settings SET
			pin_hash = ?,
			biometric_enabled = ?,
			auto_lock_minutes = ?,
			updated_at = ?
	`, pinHash, biometric, settings.AutoLockMinutes, now)
	if err != nil {
		return fmt.Errorf("store: update security settings: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: security rows affected: %w", err)
	}
	if affected > 0 {
		return nil
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO security_settings (pin_hash, biometric_enabled, auto_lock_minutes, updated_at)
		VALUES (?, ?, ?, ?)
	`, pinHash, biometric, settings.AutoLockMinutes, now)
	if err != nil {
		return fmt.Errorf("store: insert security settings: %w", err)
	}
	return nil
}

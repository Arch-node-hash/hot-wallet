package setting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"hotwallet/backend/internal/infrastructure/store"
)

// Service mengelola pengaturan, termasuk keamanan (PIN/biometric).
type Service struct {
	store *store.Store
}

func NewService(st *store.Store) *Service { return &Service{store: st} }

// Security adalah status keamanan yang aman dikirim ke klien (tanpa hash PIN).
type Security struct {
	PinEnabled       bool   `json:"pin_enabled"`
	BiometricEnabled bool   `json:"biometric_enabled"`
	AutoLockMinutes  int    `json:"auto_lock_minutes"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

// SecurityUpdate adalah perubahan pengaturan keamanan.
type SecurityUpdate struct {
	Pin              *string `json:"pin,omitempty"`
	RemovePin        bool    `json:"remove_pin,omitempty"`
	BiometricEnabled *bool   `json:"biometric_enabled,omitempty"`
	AutoLockMinutes  *int    `json:"auto_lock_minutes,omitempty"`
}

// GetSecurity mengambil pengaturan keamanan saat ini.
func (s *Service) GetSecurity(ctx context.Context) (*Security, error) {
	settings, err := s.store.GetSecuritySettings(ctx)
	if err != nil {
		return nil, err
	}
	return &Security{
		PinEnabled:       settings.PinEnabled,
		BiometricEnabled: settings.BiometricEnabled,
		AutoLockMinutes:  settings.AutoLockMinutes,
		UpdatedAt:        settings.UpdatedAt,
	}, nil
}

// UpdateSecurity menyimpan pengaturan keamanan.
func (s *Service) UpdateSecurity(ctx context.Context, req SecurityUpdate) (*Security, error) {
	settings, err := s.store.GetSecuritySettings(ctx)
	if err != nil {
		return nil, err
	}

	if req.Pin != nil {
		pin := strings.TrimSpace(*req.Pin)
		if len(pin) < 4 {
			return nil, fmt.Errorf("PIN minimal 4 karakter")
		}
		settings.PinHash = hashPIN(pin)
	}
	if req.RemovePin {
		settings.PinHash = ""
	}
	if req.BiometricEnabled != nil {
		settings.BiometricEnabled = *req.BiometricEnabled
	}
	if req.AutoLockMinutes != nil && *req.AutoLockMinutes > 0 {
		settings.AutoLockMinutes = *req.AutoLockMinutes
	}

	if err := s.store.SaveSecuritySettings(ctx, *settings); err != nil {
		return nil, err
	}
	return s.GetSecurity(ctx)
}

// VerifyPin melaporkan apakah PIN cocok dengan yang tersimpan.
func (s *Service) VerifyPin(ctx context.Context, pin string) (bool, error) {
	settings, err := s.store.GetSecuritySettings(ctx)
	if err != nil {
		return false, err
	}
	if !settings.PinEnabled {
		return true, nil
	}
	return hashPIN(pin) == settings.PinHash, nil
}

func hashPIN(pin string) string {
	sum := sha256.Sum256([]byte(pin))
	return hex.EncodeToString(sum[:])
}

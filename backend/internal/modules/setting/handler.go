package setting

import (
	"hotwallet/backend/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// Handler adalah skeleton endpoint Setting.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// CreateWallet membuat wallet baru.
func (h *Handler) CreateWallet(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "settings/wallets/create")
}

// ImportWallet mengimpor wallet dari mnemonic/private key.
func (h *Handler) ImportWallet(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "settings/wallets/import")
}

// ListRPC menampilkan konfigurasi RPC/WSS per jaringan.
func (h *Handler) ListRPC(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "settings/rpc")
}

// UpdateRPC mengubah konfigurasi RPC/WSS.
func (h *Handler) UpdateRPC(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "settings/rpc")
}

// GetSecurity menampilkan pengaturan keamanan.
func (h *Handler) GetSecurity(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "settings/security")
}

// UpdateSecurity mengubah pengaturan keamanan.
func (h *Handler) UpdateSecurity(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "settings/security")
}

// GetBackup menampilkan info/riwayat backup.
func (h *Handler) GetBackup(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "settings/backup")
}

// CreateBackup membuat backup baru.
func (h *Handler) CreateBackup(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "settings/backup")
}

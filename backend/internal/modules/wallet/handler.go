package wallet

import (
	"hotwallet/backend/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// Handler adalah skeleton endpoint Wallet.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List menampilkan daftar wallet.
func (h *Handler) List(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "wallets")
}

// Create menambah wallet.
func (h *Handler) Create(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "wallets")
}

// Get menampilkan detail wallet (termasuk alamat).
func (h *Handler) Get(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "wallets")
}

// Balance menampilkan saldo wallet dan ringkasan per jaringan.
func (h *Handler) Balance(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "wallets/balance")
}

// Send mengirim aset dari wallet.
func (h *Handler) Send(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "wallets/send")
}

// Receive menampilkan alamat/QR untuk menerima aset.
func (h *Handler) Receive(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "wallets/receive")
}

// Delete menghapus wallet.
func (h *Handler) Delete(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "wallets")
}

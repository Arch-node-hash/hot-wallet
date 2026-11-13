package contact

import (
	"hotwallet/backend/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// Handler adalah skeleton endpoint Contact.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List menampilkan daftar kontak.
func (h *Handler) List(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "contacts")
}

// Create menyimpan kontak alamat wallet.
func (h *Handler) Create(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "contacts")
}

// Get menampilkan detail kontak.
func (h *Handler) Get(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "contacts")
}

// Update mengubah kontak.
func (h *Handler) Update(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "contacts")
}

// Delete menghapus kontak.
func (h *Handler) Delete(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "contacts")
}

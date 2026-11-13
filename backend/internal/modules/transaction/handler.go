package transaction

import (
	"hotwallet/backend/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// Handler adalah skeleton endpoint Transaction.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List menampilkan riwayat transaksi.
func (h *Handler) List(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "transactions")
}

// Get menampilkan detail transaksi.
func (h *Handler) Get(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "transactions")
}

// Status menampilkan status transaksi.
func (h *Handler) Status(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "transactions/status")
}

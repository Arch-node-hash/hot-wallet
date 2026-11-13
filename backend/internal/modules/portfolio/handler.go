package portfolio

import (
	"hotwallet/backend/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// Handler adalah skeleton endpoint Portfolio.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Summary menampilkan total aset.
func (h *Handler) Summary(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "portfolio/summary")
}

// Tokens menampilkan daftar token dan saldo.
func (h *Handler) Tokens(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "portfolio/tokens")
}

// Networks menampilkan rincian aset per jaringan.
func (h *Handler) Networks(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusNotImplemented, "portfolio/networks")
}

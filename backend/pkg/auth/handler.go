package auth

import (
	"hotwallet/backend/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// Handler adalah skeleton endpoint Auth.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Me(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "auth/me")
}

func (h *Handler) Logout(c fiber.Ctx) error {
	return response.SendMessage(c, fiber.StatusOK, "logged out")
}

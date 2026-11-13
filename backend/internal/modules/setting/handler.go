package setting

import (
	"hotwallet/backend/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// Handler adalah skeleton endpoint Settings.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Get(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "settings")
}

func (h *Handler) Update(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "settings")
}

func (h *Handler) UploadAvatar(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "settings/avatar")
}

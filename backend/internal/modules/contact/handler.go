package contact

import (
	"hotwallet/backend/internal/shared/models"
	"hotwallet/backend/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// Handler adalah endpoint Contact.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List GET /api/v1/contacts
func (h *Handler) List(c fiber.Ctx) error {
	contacts, err := h.svc.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}
	return c.JSON(models.APIResponse{Success: true, Data: contacts})
}

func (h *Handler) Get(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "team")
}

func (h *Handler) Create(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusCreated, "team")
}

func (h *Handler) Update(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "team")
}

func (h *Handler) Patch(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "team")
}

func (h *Handler) Delete(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "team")
}

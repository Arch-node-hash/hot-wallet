package setting

import (
	"hotwallet/backend/internal/shared/models"
	"hotwallet/backend/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// Handler adalah endpoint Settings.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Get(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "settings")
}

func (h *Handler) Update(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "settings")
}

// GetSecurity GET /api/v1/settings/security
func (h *Handler) GetSecurity(c fiber.Ctx) error {
	data, err := h.svc.GetSecurity(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}
	return c.JSON(models.APIResponse{Success: true, Data: data})
}

// UpdateSecurity PUT /api/v1/settings/security
func (h *Handler) UpdateSecurity(c fiber.Ctx) error {
	var req SecurityUpdate
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
	}

	data, err := h.svc.UpdateSecurity(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}
	return c.JSON(models.APIResponse{Success: true, Message: "pengaturan keamanan disimpan", Data: data})
}

func (h *Handler) UploadAvatar(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "settings/avatar")
}

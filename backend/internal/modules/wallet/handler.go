package wallet

import (
	"hotwallet/backend/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// Handler adalah skeleton endpoint Dashboard.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Get(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "dashboard")
}

// TimeEntryHandler adalah skeleton endpoint Time Entry.
type TimeEntryHandler struct{ svc *TimeEntryService }

func NewTimeEntryHandler(svc *TimeEntryService) *TimeEntryHandler {
	return &TimeEntryHandler{svc: svc}
}

func (h *TimeEntryHandler) List(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "time-entries")
}

func (h *TimeEntryHandler) Get(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "time-entries")
}

func (h *TimeEntryHandler) Create(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusCreated, "time-entries")
}

func (h *TimeEntryHandler) Update(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "time-entries")
}

func (h *TimeEntryHandler) Patch(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "time-entries")
}

func (h *TimeEntryHandler) Delete(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "time-entries")
}

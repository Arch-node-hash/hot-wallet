package activity

import (
	"hotwallet/backend/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// Handler adalah skeleton endpoint Event.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) List(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "calendar/events")
}

func (h *Handler) Get(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "calendar/events")
}

func (h *Handler) Create(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusCreated, "calendar/events")
}

func (h *Handler) Update(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "calendar/events")
}

func (h *Handler) Patch(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "calendar/events")
}

func (h *Handler) Delete(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "calendar/events")
}

// ReminderHandler adalah skeleton endpoint Reminder.
type ReminderHandler struct{ svc *ReminderService }

func NewReminderHandler(svc *ReminderService) *ReminderHandler { return &ReminderHandler{svc: svc} }

func (h *ReminderHandler) List(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "reminders")
}

func (h *ReminderHandler) Get(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "reminders")
}

func (h *ReminderHandler) Create(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusCreated, "reminders")
}

func (h *ReminderHandler) Update(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "reminders")
}

func (h *ReminderHandler) Patch(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "reminders")
}

func (h *ReminderHandler) Delete(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "reminders")
}

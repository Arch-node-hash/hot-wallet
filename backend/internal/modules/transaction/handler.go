package transaction

import (
	"hotwallet/backend/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// Handler adalah skeleton endpoint Task.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) List(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "tasks")
}

func (h *Handler) Get(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "tasks")
}

func (h *Handler) Create(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusCreated, "tasks")
}

func (h *Handler) Update(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "tasks")
}

func (h *Handler) Patch(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "tasks")
}

func (h *Handler) Delete(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "tasks")
}

// ProjectHandler adalah skeleton endpoint Project.
type ProjectHandler struct{ svc *ProjectService }

func NewProjectHandler(svc *ProjectService) *ProjectHandler { return &ProjectHandler{svc: svc} }

func (h *ProjectHandler) List(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "projects")
}

func (h *ProjectHandler) Get(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "projects")
}

func (h *ProjectHandler) Create(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusCreated, "projects")
}

func (h *ProjectHandler) Update(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "projects")
}

func (h *ProjectHandler) Patch(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "projects")
}

func (h *ProjectHandler) Delete(c fiber.Ctx) error {
	return response.SendPlaceholder(c, fiber.StatusOK, "projects")
}

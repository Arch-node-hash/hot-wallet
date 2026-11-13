package health

import (
	"hotwallet/backend/internal/infrastructure/store"

	"github.com/gofiber/fiber/v3"
)

// Handler memeriksa status API dan koneksi database.
type Handler struct {
	store *store.Store
}

func NewHandler(st *store.Store) *Handler {
	return &Handler{store: st}
}

func (h *Handler) Get(c fiber.Ctx) error {
	database := "ok"
	if err := h.store.Ping(); err != nil {
		database = "unavailable"
	}

	return c.JSON(fiber.Map{
		"status":   "ok",
		"database": database,
		"db_path":  h.store.Path(),
	})
}

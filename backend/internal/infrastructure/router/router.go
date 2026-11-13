package router

import (
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	static "github.com/gofiber/fiber/v3/middleware/static"

	"hotwallet/backend/config"
	"hotwallet/backend/internal/infrastructure/store"
)

// New merakit aplikasi Fiber beserta seluruh route skeleton.
func New(cfg config.Config, st *store.Store) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "HotWallet API",
		BodyLimit:    1 << 20,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} ${latency} ${method} ${path}\n",
		TimeFormat: "15:04:05",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}))

	// Services (placeholder - nanti akan diimplementasikan)
	// TODO: Implement service interfaces dari internal/shared/interfaces

	// API v1 Routes (menggunakan struktur 5 halaman)
	setupAPIV1Routes(app)

	// Avatar disimpan di data/avatars.
	app.Get("/avatars/*", static.New(filepath.Join(cfg.DataDir, "avatars")))

	// Info root (API-only, tidak ada halaman frontend).
	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"name":    "HotWallet API",
			"version": "1.0.0",
			"status":  "ok",
			"health":  "/api/v1/health",
			"api_v1":  "/api/v1",
			"pages": []string{
				"Wallet    - /api/v1/wallets",
				"Portfolio - /api/v1/portfolio",
				"Transaction - /api/v1/transactions",
				"Contact   - /api/v1/contacts",
				"Setting   - /api/v1/settings",
			},
			"message": "Hot Wallet Backend dengan struktur 5 halaman",
		})
	})

	// Fallback API
	app.All("/api/*", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	})

	return app
}

// setupAPIV1Routes mengatur semua route untuk API v1
func setupAPIV1Routes(app *fiber.App) {
	// API v1 routes
	apiV1 := app.Group("/api/v1")

	// Health check
	apiV1.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "timestamp": time.Now().UTC()})
	})

	// Wallet endpoints
	walletGroup := apiV1.Group("/wallets")
	walletGroup.Get("/", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Wallet list - not implemented"})
	})
	walletGroup.Post("/", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Wallet create - not implemented"})
	})

	walletIDGroup := walletGroup.Group("/:walletID")
	walletIDGroup.Get("/", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Wallet detail - not implemented"})
	})
	walletIDGroup.Get("/balance", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Wallet balance - not implemented"})
	})
	walletIDGroup.Post("/send", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Wallet send - not implemented"})
	})
	walletIDGroup.Get("/receive", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Wallet receive - not implemented"})
	})
	walletIDGroup.Delete("/", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Wallet delete - not implemented"})
	})

	// Portfolio endpoints
	portfolioGroup := apiV1.Group("/portfolio")
	portfolioGroup.Get("/summary", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Portfolio summary - not implemented"})
	})
	portfolioGroup.Get("/tokens", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Portfolio tokens - not implemented"})
	})
	portfolioGroup.Get("/networks", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Portfolio networks - not implemented"})
	})

	// Transaction endpoints
	transactionGroup := apiV1.Group("/transactions")
	transactionGroup.Get("/", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Transaction list - not implemented"})
	})
	transactionGroup.Get("/:txHash", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Transaction detail - not implemented"})
	})
	transactionGroup.Get("/:txHash/status", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Transaction status - not implemented"})
	})

	// Contact endpoints
	contactGroup := apiV1.Group("/contacts")
	contactGroup.Get("/", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Contact list - not implemented"})
	})
	contactGroup.Post("/", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Contact create - not implemented"})
	})

	contactIDGroup := contactGroup.Group("/:contactID")
	contactIDGroup.Get("/", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Contact detail - not implemented"})
	})
	contactIDGroup.Put("/", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Contact update - not implemented"})
	})
	contactIDGroup.Delete("/", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Contact delete - not implemented"})
	})

	// Setting endpoints
	settingGroup := apiV1.Group("/settings")
	settingGroup.Post("/wallets/create", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Setting create wallet - not implemented"})
	})
	settingGroup.Post("/wallets/import", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Setting import wallet - not implemented"})
	})
	settingGroup.Get("/rpc", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Setting RPC list - not implemented"})
	})
	settingGroup.Put("/rpc", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Setting RPC update - not implemented"})
	})
	settingGroup.Get("/security", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Setting security get - not implemented"})
	})
	settingGroup.Put("/security", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Setting security update - not implemented"})
	})
	settingGroup.Get("/backup", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Setting backup get - not implemented"})
	})
	settingGroup.Post("/backup", func(c fiber.Ctx) error {
		return c.Status(501).JSON(fiber.Map{"message": "Setting backup create - not implemented"})
	})
}

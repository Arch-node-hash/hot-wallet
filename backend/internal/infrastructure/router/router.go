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
	"hotwallet/backend/internal/modules/contact"
	"hotwallet/backend/internal/modules/portfolio"
	"hotwallet/backend/internal/modules/setting"
	"hotwallet/backend/internal/modules/transaction"
	"hotwallet/backend/internal/modules/wallet"
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

	// API v1 Routes (5 fitur: Wallet, Portfolio, Transaction, Contact, Setting)
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

// setupAPIV1Routes mengatur semua route untuk API v1.
func setupAPIV1Routes(app *fiber.App) {
	apiV1 := app.Group("/api/v1")

	// Health check
	apiV1.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "timestamp": time.Now().UTC()})
	})

	// Wallet
	walletH := wallet.NewHandler(wallet.NewService())
	walletGroup := apiV1.Group("/wallets")
	walletGroup.Get("/", walletH.List)
	walletGroup.Post("/", walletH.Create)

	walletIDGroup := walletGroup.Group("/:walletID")
	walletIDGroup.Get("/", walletH.Get)
	walletIDGroup.Delete("/", walletH.Delete)
	walletIDGroup.Get("/balance", walletH.Balance)
	walletIDGroup.Post("/send", walletH.Send)
	walletIDGroup.Get("/receive", walletH.Receive)

	// Portfolio
	portfolioH := portfolio.NewHandler(portfolio.NewService())
	portfolioGroup := apiV1.Group("/portfolio")
	portfolioGroup.Get("/summary", portfolioH.Summary)
	portfolioGroup.Get("/tokens", portfolioH.Tokens)
	portfolioGroup.Get("/networks", portfolioH.Networks)

	// Transaction
	transactionH := transaction.NewHandler(transaction.NewService())
	transactionGroup := apiV1.Group("/transactions")
	transactionGroup.Get("/", transactionH.List)
	transactionGroup.Get("/:txHash", transactionH.Get)
	transactionGroup.Get("/:txHash/status", transactionH.Status)

	// Contact
	contactH := contact.NewHandler(contact.NewService())
	contactGroup := apiV1.Group("/contacts")
	contactGroup.Get("/", contactH.List)
	contactGroup.Post("/", contactH.Create)

	contactIDGroup := contactGroup.Group("/:contactID")
	contactIDGroup.Get("/", contactH.Get)
	contactIDGroup.Put("/", contactH.Update)
	contactIDGroup.Delete("/", contactH.Delete)

	// Setting
	settingH := setting.NewHandler(setting.NewService())
	settingGroup := apiV1.Group("/settings")
	settingGroup.Post("/wallets/create", settingH.CreateWallet)
	settingGroup.Post("/wallets/import", settingH.ImportWallet)
	settingGroup.Get("/rpc", settingH.ListRPC)
	settingGroup.Put("/rpc", settingH.UpdateRPC)
	settingGroup.Get("/security", settingH.GetSecurity)
	settingGroup.Put("/security", settingH.UpdateSecurity)
	settingGroup.Get("/backup", settingH.GetBackup)
	settingGroup.Post("/backup", settingH.CreateBackup)
}

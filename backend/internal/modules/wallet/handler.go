package wallet

import (
	"hotwallet/backend/internal/shared/models"

	"github.com/gofiber/fiber/v3"
)

// Handler menangani HTTP request untuk wallet
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListWallets GET /api/v1/wallets
func (h *Handler) ListWallets(c fiber.Ctx) error {
	ctx := c.Context()
	wallets, err := h.svc.ListWallets(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}
	return c.JSON(models.APIResponse{Success: true, Data: wallets})
}

// GetWallet GET /api/v1/wallets/:walletID
func (h *Handler) GetWallet(c fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("walletID")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "walletID is required",
		})
	}

	wallet, err := h.svc.GetWallet(ctx, id)
	if err != nil {
		if err.Error() == "wallet not found" {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Error:   "wallet not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}
	return c.JSON(models.APIResponse{Success: true, Data: wallet})
}

// CreateWallet POST /api/v1/wallets
func (h *Handler) CreateWallet(c fiber.Ctx) error {
	ctx := c.Context()

	var req CreateWalletRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
	}

	// Validasi
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "name is required",
		})
	}
	if req.Network == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "network is required",
		})
	}

	wallet, err := h.svc.CreateWallet(ctx, req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{Success: true, Data: wallet})
}

// UpdateWalletAddress PUT /api/v1/wallets/:walletID/address
func (h *Handler) UpdateWalletAddress(c fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("walletID")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "walletID is required",
		})
	}

	var req UpdateWalletAddressRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
	}

	// Validasi
	if req.PrivateKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "private_key is required",
		})
	}
	if req.Network == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "network is required",
		})
	}

	wallet, err := h.svc.UpdateWalletAddress(ctx, id, req)
	if err != nil {
		if err.Error() == "wallet not found" {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Error:   "wallet not found",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	return c.JSON(models.APIResponse{Success: true, Data: wallet})
}

// DeleteWallet DELETE /api/v1/wallets/:walletID
func (h *Handler) DeleteWallet(c fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("walletID")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "walletID is required",
		})
	}

	err := h.svc.DeleteWallet(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	return c.JSON(models.APIResponse{Success: true, Message: "wallet deleted"})
}

// GetWalletBalance GET /api/v1/wallets/:walletID/balance
func (h *Handler) GetWalletBalance(c fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("walletID")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "walletID is required",
		})
	}

	wallet, err := h.svc.GetWallet(ctx, id)
	if err != nil {
		if err.Error() == "wallet not found" {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Error:   "wallet not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	// Return hanya network_balances
	return c.JSON(models.APIResponse{Success: true, Data: wallet.NetworkBalances})
}

// GetWalletReceive GET /api/v1/wallets/:walletID/receive
func (h *Handler) GetWalletReceive(c fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("walletID")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "walletID is required",
		})
	}

	wallet, err := h.svc.GetWallet(ctx, id)
	if err != nil {
		if err.Error() == "wallet not found" {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Error:   "wallet not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	// Return info receive (address, network, QR code data)
	receiveInfo := map[string]interface{}{
		"address":        wallet.Address,
		"network":        wallet.Network,
		"derivation_path": wallet.DerivationPath,
		"qr_data":        wallet.Address, // untuk QR code
	}

	return c.JSON(models.APIResponse{Success: true, Data: receiveInfo})
}
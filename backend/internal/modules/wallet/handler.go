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
		"address":         wallet.Address,
		"network":         wallet.Network,
		"derivation_path": wallet.DerivationPath,
		"qr_data":         wallet.Address, // untuk QR code
	}

	return c.JSON(models.APIResponse{Success: true, Data: receiveInfo})
}

// ---------------------------------------------------------------------------
// Wallet page endpoints
// ---------------------------------------------------------------------------

// Overview GET /api/v1/wallets/overview
func (h *Handler) Overview(c fiber.Ctx) error {
	data, err := h.svc.Overview(c.Context())
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(models.APIResponse{Success: true, Data: data})
}

// Addresses GET /api/v1/wallets/addresses
func (h *Handler) Addresses(c fiber.Ctx) error {
	data, err := h.svc.Addresses(c.Context())
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(models.APIResponse{Success: true, Data: data})
}

// Networks GET /api/v1/wallets/networks
func (h *Handler) Networks(c fiber.Ctx) error {
	data, err := h.svc.Networks(c.Context())
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(models.APIResponse{Success: true, Data: data})
}

// RefreshAll POST /api/v1/wallets/refresh
func (h *Handler) RefreshAll(c fiber.Ctx) error {
	data, err := h.svc.RefreshAll(c.Context())
	if err != nil {
		return fail(c, fiber.StatusBadGateway, err)
	}
	return c.JSON(models.APIResponse{Success: true, Message: "saldo diperbarui", Data: data})
}

// RefreshWallet POST /api/v1/wallets/:walletID/refresh
func (h *Handler) RefreshWallet(c fiber.Ctx) error {
	id := c.Params("walletID")
	if id == "" {
		return fail(c, fiber.StatusBadRequest, errString("walletID is required"))
	}
	data, err := h.svc.RefreshWallet(c.Context(), id)
	if err != nil {
		return fail(c, statusFor(err), err)
	}
	return c.JSON(models.APIResponse{Success: true, Message: "saldo diperbarui", Data: data})
}

// Rotate POST /api/v1/wallets/:walletID/rotate
func (h *Handler) Rotate(c fiber.Ctx) error {
	id := c.Params("walletID")
	if id == "" {
		return fail(c, fiber.StatusBadRequest, errString("walletID is required"))
	}
	data, err := h.svc.Rotate(c.Context(), id)
	if err != nil {
		return fail(c, statusFor(err), err)
	}
	return c.JSON(models.APIResponse{Success: true, Message: "alamat baru dibuat", Data: data})
}

// SendEstimate POST /api/v1/wallets/:walletID/send/estimate
func (h *Handler) SendEstimate(c fiber.Ctx) error {
	id := c.Params("walletID")
	if id == "" {
		return fail(c, fiber.StatusBadRequest, errString("walletID is required"))
	}
	var req SendEstimateRequest
	if err := c.Bind().Body(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, errString("invalid request body: "+err.Error()))
	}
	data, err := h.svc.SendEstimate(c.Context(), id, req)
	if err != nil {
		return fail(c, statusFor(err), err)
	}
	return c.JSON(models.APIResponse{Success: true, Data: data})
}

// Send POST /api/v1/wallets/:walletID/send
func (h *Handler) Send(c fiber.Ctx) error {
	id := c.Params("walletID")
	if id == "" {
		return fail(c, fiber.StatusBadRequest, errString("walletID is required"))
	}
	var req SendRequest
	if err := c.Bind().Body(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, errString("invalid request body: "+err.Error()))
	}
	data, err := h.svc.Send(c.Context(), id, req)
	if err != nil {
		return fail(c, statusFor(err), err)
	}
	return c.JSON(models.APIResponse{Success: true, Message: "transaksi dikirim", Data: data})
}

type errString string

func (e errString) Error() string { return string(e) }

func fail(c fiber.Ctx, status int, err error) error {
	return c.Status(status).JSON(models.APIResponse{Success: false, Error: err.Error()})
}

func statusFor(err error) int {
	switch {
	case err == nil:
		return fiber.StatusOK
	case err.Error() == "wallet not found":
		return fiber.StatusNotFound
	default:
		return fiber.StatusBadRequest
	}
}

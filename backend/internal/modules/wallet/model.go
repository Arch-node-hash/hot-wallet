package wallet

import "hotwallet/backend/internal/shared/models"

// Model wallet memakai tipe bersama di internal/shared/models supaya
// request/response konsisten dengan interface service.

type Wallet = models.Wallet
type WalletBalance = models.WalletBalance
type NetworkSummary = models.NetworkSummary

type CreateWalletRequest = models.CreateWalletRequest
type ImportWalletRequest = models.ImportWalletRequest
type SendRequest = models.SendRequest

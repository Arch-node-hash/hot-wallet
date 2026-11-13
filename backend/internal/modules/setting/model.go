package setting

import "hotwallet/backend/internal/shared/models"

// Model setting memakai tipe bersama di internal/shared/models.

type Setting = models.Setting
type RPCConfig = models.RPCConfig
type SecuritySetting = models.SecuritySetting
type BackupHistory = models.BackupHistory

type CreateWalletRequest = models.CreateWalletRequest
type ImportWalletRequest = models.ImportWalletRequest
type UpdateSecurityRequest = models.UpdateSecurityRequest
type BackupRequest = models.BackupRequest

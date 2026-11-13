package setting

type Notification struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

type Settings struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	FullName      string         `json:"full_name"`
	Email         string         `json:"email"`
	WalletAddress string         `json:"wallet_address"`
	Avatar        string         `json:"avatar"`
	Notifications []Notification `json:"notifications"`
	DarkMode      bool           `json:"dark_mode"`
}

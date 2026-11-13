package types

// ID adalah tipe umum untuk identifier.
type ID string

// Pagination menyimpan metadata pagination.
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Page adalah bentuk response list standar.
type Page struct {
	Data any         `json:"data"`
	Meta *Pagination `json:"meta,omitempty"`
}

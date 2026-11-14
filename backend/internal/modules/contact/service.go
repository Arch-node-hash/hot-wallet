package contact

import (
	"context"

	"hotwallet/backend/internal/infrastructure/store"
)

// Service mengelola contact book yang dipakai pemilih tujuan saat kirim.
type Service struct {
	store *store.Store
}

func NewService(st *store.Store) *Service { return &Service{store: st} }

// Contact adalah satu entri contact book.
type Contact struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	Network   string `json:"network"`
	Notes     string `json:"notes,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// List mengembalikan seluruh kontak.
func (s *Service) List(ctx context.Context) ([]Contact, error) {
	rows, err := s.store.ListContacts(ctx)
	if err != nil {
		return nil, err
	}

	contacts := make([]Contact, 0, len(rows))
	for _, row := range rows {
		contacts = append(contacts, Contact{
			ID:        row.ID,
			Name:      row.Name,
			Address:   row.Address,
			Network:   row.Network,
			Notes:     row.Notes,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return contacts, nil
}

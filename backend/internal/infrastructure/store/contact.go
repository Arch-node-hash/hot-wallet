package store

import (
	"context"
	"fmt"
)

// ContactRow adalah satu kontak di contact book.
type ContactRow struct {
	ID        string
	Name      string
	Address   string
	Network   string
	Notes     string
	CreatedAt string
	UpdatedAt string
}

// ListContacts mengambil seluruh kontak, diurutkan berdasarkan nama.
func (s *Store) ListContacts(ctx context.Context) ([]ContactRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, address, network, COALESCE(notes, ''), created_at, updated_at
		FROM contacts
		ORDER BY name COLLATE NOCASE
	`)
	if err != nil {
		return nil, fmt.Errorf("store: list contacts: %w", err)
	}
	defer rows.Close()

	var contacts []ContactRow
	for rows.Next() {
		var c ContactRow
		if err := rows.Scan(&c.ID, &c.Name, &c.Address, &c.Network, &c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: scan contact: %w", err)
		}
		contacts = append(contacts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: rows contact: %w", err)
	}
	return contacts, nil
}

package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Backup membuat salinan database SQLite di data/backups.
// Menggunakan VACUUM INTO supaya hasil backup konsisten.
func (s *Store) Backup() (string, error) {
	backupDir := filepath.Join(filepath.Dir(s.path), "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", fmt.Errorf("store: backup dir: %w", err)
	}

	stamp := time.Now().UTC().Format("20060102-150405")
	dst := filepath.Join(backupDir, stamp+"-"+filepath.Base(s.path))
	tmp := dst + ".tmp"
	_ = os.Remove(tmp)

	if _, err := s.db.Exec(`VACUUM INTO ?`, tmp); err != nil {
		return "", fmt.Errorf("store: backup: %w", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("store: backup rename: %w", err)
	}
	return dst, nil
}

// Restore menyalin seluruh isi database backup ke database aktif.
func (s *Store) Restore(backupPath string) error {
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("store: restore %s: %w", backupPath, err)
	}

	src, err := sql.Open(driverName, backupPath)
	if err != nil {
		return fmt.Errorf("store: restore open: %w", err)
	}
	defer src.Close()

	rows, err := src.Query(`SELECT name, data, updated_at FROM documents`)
	if err != nil {
		return fmt.Errorf("store: restore read: %w", err)
	}
	defer rows.Close()

	type doc struct {
		name      string
		data      string
		updatedAt string
	}

	var docs []doc
	for rows.Next() {
		var d doc
		if err := rows.Scan(&d.name, &d.data, &d.updatedAt); err != nil {
			return fmt.Errorf("store: restore scan: %w", err)
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: restore rows: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("store: restore close: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: restore begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM documents`); err != nil {
		return fmt.Errorf("store: restore clear: %w", err)
	}
	for _, d := range docs {
		if _, err := tx.Exec(
			`INSERT INTO documents (name, data, updated_at) VALUES (?, ?, ?)`,
			d.name, d.data, d.updatedAt,
		); err != nil {
			return fmt.Errorf("store: restore insert: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: restore commit: %w", err)
	}
	return nil
}

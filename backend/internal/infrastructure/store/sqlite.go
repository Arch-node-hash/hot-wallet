package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const driverName = "sqlite3"

// Store menyimpan dokumen JSON di dalam SQLite.
// Satu koleksi = satu baris di tabel documents.
type Store struct {
	db   *sql.DB
	path string
}

// Open membuka (atau membuat) database SQLite di path yang diberikan.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("store: create dir: %w", err)
	}

	db, err := sql.Open(driverName, path)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}

	// SQLite paling aman dengan satu koneksi untuk penulisan.
	db.SetMaxOpenConns(1)

	s := &Store{db: db, path: path}
	if err := s.init(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) init() error {
	statements := []string{
		`PRAGMA journal_mode = WAL`,
		`PRAGMA busy_timeout = 5000`,
		`CREATE TABLE IF NOT EXISTS documents (
			name       TEXT PRIMARY KEY,
			data       TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("store: init: %w", err)
		}
	}
	return nil
}

// Path mengembalikan lokasi file database.
func (s *Store) Path() string {
	return s.path
}

// Ping memastikan koneksi database masih hidup.
func (s *Store) Ping() error {
	return s.db.Ping()
}

// Close menutup koneksi database.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB mengembalikan koneksi database yang mendasar
func (s *Store) DB() *sql.DB {
	return s.db
}

// Load membaca satu koleksi ke dst. Jika koleksi belum ada, dst dibiarkan
// kosong dan tidak dianggap error.
func (s *Store) Load(name string, dst any) error {
	var data string
	err := s.db.QueryRow(`SELECT data FROM documents WHERE name = ?`, name).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("store: load %s: %w", name, err)
	}
	if err := json.Unmarshal([]byte(data), dst); err != nil {
		return fmt.Errorf("store: decode %s: %w", name, err)
	}
	return nil
}

// Save menulis satu koleksi (upsert) ke database.
func (s *Store) Save(name string, src any) error {
	data, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("store: encode %s: %w", name, err)
	}

	_, err = s.db.Exec(`
		INSERT INTO documents (name, data, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			data = excluded.data,
			updated_at = excluded.updated_at
	`, name, string(data), time.Now().UTC().Format(time.RFC3339))

	if err != nil {
		return fmt.Errorf("store: save %s: %w", name, err)
	}
	return nil
}

// Delete menghapus satu koleksi.
func (s *Store) Delete(name string) error {
	if _, err := s.db.Exec(`DELETE FROM documents WHERE name = ?`, name); err != nil {
		return fmt.Errorf("store: delete %s: %w", name, err)
	}
	return nil
}

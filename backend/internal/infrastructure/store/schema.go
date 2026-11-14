package store

import (
	"database/sql"
	"fmt"
)

// InitSchema menginisialisasi schema database untuk semua modul
func InitSchema(db *sql.DB) error {
	// Tabel untuk Wallet
	if err := createWalletTables(db); err != nil {
		return fmt.Errorf("wallet schema: %w", err)
	}

	// Tabel untuk Portfolio
	if err := createPortfolioTables(db); err != nil {
		return fmt.Errorf("portfolio schema: %w", err)
	}

	// Tabel untuk Transaction
	if err := createTransactionTables(db); err != nil {
		return fmt.Errorf("transaction schema: %w", err)
	}

	// Tabel untuk Contact
	if err := createContactTables(db); err != nil {
		return fmt.Errorf("contact schema: %w", err)
	}

	// Tabel untuk Setting
	if err := createSettingTables(db); err != nil {
		return fmt.Errorf("setting schema: %w", err)
	}

	// Tabel token (metadata untuk pembacaan saldo ERC-20/SPL)
	if err := createTokenTables(db); err != nil {
		return fmt.Errorf("token schema: %w", err)
	}

	return nil
}

// createWalletTables membuat tabel untuk modul Wallet
func createWalletTables(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS wallets (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			address TEXT NOT NULL,
			network TEXT NOT NULL,
			derivation_path TEXT NOT NULL,
			derivation_index INTEGER DEFAULT 0,
			balance TEXT DEFAULT '0',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS wallet_balances (
			wallet_id TEXT NOT NULL,
			network TEXT NOT NULL,
			token_address TEXT,
			token_symbol TEXT NOT NULL,
			balance TEXT NOT NULL,
			value_usd TEXT DEFAULT '0',
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (wallet_id, network, token_address),
			FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS network_summaries (
			wallet_id TEXT NOT NULL,
			network TEXT NOT NULL,
			total_balance TEXT DEFAULT '0',
			total_value_usd TEXT DEFAULT '0',
			token_count INTEGER DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (wallet_id, network),
			FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE CASCADE
		)`,
	}

	return executeStatements(db, statements)
}

// createPortfolioTables membuat tabel untuk modul Portfolio
func createPortfolioTables(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS portfolio_summary (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			total_assets TEXT DEFAULT '0',
			total_value_usd TEXT DEFAULT '0',
			network_count INTEGER DEFAULT 0,
			token_count INTEGER DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS portfolio_tokens (
			wallet_id TEXT NOT NULL,
			network TEXT NOT NULL,
			token_address TEXT,
			token_symbol TEXT NOT NULL,
			token_name TEXT,
			balance TEXT NOT NULL,
			price_usd TEXT DEFAULT '0',
			value_usd TEXT DEFAULT '0',
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (wallet_id, network, token_address),
			FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS network_assets (
			wallet_id TEXT NOT NULL,
			network TEXT NOT NULL,
			total_tokens INTEGER DEFAULT 0,
			total_value_usd TEXT DEFAULT '0',
			percentage REAL DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (wallet_id, network),
			FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE CASCADE
		)`,
	}

	return executeStatements(db, statements)
}

// createTransactionTables membuat tabel untuk modul Transaction
func createTransactionTables(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,
			wallet_id TEXT NOT NULL,
			network TEXT NOT NULL,
			tx_hash TEXT NOT NULL,
			from_address TEXT NOT NULL,
			to_address TEXT NOT NULL,
			amount TEXT NOT NULL,
			token_symbol TEXT,
			token_address TEXT,
			fee TEXT,
			status TEXT NOT NULL,
			block_number INTEGER,
			timestamp TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE CASCADE
		)`,

		`CREATE INDEX IF NOT EXISTS idx_transactions_wallet ON transactions(wallet_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_timestamp ON transactions(timestamp)`,
	}

	return executeStatements(db, statements)
}

// createContactTables membuat tabel untuk modul Contact
func createContactTables(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS contacts (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			address TEXT NOT NULL,
			network TEXT NOT NULL,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE INDEX IF NOT EXISTS idx_contacts_name ON contacts(name)`,
		`CREATE INDEX IF NOT EXISTS idx_contacts_network ON contacts(network)`,
	}

	return executeStatements(db, statements)
}

// createSettingTables membuat tabel untuk modul Setting
func createSettingTables(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS rpc_configs (
			network TEXT PRIMARY KEY,
			rpc_url TEXT NOT NULL,
			wss_url TEXT,
			chain_id INTEGER,
			explorer_url TEXT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS security_settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pin_hash TEXT,
			biometric_enabled INTEGER DEFAULT 0,
			auto_lock_minutes INTEGER DEFAULT 5,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS backup_history (
			id TEXT PRIMARY KEY,
			backup_type TEXT NOT NULL,
			backup_date TIMESTAMP NOT NULL,
			file_path TEXT,
			encrypted INTEGER DEFAULT 1,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	return executeStatements(db, statements)
}

// createTokenTables membuat tabel metadata token untuk pembacaan saldo.
func createTokenTables(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS tokens (
			network TEXT NOT NULL,
			address TEXT NOT NULL,
			symbol TEXT NOT NULL,
			name TEXT,
			decimals INTEGER NOT NULL,
			PRIMARY KEY (network, address)
		)`,

		// Harga token per jaringan. Diisi dari luar (mis. watcher harga) dan
		// dibaca saat menghitung nilai USD portfolio. Stablecoin tetap punya
		// fallback $1 walau tabel ini kosong.
		`CREATE TABLE IF NOT EXISTS token_prices (
			network TEXT NOT NULL,
			token_address TEXT NOT NULL DEFAULT '',
			symbol TEXT NOT NULL,
			price_usd TEXT NOT NULL DEFAULT '0',
			change_24h REAL NOT NULL DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (network, token_address)
		)`,
	}

	if err := executeStatements(db, statements); err != nil {
		return err
	}

	// Migrasi untuk database lama: kolom wss_url belum ada di rpc_configs,
	// kolom derivation_index belum ada di wallets.
	if err := ensureColumn(db, "rpc_configs", "wss_url", "TEXT"); err != nil {
		return err
	}
	if err := ensureColumn(db, "wallets", "derivation_index", "INTEGER DEFAULT 0"); err != nil {
		return err
	}
	return nil
}

// ensureColumn menambahkan kolom bila belum ada (SQLite tidak punya
// ADD COLUMN IF NOT EXISTS).
func ensureColumn(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid          int
			name         string
			columnType   string
			notNull      int
			defaultValue any
			primaryKey   int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}

// executeStatements mengeksekusi kumpulan SQL statements
func executeStatements(db *sql.DB, statements []string) error {
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("execute %s: %w", stmt, err)
		}
	}
	return nil
}

// InsertDefaultSettings menyisipkan pengaturan default
func InsertDefaultSettings(db *sql.DB) error {
	defaultSettings := []struct {
		key   string
		value string
	}{
		{"app_name", "Hot Wallet"},
		{"currency", "USD"},
		{"language", "id"},
		{"theme", "light"},
		{"default_network", "Ethereum Mainnet"},
	}

	for _, setting := range defaultSettings {
		_, err := db.Exec(`
			INSERT OR IGNORE INTO settings (key, value) 
			VALUES (?, ?)
		`, setting.key, setting.value)
		if err != nil {
			return fmt.Errorf("insert setting %s: %w", setting.key, err)
		}
	}

	return nil
}

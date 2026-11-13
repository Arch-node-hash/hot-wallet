# HotWallet Backend Skeleton (Fiber v3 + SQLite)

Backend API minimal memakai [Fiber v3](https://gofiber.io) dan **SQLite**
(`github.com/mattn/go-sqlite3`). Tidak ada frontend atau penyajian halaman
statis — server ini hanya menyediakan endpoint JSON.

Semua handler dan service domain masih berupa **placeholder**. Struktur folder
tetap dipertahankan supaya mudah diisi logika nyata satu per satu.

## Prasyarat

- Go 1.26+
- GCC / CGO aktif (dibutuhkan driver SQLite)

Cek dengan:

```bash
go env CGO_ENABLED   # harus 1
```

## Menjalankan

```bash
cd backend
cp .env.example .env   # opsional
go run ./cmd/api
```

Server berjalan di <http://localhost:5009>. Database SQLite dibuat otomatis
di `data/hotwallet.db`.

```bash
curl http://localhost:5009/
curl http://localhost:5009/api/v1/health
```

## Konfigurasi

Semua lewat environment variable (atau file `backend/.env`):

| Variable   | Default            | Keterangan                          |
|------------|--------------------|-------------------------------------|
| `PORT`     | `5009`             | Port server Go                      |
| `DATA_DIR` | `./data`           | Folder avatar & backup              |
| `DB_PATH`  | `./data/hotwallet.db`  | Lokasi file database SQLite         |
| `ETHEREUM_WSS` dll | *(kosong)*  | Endpoint WebSocket per jaringan (`ETHEREUM_WSS`, `BSC_WSS`, `ARBITRUM_WSS`, `ROBINHOOD_WSS`, `SOLANA_WSS`) |
| `WS_WATCHER_ENABLED` | `true`    | Nyalakan watcher saldo real-time    |
| `WS_DEBOUNCE` | `4s`             | Jarak minimum refresh per blok (EVM) |
| `WS_REFRESH` | `30s`             | Refresh periodik cadangan           |
| `WALLET_MNEMONIC` | *(dibuat otomatis)* | Seed phrase BIP39 12 kata untuk identitas wallet |

Endpoint WebSocket **tidak di-hardcode** di source. Saat pertama dijalankan,
nilai dari env di-seed ke tabel `rpc_configs.wss_url`; setelah itu nilainya
dibaca dari database dan bisa diubah lewat halaman Setting/RPC tanpa rebuild.
Perubahan endpoint, wallet, atau daftar token membuat watcher terkait
restart otomatis dalam ±15 detik.

## Watcher saldo real-time

Watcher berjalan di background (satu koneksi WebSocket per jaringan yang punya
wallet) dan menyimpan setiap hasil pembacaan ke SQLite:

- **EVM (Ethereum, BNB Chain, Arbitrum, Robinhood)**: `eth_subscribe` `newHeads`
  sebagai trigger, lalu batch `eth_getBalance` + `eth_call balanceOf` pada
  koneksi yang sama. Blok cepat di-debounce.
- **Solana**: `accountSubscribe` untuk wallet dan Associated Token Account,
  ditambah `getMultipleAccounts` (HTTP, karena endpoint publik gratis tidak
  mengizinkan request JSON-RPC lewat WS) untuk saldo awal dan refresh.

Hasil ditulis ke `wallet_balances`, `wallets.balance` (native), dan
`network_summaries`. Metadata token yang dipantau ada di tabel `tokens`.

## Seed phrase & identitas wallet

Identitas wallet memakai **seed phrase BIP39** (12 kata). Saat server pertama
kali dijalankan dan `WALLET_MNEMONIC` masih kosong, server membuat seed acak
lalu menuliskannya ke `backend/.env`. File `.env` tidak ikut git — **backup
file itu**, karena seed yang sama diperlukan untuk memulihkan semua alamat.

Dari satu seed, alamat diturunkan secara HD:

| Jaringan | Path derivasi |
|---|---|
| EVM (Ethereum, BNB Chain, Arbitrum, Robinhood) | `m/44'/60'/0'/0/{index}` |
| Solana | `m/44'/501'/{index}'/0'` |

Tabel `wallets` hanya menyimpan `address`, `network`, dan `derivation_path` —
seed phrase tidak dimasukkan ke database. Helper-nya ada di
`internal/blockchain/wallet`:

```go
mnemonic, _ := wallet.NewMnemonic(12)          // buat seed phrase
ok := wallet.ValidateMnemonic(mnemonic)        // cek checksum BIP39

evm, _ := wallet.DeriveEVM(mnemonic, "", 0)    // alamat EVM + private key
sol, _ := wallet.DeriveSolana(mnemonic, "", 0) // alamat Solana + ed25519 key
```

Alamat EVM memakai checksum EIP-55; alamat Solana memakai base58 dari public
key ed25519. Private key hasil derivasi **tidak** ditulis ke database.

## Struktur

```
backend/
├── cmd/api/main.go          # Entry point + seed RPC/token + start watcher
├── config/config.go         # Konfigurasi via env/.env
├── data/
│   ├── avatars/
│   └── hotwallet.db
├── internal/
│   ├── blockchain/
│   │   ├── evm/             # Client + watcher WS EVM
│   │   ├── solana/          # Client + watcher WS Solana + ATA token
│   │   ├── network/         # Registry + nama jaringan kanonik
│   │   ├── tokenmeta/       # Metadata token bawaan
│   │   ├── wallet/          # HD wallet/keystore
│   │   ├── watcher/         # Manager watcher per jaringan
│   │   └── wsrpc/           # Client JSON-RPC over WebSocket
│   ├── infrastructure/
│   │   ├── router/router.go # Semua route API
│   │   ├── store/           # SQLite, schema, saldo, rpc_configs
│   │   ├── security/        # Enkripsi/kunci
│   │   └── health/
│   ├── modules/             # Satu paket per fitur (handler + service + model)
│   │   ├── wallet/          # handler.go, service.go, model.go
│   │   ├── portfolio/       # handler.go, service.go, model.go
│   │   ├── transaction/     # handler.go, service.go, model.go
│   │   ├── contact/         # handler.go, service.go, model.go
│   │   └── setting/         # handler.go, service.go, model.go
│   └── shared/              # Model, interface, error
├── pkg/
│   ├── auth/
│   ├── crud/
│   ├── logger/
│   ├── response/
│   └── validator/
├── go.mod
├── go.sum
├── .env.example
├── .gitignore
├── README.md
└── CONTRIBUTING.md
```

Handler fitur di `internal/modules/*` masih skeleton; routing saat ini memakai
placeholder inline di `router.go`. Watcher saldo dan penyimpanan ke database
sudah berjalan.

Alur request:

```
Request → Fiber middleware → router → handler → service → crud.Repo → store → SQLite
```

## Skema SQLite

Database dibuat otomatis di `data/hotwallet.db`. Tabel yang dipakai:

| Tabel | Keterangan |
|---|---|
| `wallets` | Wallet + saldo native terakhir |
| `wallet_balances` | Saldo native & token per wallet/jaringan (hasil watcher) |
| `network_summaries` | Ringkasan saldo per wallet/jaringan |
| `tokens` | Metadata token yang dipantau (ERC-20/SPL) |
| `rpc_configs` | Endpoint RPC + WSS per jaringan |
| `transactions` | Riwayat transaksi |
| `contacts` | Kontak alamat wallet |
| `settings` | Pengaturan aplikasi |
| `security_settings` | PIN/biometric/auto-lock |
| `backup_history` | Riwayat backup |
| `portfolio_summary`, `portfolio_tokens`, `network_assets` | Data portfolio |
| `documents` | Penyimpanan JSON generik (dipakai `pkg/crud`) |

## Rute

Semua endpoint fitur ada di bawah `/api/v1`. **Handler-nya masih placeholder
`501 Not Implemented`**, kecuali `/`, `/api/v1/health`, dan watcher saldo yang
berjalan di background.

| Method | Path | Fitur |
|---|---|---|
| GET | `/` | Info server |
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/wallets` | Daftar wallet |
| POST | `/api/v1/wallets` | Tambah wallet |
| GET | `/api/v1/wallets/{walletID}` | Detail wallet (termasuk alamat) |
| GET | `/api/v1/wallets/{walletID}/balance` | Saldo wallet + ringkasan per jaringan |
| POST | `/api/v1/wallets/{walletID}/send` | Kirim aset |
| GET | `/api/v1/wallets/{walletID}/receive` | Alamat/QR receive |
| DELETE | `/api/v1/wallets/{walletID}` | Hapus wallet |
| GET | `/api/v1/portfolio/summary` | Total aset |
| GET | `/api/v1/portfolio/tokens` | Daftar token + saldo |
| GET | `/api/v1/portfolio/networks` | Rincian aset per jaringan |
| GET | `/api/v1/transactions` | Riwayat transaksi |
| GET | `/api/v1/transactions/{txHash}` | Detail transaksi |
| GET | `/api/v1/transactions/{txHash}/status` | Status transaksi |
| GET | `/api/v1/contacts` | Daftar kontak |
| POST | `/api/v1/contacts` | Simpan kontak |
| GET | `/api/v1/contacts/{contactID}` | Detail kontak |
| PUT | `/api/v1/contacts/{contactID}` | Edit kontak |
| DELETE | `/api/v1/contacts/{contactID}` | Hapus kontak |
| POST | `/api/v1/settings/wallets/create` | Buat wallet baru |
| POST | `/api/v1/settings/wallets/import` | Impor wallet |
| GET | `/api/v1/settings/rpc` | Daftar RPC/WSS |
| PUT | `/api/v1/settings/rpc` | Ubah RPC/WSS |
| GET | `/api/v1/settings/security` | Baca pengaturan keamanan |
| PUT | `/api/v1/settings/security` | Ubah pengaturan keamanan |
| GET | `/api/v1/settings/backup` | Info backup |
| POST | `/api/v1/settings/backup` | Buat backup |
| GET | `/avatars/*` | File avatar dari `data/avatars` |

Tidak ada endpoint dashboard, analytics, tasks, projects, team, reminders,
calendar, time-entries, atau auth. Path yang tidak terdaftar akan mendapat
`404 {"error":"not found"}`.

## Mengisi logika

1. Buka `internal/modules/<domain>/handler.go` dan ganti isi method.
2. Tambahkan dependency/implementasi di `internal/modules/<domain>/service.go`.
3. Sesuaikan model di `internal/modules/<domain>/model.go` bila perlu.
4. Daftarkan dependency baru di `internal/infrastructure/router/router.go`.

Panduan lengkap ada di [`CONTRIBUTING.md`](./CONTRIBUTING.md).


## 🏗️ Struktur Aplikasi

### 5 Halaman Utama:

1. **Wallet** (`/api/v1/wallets`)
   - Daftar wallet
   - Saldo per jaringan
   - Alamat wallet
   - Send & Receive

2. **Portfolio** (`/api/v1/portfolio`)
   - Total aset
   - Daftar token dan saldo
   - Rincian aset per jaringan

3. **Transaction** (`/api/v1/transactions`)
   - Riwayat transaksi
   - Status transaksi

4. **Contact** (`/api/v1/contacts`)
   - Simpan alamat wallet
   - Edit dan hapus kontak

5. **Setting** (`/api/v1/settings`)
   - Create & Import Wallet
   - RPC jaringan
   - Keamanan
   - Backup

### 🏦 Jaringan yang Didukung:
- **EVM**: Ethereum Mainnet, BNB Chain, Arbitrum One, Robinhood Chain
- **Non-EVM**: Solana Mainnet
- **Devnet**: Solana Devnet (opsional, endpoint dari konfigurasi)

## 📁 Struktur Folder Backend

```
backend/
├── cmd/api/main.go          # Entry point
├── config/                  # Configuration
├── internal/
│   ├── blockchain/         # Blockchain clients
│   │   ├── evm/           # EVM networks
│   │   ├── solana/        # Solana network
│   │   └── network/       # Network registry
│   ├── infrastructure/
│   │   ├── router/        # API routes
│   │   ├── store/         # Database layer
│   │   └── security/      # Key management
│   ├── modules/           # Business logic per halaman
│   │   ├── wallet/        # handler.go, service.go, model.go
│   │   ├── portfolio/     # handler.go, service.go, model.go
│   │   ├── transaction/   # handler.go, service.go, model.go
│   │   ├── contact/       # handler.go, service.go, model.go
│   │   └── setting/       # handler.go, service.go, model.go
│   └── shared/
│       ├── models/        # Data models
│       └── interfaces/    # Service interfaces
└── data/                  # Database files
```

## 🚀 Menjalankan Aplikasi

1. **Copy environment file:**
   ```bash
   cp .env.example .env
   ```

2. **Update .env dengan konfigurasi Anda**

3. **Install dependencies:**
   ```bash
   go mod download
   ```

4. **Run server:**
   ```bash
   go run ./cmd/api
   ```

5. **API akan berjalan di:** `http://localhost:5009`

## 📚 API Documentation

### Base URL: `http://localhost:5009/api/v1`

#### Wallet Endpoints:
- `GET /wallets` - List all wallets
- `POST /wallets` - Tambah wallet
- `GET /wallets/{id}` - Detail wallet (termasuk alamat)
- `GET /wallets/{id}/balance` - Saldo wallet + ringkasan per jaringan
- `POST /wallets/{id}/send` - Send transaction
- `GET /wallets/{id}/receive` - Alamat/QR receive
- `DELETE /wallets/{id}` - Hapus wallet

#### Portfolio Endpoints:
- `GET /portfolio/summary` - Total aset
- `GET /portfolio/tokens` - Daftar token + saldo
- `GET /portfolio/networks` - Rincian aset per jaringan

#### Transaction Endpoints:
- `GET /transactions` - Riwayat transaksi
- `GET /transactions/{txHash}` - Detail transaksi
- `GET /transactions/{txHash}/status` - Status transaksi

#### Contact Endpoints:
- `GET /contacts` - Daftar kontak
- `POST /contacts` - Simpan kontak
- `GET /contacts/{id}` - Detail kontak
- `PUT /contacts/{id}` - Edit kontak
- `DELETE /contacts/{id}` - Hapus kontak

#### Setting Endpoints:
- `POST /settings/wallets/create` - Buat wallet baru
- `POST /settings/wallets/import` - Impor wallet
- `GET /settings/rpc` - Daftar RPC/WSS
- `PUT /settings/rpc` - Ubah RPC/WSS
- `GET /settings/security` - Baca pengaturan keamanan
- `PUT /settings/security` - Ubah pengaturan keamanan
- `GET /settings/backup` - Info backup
- `POST /settings/backup` - Buat backup

## 🔐 Security Features

- Key encryption menggunakan AES-GCM
- PIN protection dengan hashing
- Biometric authentication support
- Auto-lock untuk session timeout
- Encrypted database backup

## 🔧 Development

Struktur sudah di-setup dengan:
- Database schema untuk semua 5 halaman
- API routes dengan placeholder handlers
- Service layer interfaces
- Response models
- Network registry dengan default networks

**Selanjutnya:**
Implementasikan logic business di masing-masing modul:
1. `internal/modules/wallet/`
2. `internal/modules/portfolio/`
3. `internal/modules/transaction/`
4. `internal/modules/contact/`
5. `internal/modules/setting/`

Database sudah siap dengan semua tabel yang diperlukan.
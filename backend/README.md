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
curl http://localhost:5009/api/health
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

## Struktur

```
backend/
├── cmd/
│   └── api/
│       └── main.go
├── config/
│   └── config.go
├── data/
│   ├── avatars/
│   └── hotwallet.db
├── internal/
│   ├── modules/
│   │   ├── activity/
│   │   │   ├── handler.go
│   │   │   ├── model.go
│   │   │   └── service.go
│   │   ├── contact/
│   │   │   ├── handler.go
│   │   │   ├── model.go
│   │   │   └── service.go
│   │   ├── portfolio/
│   │   │   ├── handler.go
│   │   │   ├── model.go
│   │   │   └── service.go
│   │   ├── setting/
│   │   │   ├── handler.go
│   │   │   ├── model.go
│   │   │   └── service.go
│   │   ├── transaction/
│   │   │   ├── handler.go
│   │   │   ├── model.go
│   │   │   └── service.go
│   │   └── wallet/
│   │       ├── handler.go
│   │       ├── model.go
│   │       └── service.go
│   ├── blockchain/
│   │   ├── wallet/
│   │   │   ├── master.go
│   │   │   ├── derivation.go
│   │   │   ├── address.go
│   │   │   └── keystore.go
│   │   ├── evm/
│   │   │   ├── client.go
│   │   │   ├── balance.go
│   │   │   ├── transfer.go
│   │   │   ├── token.go
│   │   │   ├── gas.go
│   │   │   └── transaction.go
│   │   ├── solana/
│   │   │   ├── client.go
│   │   │   ├── balance.go
│   │   │   ├── transfer.go
│   │   │   ├── token.go
│   │   │   └── transaction.go
│   │   └── network/
│   │       ├── registry.go
│   │       ├── network.go
│   │       └── rpc.go
│   ├── infrastructure/
│   │   ├── router/
│   │   │   └── router.go
│   │   ├── store/
│   │   │   ├── sqlite.go
│   │   │   ├── backup.go
│   │   │   └── migrations/
│   │   ├── security/
│   │   │   ├── encryption.go
│   │   │   └── keymanager.go
│   │   └── health/
│   │       └── handler.go
│   └── shared/
│       ├── errors/
│       │   └── errors.go
│       └── types/
│           └── types.go
├── pkg/
│   ├── auth/
│   ├── crud/
│   ├── logger/
│   ├── response/
│   └── validator/
├── go.mod
├── go.sum
├── .env
├── .env.example
├── .gitignore
├── README.md
└── CONTRIBUTING.md
```

Package `internal/blockchain/*`, `internal/infrastructure/security`, dan
`internal/shared/*` masih berupa skeleton (kontrak + TODO), belum terhubung ke
route mana pun.

Alur request:

```
Request → Fiber middleware → router → handler → service → crud.Repo → store → SQLite
```

## Skema SQLite

Tabel `documents`:

| Kolom        | Tipe | Keterangan                        |
|--------------|------|-----------------------------------|
| `name`       | TEXT | Nama koleksi (primary key)        |
| `data`       | TEXT | Dokumen JSON koleksi              |
| `updated_at` | TEXT | Waktu update terakhir (RFC3339)   |

Contoh nama koleksi: `tasks`, `projects`, `team`, `events`, `reminders`,
`time_entries`, `settings`, `analytics`, `dashboard`, `users`.

## Rute

| Method | Path | Keterangan |
|---|---|---|
| GET | `/` | Info server |
| GET | `/api/health` | Status API + database |
| GET | `/api/auth/me` | Placeholder user |
| POST | `/api/auth/logout` | Placeholder logout |
| GET | `/api/dashboard` | Placeholder dashboard |
| GET | `/api/analytics` | Placeholder analytics |
| GET/PUT | `/api/settings` | Placeholder settings |
| POST | `/api/settings/avatar` | Placeholder upload avatar |
| GET/POST | `/api/tasks` | Placeholder list/tambah task |
| GET/PUT/PATCH/DELETE | `/api/tasks/{id}` | Placeholder detail/ubah/hapus |
| GET/POST | `/api/projects` | Placeholder list/tambah project |
| GET/PUT/PATCH/DELETE | `/api/projects/{id}` | Placeholder detail/ubah/hapus |
| GET/POST | `/api/team` | Placeholder list/tambah anggota |
| GET/PUT/PATCH/DELETE | `/api/team/{id}` | Placeholder detail/ubah/hapus |
| GET/POST | `/api/reminders` | Placeholder list/tambah reminder |
| GET/PUT/PATCH/DELETE | `/api/reminders/{id}` | Placeholder detail/ubah/hapus |
| GET/POST | `/api/calendar/events` | Placeholder list/tambah event |
| GET/PUT/PATCH/DELETE | `/api/calendar/events/{id}` | Placeholder detail/ubah/hapus |
| GET/POST | `/api/time-entries` | Placeholder list/tambah time entry |
| GET/PUT/PATCH/DELETE | `/api/time-entries/{id}` | Placeholder detail/ubah/hapus |
| GET | `/avatars/*` | File avatar dari `data/avatars` |

Contoh response placeholder:

```json
{
  "resource": "tasks",
  "message": "placeholder"
}
```

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
- **EVM Networks**: Ethereum, BNB Chain, Arbitrum, Polygon
- **Non-EVM**: Solana
- **Testnets**: Sepolia, BSC Testnet, Arbitrum Sepolia, Solana Devnet

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
│   │   ├── wallet/
│   │   ├── portfolio/
│   │   ├── transaction/
│   │   ├── contact/
│   │   └── setting/
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
   go run cmd/api/main.go
   ```

5. **API akan berjalan di:** `http://localhost:5009`

## 📚 API Documentation

### Base URL: `http://localhost:5009/api/v1`

#### Wallet Endpoints:
- `GET /wallets` - List all wallets
- `POST /wallets` - Create new wallet
- `GET /wallets/{id}` - Get wallet detail
- `GET /wallets/{id}/balance` - Get wallet balance
- `POST /wallets/{id}/send` - Send transaction
- `GET /wallets/{id}/receive` - Get receive address

#### Portfolio Endpoints:
- `GET /portfolio/summary` - Portfolio summary
- `GET /portfolio/tokens` - List all tokens
- `GET /portfolio/networks` - Network breakdown

#### Transaction Endpoints:
- `GET /transactions` - Transaction history
- `GET /transactions/{txHash}` - Transaction detail
- `GET /transactions/{txHash}/status` - Transaction status

#### Contact Endpoints:
- `GET /contacts` - List contacts
- `POST /contacts` - Create contact
- `PUT /contacts/{id}` - Update contact
- `DELETE /contacts/{id}` - Delete contact

#### Setting Endpoints:
- `POST /settings/wallets/create` - Create wallet
- `POST /settings/wallets/import` - Import wallet
- `GET /settings/rpc` - List RPC configs
- `PUT /settings/rpc` - Update RPC config
- `GET /settings/security` - Get security settings
- `PUT /settings/security` - Update security settings
- `POST /settings/backup` - Create backup

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
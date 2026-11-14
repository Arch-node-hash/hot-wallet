# HotWallet Backend Skeleton (Fiber v3 + SQLite)

Backend API minimal memakai [Fiber v3](https://gofiber.io) dan **SQLite**
(`github.com/mattn/go-sqlite3`). Tidak ada frontend atau penyajian halaman
statis — server ini hanya menyediakan endpoint JSON.

Modul **Wallet** sudah berisi logika nyata (overview, alamat, rotate,
refresh, estimasi & kirim transaksi), termasuk Contact Book (baca daftar) dan
pengaturan keamanan (PIN). Modul Portfolio, Transaction, dan sisanya masih
**placeholder** `501 Not Implemented`; strukturnya dipertahankan supaya mudah
diisi logika nyata satu per satu.

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
| `ETH_PRIVATE_KEY` | *(wajib diisi)* | Private key Ethereum (64 hex characters dengan 0x prefix) |
| `SOLANA_PRIVATE_KEY` | *(wajib diisi)* | Private key Solana (base58 encoded) |

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

## Private Keys untuk Wallet

Wallet menggunakan **private key langsung** tanpa seed phrase:

1. **Ethereum & EVM chains**: `ETH_PRIVATE_KEY` (64 hex characters, `0x` opsional)
2. **Solana**: `SOLANA_PRIVATE_KEY` (base58 / hex / JSON array)

Satu `ETH_PRIVATE_KEY` dipakai untuk **semua jaringan EVM** (Ethereum, BNB
Chain, Arbitrum, Robinhood). Alamatnya sama, tetapi tabel `wallets` tetap
menyimpan **satu baris per jaringan** (ID `eth-`, `bsc-`, `arb-`, `rob-`).
Solana memakai kurva ed25519 yang berbeda, jadi wajib punya private key
sendiri (`sol-`).

Contoh di `.env`:
```
ETH_PRIVATE_KEY=0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
SOLANA_PRIVATE_KEY=<base58 seed 32 byte atau keypair 64 byte>
```

Format `SOLANA_PRIVATE_KEY` yang diterima:
- base58 32 byte (seed) atau 64 byte (keypair hasil export Phantom/solana-keygen)
- hex 64 karakter (seed) atau 128 karakter (keypair)
- JSON array byte, format file `solana-keygen` (`[12,34,...]`)

Private keys **tidak disimpan di database** - hanya di environment variable.
Saat startup, wallet yang belum ada dibuat otomatis dari key tersebut (satu
wallet per jaringan). File `.env` tidak ikut git — **backup file itu dengan aman**.

Kalau private key diganti lalu server dijalankan ulang, alamat wallet ikut
diperbarui **otomatis** (tanpa menambah wallet baru), selama jaringan tersebut
hanya punya satu wallet. Deteksinya memakai alamat hasil derivasi key: alamat
berbeda = key berbeda. Saldo lama dihapus dan akan diisi ulang oleh watcher.
Kalau satu jaringan punya lebih dari satu wallet, auto-sync dilewati dan
alamat harus diperbarui manual lewat
`PUT /api/v1/wallets/{walletID}/address`. ID wallet sengaja tidak berubah
agar referensi ke wallet tersebut tetap stabil.

### Derivasi alamat untuk "Perbaharui Address"

Rotate alamat memakai derivasi HD deterministik dari private key env, **tanpa
menyimpan kunci tambahan**:

- **Index 0** selalu alamat langsung dari private key (kompatibel dengan wallet
  yang sudah ada).
- **Index > 0** diturunkan dari seed `SHA256("HotWallet/address/v1/" || key)`
  lewat BIP32 (EVM) atau SLIP-0010 (Solana).
- Index tersimpan di `wallets.derivation_index`. Saat startup, env-sync
  menurunkan ulang alamat pada index tersimpan, sehingga hasil rotate tetap
  sama setelah restart.

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
│   │   ├── wallet/          # handler.go, service.go, model.go, page.go
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

Handler fitur di `internal/modules/*` selain Wallet/Contact/Setting masih
skeleton. Watcher saldo, pembacaan saldo via RPC HTTP, dan penyimpanan ke
database sudah berjalan.

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
| `token_prices` | Harga token per jaringan (sumber nilai USD; diisi dari luar) |
| `rpc_configs` | Endpoint RPC + WSS per jaringan |
| `transactions` | Riwayat transaksi |
| `contacts` | Kontak alamat wallet |
| `settings` | Pengaturan aplikasi |
| `security_settings` | PIN/biometric/auto-lock |
| `backup_history` | Riwayat backup |
| `portfolio_summary`, `portfolio_tokens`, `network_assets` | Data portfolio |
| `documents` | Penyimpanan JSON generik (dipakai `pkg/crud`) |

## Rute

Semua endpoint fitur ada di bawah `/api/v1`. Endpoint **Wallet** (kecuali
`POST /wallets` create manual), `GET /contacts`, dan
`GET/PUT /settings/security` sudah berfungsi; sisanya masih placeholder
`501 Not Implemented`.

| Method | Path | Fitur | Status |
|---|---|---|---|
| GET | `/` | Info server | ✅ |
| GET | `/api/v1/health` | Health check | ✅ |
| GET | `/api/v1/wallets` | Daftar wallet (diperkaya harga + nilai USD) | ✅ |
| POST | `/api/v1/wallets` | Tambah wallet manual (import private key) | ✅ |
| GET | `/api/v1/wallets/overview` | Tab ALL: total USD, perubahan 24 jam, kartu jaringan | ✅ |
| GET | `/api/v1/wallets/addresses` | Modal alamat 2 kolom (EVM + Solana) | ✅ |
| GET | `/api/v1/wallets/networks` | Metadata tab jaringan | ✅ |
| POST | `/api/v1/wallets/refresh` | Baca ulang saldo semua wallet, kembalikan overview | ✅ |
| GET | `/api/v1/wallets/{walletID}` | Detail wallet + saldo per token | ✅ |
| GET | `/api/v1/wallets/{walletID}/balance` | Saldo wallet + ringkasan per jaringan | ✅ |
| POST | `/api/v1/wallets/{walletID}/refresh` | Baca ulang saldo satu wallet | ✅ |
| POST | `/api/v1/wallets/{walletID}/rotate` | Perbaharui Address (alamat baru) | ✅ |
| POST | `/api/v1/wallets/{walletID}/send/estimate` | Estimasi biaya kirim | ✅ |
| POST | `/api/v1/wallets/{walletID}/send` | Kirim aset (native/ERC-20/SPL) | ✅ |
| GET | `/api/v1/wallets/{walletID}/receive` | Alamat/QR receive | ✅ |
| PUT | `/api/v1/wallets/{walletID}/address` | Ganti alamat dari private key kiriman | ✅ |
| DELETE | `/api/v1/wallets/{walletID}` | Hapus wallet | ✅ |
| GET | `/api/v1/portfolio/summary` | Total aset | 501 |
| GET | `/api/v1/portfolio/tokens` | Daftar token + saldo | 501 |
| GET | `/api/v1/portfolio/networks` | Rincian aset per jaringan | 501 |
| GET | `/api/v1/transactions` | Riwayat transaksi | 501 |
| GET | `/api/v1/transactions/{txHash}` | Detail transaksi | 501 |
| GET | `/api/v1/transactions/{txHash}/status` | Status transaksi | 501 |
| GET | `/api/v1/contacts` | Daftar kontak | ✅ |
| POST | `/api/v1/contacts` | Simpan kontak | 501 |
| GET | `/api/v1/contacts/{contactID}` | Detail kontak | 501 |
| PUT | `/api/v1/contacts/{contactID}` | Edit kontak | 501 |
| DELETE | `/api/v1/contacts/{contactID}` | Hapus kontak | 501 |
| GET | `/api/v1/settings/rpc` | Daftar RPC/WSS | 501 |
| PUT | `/api/v1/settings/rpc` | Ubah RPC/WSS | 501 |
| GET | `/api/v1/settings/security` | Baca pengaturan keamanan | ✅ |
| PUT | `/api/v1/settings/security` | Ubah pengaturan keamanan | ✅ |
| GET | `/api/v1/settings/backup` | Info backup | 501 |
| POST | `/api/v1/settings/backup` | Buat backup | 501 |
| GET | `/avatars/*` | File avatar dari `data/avatars` | ✅ |

Tidak ada endpoint dashboard, analytics, tasks, projects, team, reminders,
calendar, time-entries, atau auth. Path yang tidak terdaftar akan mendapat
`404 {"error":"not found"}`.

### Envelope respons

Semua endpoint JSON memakai bentuk yang sama (`internal/shared/models/api.go`):

```json
{ "success": true, "message": "opsional", "data": { } }
{ "success": false, "error": "pesan kesalahan" }
```

## Kontrak API halaman Wallet

Halaman Wallet memakai metadata jaringan statis dari
`internal/blockchain/network/meta.go`. Urutan tab:
`ALL, ETH, BNB, ARB, ROBINHOOD, SOL`.

| network_key | network | short | token_standard | address_type |
|---|---|---|---|---|
| `ethereum` | Ethereum Mainnet | ETH | ERC-20 | EVM Universal Address |
| `bsc` | BNB Chain | BNB | BEP-20 | EVM Universal Address |
| `arbitrum` | Arbitrum One | ARB | Arbitrum | EVM Universal Address |
| `robinhood` | Robinhood Chain | ROBINHOOD | RH-20 | EVM Universal Address |
| `solana` | Solana Mainnet | SOL | SPL | Solana Mainnet Base58 Address |

Nilai USD dihitung saat baca dari tabel `token_prices`. Stablecoin
(USDT/USDC/DAI/BUSD/TUSD/FDUSD/USDe/USDD) selalu dihargai $1 bila belum ada
baris harga. Native token bernilai 0 sampai `token_prices` diisi.

### `GET /api/v1/wallets/overview` — tab ALL

```bash
curl http://localhost:5009/api/v1/wallets/overview
```

```json
{
  "success": true,
  "data": {
    "total_value_usd": "9000.00",
    "change_24h_usd": "100.00",
    "change_24h_percent": 1.11,
    "refreshed_at": "2026-01-01T00:00:00Z",
    "networks": [
      {
        "network_key": "ethereum",
        "network": "Ethereum Mainnet",
        "display_name": "Ethereum",
        "short": "ETH",
        "token_standard": "ERC-20",
        "address_type": "EVM Universal Address",
        "explorer_url": "https://etherscan.io",
        "wallet_id": "eth-1Be31A94",
        "address": "0x1Be31A94361a391bBaFB2a4CCd704F57dc04d4bb",
        "address_short": "0x1Be3...d4bb",
        "native_symbol": "ETH",
        "native_balance": "2",
        "usdt_balance": "1000",
        "total_value_usd": "7000.00",
        "change_24h_percent": 1.71,
        "has_wallet": true,
        "tokens": [
          { "token_address": "", "symbol": "ETH", "decimals": 18, "balance": "2",
            "price_usd": "3000", "value_usd": "6000.00", "change_24h": 2, "is_native": true },
          { "token_address": "0xdAC1...ec7", "symbol": "USDT", "decimals": 6, "balance": "1000",
            "price_usd": "1", "value_usd": "1000.00", "change_24h": 0, "is_native": false }
        ]
      }
    ]
  }
}
```

Kartu `networks` selalu berisi 5 jaringan (wallet yang belum ada tetap muncul
dengan `has_wallet: false`), diurutkan dari `total_value_usd` terbesar.
`POST /api/v1/wallets/refresh` mengembalikan bentuk yang sama setelah membaca
ulang saldo via RPC HTTP.

### `GET /api/v1/wallets/addresses` — modal alamat 2 kolom

```bash
curl http://localhost:5009/api/v1/wallets/addresses
```

```json
{
  "success": true,
  "data": {
    "evm": {
      "address_type": "EVM Universal Address",
      "address": "0x1Be31A94361a391bBaFB2a4CCd704F57dc04d4bb",
      "address_short": "0x1Be3...d4bb",
      "networks": ["Ethereum Mainnet", "BNB Chain", "Arbitrum One", "Robinhood Chain"],
      "shorts": ["ETH", "BNB", "ARB", "ROBINHOOD"],
      "qr_data": "0x1Be31A94361a391bBaFB2a4CCd704F57dc04d4bb"
    },
    "solana": null
  }
}
```

`solana` bernilai `null` bila `SOLANA_PRIVATE_KEY` belum diisi. Data QR cukup
memakai `qr_data` (alamat mentah); render QR dilakukan di frontend.

### `GET /api/v1/wallets/networks` — metadata tab

Mengembalikan array 5 tab (urutan UI) dengan `network_key`, `network`,
`display_name`, `short`, `token_standard`, `address_type`, dan `wallet_id`/
`address`/`address_short` bila wallet-nya ada.

### `POST /api/v1/wallets/{walletID}/rotate` — Perbaharui Address

```bash
curl -X POST http://localhost:5009/api/v1/wallets/eth-1Be31A94/rotate
```

```json
{
  "success": true,
  "message": "alamat baru dibuat",
  "data": {
    "chain": "evm",
    "network_key": "ethereum",
    "address_type": "EVM Universal Address",
    "old_address": "0x1Be31A94361a391bBaFB2a4CCd704F57dc04d4bb",
    "new_address": "0x9A7c...1122",
    "derivation_index": 1,
    "note": "Alamat lama tetap valid untuk menerima dana.",
    "wallets": [
      { "wallet_id": "eth-1Be31A94", "network": "Ethereum Mainnet",
        "old_address": "0x1Be3...d4bb", "new_address": "0x9A7c...1122", "derivation_index": 1 }
    ]
  }
}
```

Alamat baru diturunkan deterministik dari private key env pada index HD
berikutnya (lihat bagian *Private Keys*). Satu kali rotate pada jaringan EVM
memperbarui **keempat** wallet EVM sekaligus karena alamatnya universal.
Alamat lama tetap valid untuk menerima dana; `wallets` berisi semua wallet yang
ikut diperbarui.

### `POST /api/v1/wallets/{walletID}/send/estimate`

```bash
curl -X POST http://localhost:5009/api/v1/wallets/eth-1Be31A94/send/estimate \
  -H 'Content-Type: application/json' \
  -d '{"to":"0x000000000000000000000000000000000000dEaD","token_address":"","amount":"0.1"}'
```

`token_address` kosong = native; isi alamat kontrak untuk ERC-20/SPL.

```json
{
  "success": true,
  "data": {
    "network": "Ethereum Mainnet",
    "network_key": "ethereum",
    "token_symbol": "ETH",
    "token_address": "",
    "amount": "0.1",
    "native_symbol": "ETH",
    "fee_native": "0.000021",
    "fee_usd": "0.06",
    "gas_limit": 21000,
    "gas_price": "1000000000",
    "total_usd": "300.00"
  }
}
```

### `POST /api/v1/wallets/{walletID}/send`

Langkah konfirmasi. Bila PIN aktif, sertakan `pin`.

```bash
curl -X POST http://localhost:5009/api/v1/wallets/eth-1Be31A94/send \
  -H 'Content-Type: application/json' \
  -d '{"to":"0x000000000000000000000000000000000000dEaD","token_address":"","amount":"0.1","pin":"123456"}'
```

```json
{
  "success": true,
  "message": "transaksi dikirim",
  "data": {
    "tx_hash": "0xabc...",
    "network": "Ethereum Mainnet",
    "network_key": "ethereum",
    "token_symbol": "ETH",
    "amount": "0.1",
    "to": "0x000000000000000000000000000000000000dEaD",
    "explorer_url": "https://etherscan.io/tx/0xabc...",
    "status": "pending"
  }
}
```

Auto-lock jaringan mengikuti format `to`: alamat EVM (0x…) hanya untuk jaringan
EVM, alamat Solana base58 hanya untuk Solana.

### `GET /api/v1/contacts` — Contact Book

```json
{ "success": true, "data": [ { "id": "c1", "name": "Alice",
  "address": "0x…", "network": "Ethereum Mainnet", "notes": "" } ] }
```

### `GET/PUT /api/v1/settings/security` — PIN

```bash
curl http://localhost:5009/api/v1/settings/security
curl -X PUT http://localhost:5009/api/v1/settings/security \
  -H 'Content-Type: application/json' -d '{"pin":"123456"}'
```

```json
{ "success": true, "data": { "pin_enabled": true, "biometric_enabled": false, "auto_lock_minutes": 5 } }
```

PIN disimpan sebagai hash SHA-256 (pengaman dasar, bukan enkripsi private key)
dan diverifikasi pada langkah konfirmasi `send`.

## Mengisi logika

1. Buka `internal/modules/<domain>/handler.go` dan ganti isi method.
2. Tambahkan dependency/implementasi di `internal/modules/<domain>/service.go`.
3. Sesuaikan model di `internal/modules/<domain>/model.go` bila perlu.
4. Daftarkan dependency baru di `internal/infrastructure/router/router.go`.

Contoh yang sudah jadi: `internal/modules/wallet/` (endpoint dasar di
`service.go`/`handler.go`, logika halaman Wallet di `page.go`), serta
`internal/modules/contact/` dan `internal/modules/setting/`.

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
│   │   ├── wallet/        # handler.go, service.go, model.go, page.go
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
- `GET /wallets` - List all wallets (diperkaya harga + nilai USD)
- `POST /wallets` - Tambah wallet (import private key)
- `GET /wallets/overview` - Tab ALL: total USD + perubahan 24 jam + kartu jaringan
- `GET /wallets/addresses` - Modal alamat 2 kolom (EVM + Solana)
- `GET /wallets/networks` - Metadata tab jaringan
- `POST /wallets/refresh` - Baca ulang saldo semua wallet
- `GET /wallets/{id}` - Detail wallet (termasuk alamat + saldo token)
- `GET /wallets/{id}/balance` - Saldo wallet + ringkasan per jaringan
- `POST /wallets/{id}/refresh` - Baca ulang saldo satu wallet
- `POST /wallets/{id}/rotate` - Perbaharui Address (alamat baru)
- `POST /wallets/{id}/send/estimate` - Estimasi biaya kirim
- `POST /wallets/{id}/send` - Kirim transaksi (native/ERC-20/SPL)
- `GET /wallets/{id}/receive` - Alamat/QR receive
- `PUT /wallets/{id}/address` - Ganti alamat dari private key kiriman
- `DELETE /wallets/{id}` - Hapus wallet

Kontrak request/response lengkap ada di bagian **Kontrak API halaman Wallet**.

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

Sudah tersedia:
- Database schema untuk semua 5 halaman
- Wallet module berfungsi penuh (overview, alamat, rotate, refresh, send)
- Contact Book (baca daftar) dan pengaturan keamanan (PIN)
- Response models + network registry dengan default networks

**Selanjutnya:**
Implementasikan modul yang masih placeholder:
1. `internal/modules/portfolio/`
2. `internal/modules/transaction/`
3. `internal/modules/contact/` (create/update/delete)
4. `internal/modules/setting/` (RPC, backup)

Database sudah siap dengan semua tabel yang diperlukan.
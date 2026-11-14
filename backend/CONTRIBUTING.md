# Alur & Panduan Ngoding Backend

Dokumen ini menjelaskan alur request di backend HotWallet dan file mana yang harus
diubah. Backend memakai **Fiber v3 + SQLite**. Modul Wallet sudah berisi logika
nyata (endpoint dasar di `service.go`/`handler.go`, logika halaman di `page.go`);
modul lain sebagian masih placeholder dan tinggal diisi logika nyata.

---

## 1. Alur besar

```
┌─────────────────────────────────────────────────────────────────────┐
│  STARTUP                                                            │
│                                                                     │
│  cmd/api/main.go                                                    │
│    ├─ config.Load()          baca .env + environment                │
│    ├─ store.Open(DBPath)     buka/buat database SQLite              │
│    └─ router.New(cfg, st)    rakit handler → route → Fiber          │
│         └─ app.Listen(:PORT)                                        │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│  REQUEST                                                            │
│                                                                     │
│  HTTP request                                                       │
│    └─ middleware Fiber     recover → logger → CORS                  │
│         └─ router.go       cocokkan path ke handler                 │
│              └─ handler.go  bind JSON / ambil :id / pilih status    │
│                   └─ service.go  validasi + logika bisnis           │
│                        └─ crud/generic.go   CRUD generik (opsional) │
│                             └─ store/sqlite.go  Load/Save           │
│                                  └─ data/hotwallet.db                   │
│                                                                     │
│  balik ke atas: data → response.SendSuccess / response.SendError    │
└─────────────────────────────────────────────────────────────────────┘
```

Aturan pembagian tanggung jawab:

| Lapisan | Boleh berisi | Tidak boleh berisi |
|---|---|---|
| `handler.go` | bind JSON, ambil `:id`, pilih status code, panggil service | logika bisnis, akses database langsung |
| `service.go` | validasi, aturan bisnis, gabung beberapa repo | `fiber.Ctx`, status code |
| `model.go` | struct + tag JSON + `GetID`/`SetID` | akses database, HTTP |
| `pkg/crud/generic.go` | operasi koleksi generik | aturan bisnis domain |
| `store/sqlite.go` | koneksi, Load/Save, transaksi | tahu soal Contact/dll |
| `router.go` | wiring + daftar route | logika bisnis |

---

## 2. Peta folder

```
backend/
├── cmd/api/main.go
├── config/config.go
├── data/
│   ├── avatars/
│   └── hotwallet.db
├── internal/
│   ├── modules/
│   │   ├── contact/        handler.go, model.go, service.go
│   │   ├── portfolio/      handler.go, model.go, service.go
│   │   ├── setting/        handler.go, model.go, service.go
│   │   ├── transaction/    handler.go, model.go, service.go
│   │   └── wallet/         handler.go, model.go, service.go, page.go
│   ├── blockchain/
│   │   ├── wallet/         master.go, derivation.go, derive_address.go, address.go, keystore.go
│   │   ├── evm/            client.go, rpc.go, balance.go, ws.go, transfer.go, token.go, gas.go, transaction.go
│   │   ├── solana/         client.go, rpc.go, ws.go, token.go, transfer.go, balance.go, transaction.go
│   │   ├── network/        registry.go, network.go, rpc.go, defaults.go, meta.go
│   │   ├── tokenmeta/      defaults.go
│   │   ├── watcher/        manager.go
│   │   └── wsrpc/          wsrpc.go
│   ├── infrastructure/
│   │   ├── router/         router.go
│   │   ├── store/          sqlite.go, backup.go, schema.go, balances.go, rpc.go, wallet.go, token_prices.go, contact.go, security.go
│   │   ├── security/       encryption.go, keymanager.go
│   │   └── health/         handler.go
│   └── shared/
│       ├── errors/         errors.go
│       ├── interfaces/     services.go
│       ├── models/         api.go
│       └── types/          types.go
├── pkg/
│   ├── auth/               handler.go, middleware.go, service.go, token.go
│   ├── crud/               generic.go, pagination.go
│   ├── logger/             log.go
│   ├── response/           json.go
│   └── validator/          validator.go
├── go.mod
├── go.sum
└── .env
```

---

## 3. Peta domain

| Domain | Model utama | Tabel | Route |
|---|---|---|---|
| `wallet` | `Wallet`, `WalletBalance`, `NetworkSummary`, view halaman di `page.go` | `wallets`, `wallet_balances`, `network_summaries`, `token_prices` | `/api/v1/wallets` |
| `portfolio` | `PortfolioSummary`, `PortfolioToken`, `NetworkAsset` | `portfolio_summary`, `portfolio_tokens`, `network_assets` | `/api/v1/portfolio` |
| `transaction` | `Transaction` | `transactions` | `/api/v1/transactions` |
| `contact` | `Contact` | `contacts` | `/api/v1/contacts` |
| `setting` | `Setting`, `RPCConfig`, `SecuritySetting`, `BackupHistory` | `settings`, `rpc_configs`, `security_settings`, `backup_history` | `/api/v1/settings` |
| `health` | — | — | `GET /api/v1/health` |

---

## 4. Kontrak tiap lapisan

### `pkg/response/json.go`

```go
response.SendSuccess(c, fiber.StatusOK, data)
response.SendError(c, err)                       // status otomatis dari error
response.SendMessage(c, fiber.StatusOK, "ok")
response.SendPaginated(c, fiber.StatusOK, data, meta)
response.SendPlaceholder(c, fiber.StatusOK, "wallets")
```

### `pkg/crud/generic.go`

```go
repo := crud.NewRepo[models.Contact](st, "contacts")

repo.All()                       // ([]T, error)
repo.Get(id)                     // (T, bool, error)   bool = ketemu?
repo.Create(item)                // (T, error)         ID otomatis kalau kosong
repo.Update(id, item)            // (T, bool, error)   replace total
repo.Patch(id, patchJSON)        // (T, bool, error)   merge partial
repo.Delete(id)                  // (bool, error)
```

Error yang tersedia (dipakai `response.SendError`):

```go
crud.ErrNotFound      // → 404
crud.ErrInvalid       // → 400
crud.ErrConflict      // → 409
crud.ErrUnauthorized  // → 401
crud.ErrForbidden     // → 403
```

Helper: `crud.Invalid(...)`, `crud.NotFoundf(...)`, `crud.Conflictf(...)`.

### `pkg/crud/pagination.go`

```go
page, limit := crud.NormalizePage(page, limit, 100)
items, meta := crud.Paginate(allItems, page, limit, 100)
```

### `internal/infrastructure/store/sqlite.go`

```go
st, err := store.Open("data/hotwallet.db")
defer st.Close()

st.Load("contacts", &items)      // koleksi tidak ada → dst tetap kosong
st.Save("contacts", items)       // upsert
st.Delete("contacts")
st.Path()
st.Ping()
```

Skema tabel:

```sql
CREATE TABLE documents (
    name       TEXT PRIMARY KEY,
    data       TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
```

Satu baris = satu koleksi (dokumen JSON). Cocok untuk prototipe; bisa
dinormalisasi menjadi tabel per domain nanti.

### `internal/infrastructure/store/backup.go`

```go
path, err := st.Backup()         // → data/backups/<stamp>-hotwallet.db
err = st.Restore(path)           // salin isi backup ke database aktif
```

### `internal/blockchain/wallet`

```go
// Menggunakan private key langsung dari environment variables
// ETH_PRIVATE_KEY dan SOLANA_PRIVATE_KEY
```

Wallet menggunakan private key langsung tanpa seed phrase:
- `ETH_PRIVATE_KEY` untuk Ethereum & EVM chains (64 hex, `0x` opsional).
  Satu key ini membuat satu wallet per jaringan EVM dengan alamat yang sama.
- `SOLANA_PRIVATE_KEY` untuk Solana (base58/hex 32 atau 64 byte, atau JSON
  array `solana-keygen`). Kurva ed25519, jadi tidak bisa memakai key EVM.

Private keys dibaca dari environment variable dan **tidak disimpan ke database**.
Saat startup, `AutoCreateWalletFromEnv` membuat wallet yang belum ada. Kalau
key diganti, alamat wallet tunggal di jaringan itu ikut diperbarui (deteksi
dari perbandingan alamat hasil derivasi); saldo lama dihapus. Jaringan dengan
lebih dari satu wallet dilewati dan harus di-update lewat `PUT /address`.

Rotate alamat ("Perbaharui Address") memakai derivasi HD deterministik dari key
yang sama (`derive_address.go`): index 0 = alamat langsung dari key, index > 0
diturunkan dari seed `SHA256("HotWallet/address/v1/" || key)` via BIP32 (EVM)
atau SLIP-0010 (Solana). Index disimpan di `wallets.derivation_index` sehingga
alamat hasil rotate tetap sama setelah restart. Satu rotate EVM memperbarui
keempat wallet EVM sekaligus.

### `internal/modules/wallet/page.go`

Berisi seluruh logika halaman Wallet: `Overview` (tab ALL), `Addresses` (modal
2 kolom), `Networks` (metadata tab), `Rotate`, `RefreshAll`/`RefreshWallet`,
`SendEstimate`, dan `Send`. Nilai USD dihitung saat baca dari `token_prices`
(stablecoin fallback $1). Handler tipis di `handler.go` memanggil method ini.

### `pkg/auth/token.go`

```go
token, err := auth.Generate(secret, "user-id", 24*time.Hour)
claims, err := auth.Verify(secret, token)
auth.Bearer(c.Get(fiber.HeaderAuthorization))
hash := auth.HashPIN("123456")
auth.VerifyPIN("123456", hash)
```

Middleware `auth.Protect(secret, "/api/v1/health", "/api/v1/settings/security")`
sudah ada, tapi belum dipasang di router.

---

## 5. Resep ngoding manual

### A. Mengisi endpoint placeholder

Contoh untuk `internal/modules/transaction/handler.go`:

```go
func (h *Handler) List(c fiber.Ctx) error {
    items, err := h.svc.List()
    if err != nil {
        logger.Error("task.List", err)
        return response.SendError(c, err)
    }
    return response.SendSuccess(c, fiber.StatusOK, items)
}
```

Lalu tambahkan method di `service.go` dan dependency yang dibutuhkan.

### B. Menambah / mengubah field data

Cukup edit model:

```go
// internal/modules/contact/model.go
type Contact struct {
    // ...
    Note string `json:"note"`
}
```

Handler dan service tidak perlu diubah karena `c.Bind().JSON(&p)` mengikat
seluruh struct dan repo meneruskan apa adanya.

Bila field wajib, tambahkan validasi di **service**, bukan handler:

```go
func (s *Service) Create(x Contact) (Contact, error) {
    if x.Address == "" {
        return Contact{}, crud.Invalid("address wajib diisi")
    }
    return s.repo.Create(x)
}
```

### C. Menambah endpoint read-only

1. Tulis method di `service.go`:
   ```go
   func (s *Service) Summary() (map[string]int, error) { ... }
   ```
2. Tulis handler di `handler.go`:
   ```go
   func (h *Handler) Summary(c fiber.Ctx) error {
       data, err := h.svc.Summary()
       if err != nil {
           logger.Error("transaction.Summary", err)
           return response.SendError(c, err)
       }
       return response.SendSuccess(c, fiber.StatusOK, data)
   }
   ```
3. Daftarkan di `router.go`:
   ```go
   walletGroup.Get("/:walletID/nets", func(c fiber.Ctx) error {
       return response.SendPlaceholder(c, fiber.StatusNotImplemented, "wallet networks")
   })
   ```
   Letakkan route statis **sebelum** route `/:id` agar tidak dianggap sebagai ID.

### D. Menambah endpoint fitur baru

Semua endpoint fitur berada di grup `/api/v1` pada `router.go`. Tambahkan
handler di `internal/modules/<domain>/handler.go`, lalu daftarkan di grup yang
sesuai:

```go
walletGroup := apiV1.Group("/wallets")
walletGroup.Post("/:walletID/send", walletH.Send)
```

Wallet, `GET /contacts`, dan `settings/security` sudah memakai handler asli.
Modul lain masih skeleton dan `router.go` memakai placeholder inline; setelah
handler diimplementasikan, ganti placeholder tersebut dengan method handler
aslinya. Helper `registerCRUD` masih tersedia di `router.go`, tetapi route 5
fitur didaftarkan eksplisit agar path-nya tetap sesuai.

### E. Memakai pagination di endpoint list

```go
func (h *Handler) List(c fiber.Ctx) error {
    items, err := h.svc.List()
    if err != nil {
        return response.SendError(c, err)
    }
    page, limit := fiber.Query[int](c, "page", 1), fiber.Query[int](c, "limit", 20)
    paged, meta := crud.Paginate(items, page, limit, 100)
    return response.SendPaginated(c, fiber.StatusOK, paged, response.Meta{
        Page:       meta.Page,
        Limit:      meta.Limit,
        Total:      meta.Total,
        TotalPages: meta.TotalPages,
    })
}
```

### F. Mengaktifkan proteksi token

```go
// router.go, setelah middleware CORS
app.Use(auth.Protect([]byte(secret),
    "/api/v1/health",
    "/api/v1/settings/security",
))
```

Saat ini sengaja tidak dipasang.

### G. Backup sebelum eksperimen

```go
path, err := st.Backup()
// ... eksperimen ...
err = st.Restore(path)
```

---

## 6. Aturan main

1. **Satu perubahan, satu lapisan.** Ubah model saja kalau hanya menambah field.
2. **Wiring hanya di `router.go`.** Service/handler baru tidak akan jalan kalau
   belum didaftarkan di sana.
3. **Error bisnis pakai sentinel `crud.Err*`**, jangan `fiber.NewError` ad-hoc,
   supaya `response.SendError` konsisten.
4. **Jangan taruh `fiber.Ctx` di service** — service harus bisa dites tanpa HTTP.
5. **Hati-hati mengubah `pkg/crud/generic.go` dan `store/sqlite.go`** — dipakai
   semua domain.

---

## 7. Checklist setelah ngoding

```bash
cd backend

# 1. format
gofmt -w ./cmd ./config ./pkg ./internal

# 2. compile
go build ./...

# 3. jalankan
PORT=5022 go run ./cmd/api
```

Tes endpoint yang berubah:

```bash
curl -s http://localhost:5022/api/v1/health
curl -s http://localhost:5022/api/v1/wallets
curl -s -X POST http://localhost:5022/api/v1/contacts \
  -H 'Content-Type: application/json' \
  -d '{"name":"Cek manual","address":"0x0000000000000000000000000000000000000000","network":"Ethereum Mainnet"}'
```

Kalau memakai binary (lebih aman dari proses `go run` yang tertinggal):

```bash
go build -o /tmp/hotwallet-api ./cmd/api
PORT=5022 /tmp/hotwallet-api
```

---

## 8. Debug cepat

| Gejala | Cek |
|---|---|
| `404` di endpoint baru | sudah didaftarkan di `router.go`? urutan route `/:id`? |
| `500` padahal build OK | data koleksi rusak / field tidak cocok; lihat log `logger.Error` |
| `405 Method Not Allowed` | method route beda (GET vs POST/PUT/PATCH) |
| Data tidak berubah | nama koleksi/tabel salah (`contacts` vs `wallets`) |
| `database is locked` | ada proses lain memakai DB yang sama; hentikan dulu |
| `CGO_ENABLED=0` | driver SQLite butuh CGO; aktifkan dan sediakan GCC |
| Perubahan tidak muncul | server lama masih jalan di port yang sama; matikan dulu |

---

## 9. Referensi route

| Method | Path |
|---|---|
| GET | `/` |
| GET | `/api/v1/health` |
| GET/POST | `/api/v1/wallets` |
| GET | `/api/v1/wallets/overview` |
| GET | `/api/v1/wallets/addresses` |
| GET | `/api/v1/wallets/networks` |
| POST | `/api/v1/wallets/refresh` |
| GET/DELETE | `/api/v1/wallets/:walletID` |
| GET | `/api/v1/wallets/:walletID/balance` |
| POST | `/api/v1/wallets/:walletID/refresh` |
| POST | `/api/v1/wallets/:walletID/rotate` |
| POST | `/api/v1/wallets/:walletID/send/estimate` |
| POST | `/api/v1/wallets/:walletID/send` |
| GET | `/api/v1/wallets/:walletID/receive` |
| PUT | `/api/v1/wallets/:walletID/address` |
| GET | `/api/v1/portfolio/summary` |
| GET | `/api/v1/portfolio/tokens` |
| GET | `/api/v1/portfolio/networks` |
| GET | `/api/v1/transactions` |
| GET | `/api/v1/transactions/:txHash` |
| GET | `/api/v1/transactions/:txHash/status` |
| GET/POST | `/api/v1/contacts` |
| GET/PUT/DELETE | `/api/v1/contacts/:contactID` |
| GET/PUT | `/api/v1/settings/rpc` |
| GET/PUT | `/api/v1/settings/security` |
| GET/POST | `/api/v1/settings/backup` |
| GET | `/avatars/*` |

Endpoint Wallet, `GET /contacts`, dan `GET/PUT /settings/security` sudah
berfungsi. Endpoint lain masih mengembalikan `501 Not Implemented` sampai
handler di `internal/modules/*` diisi. Path di luar tabel ini (misalnya
`/api/dashboard`, `/api/tasks`, `/api/auth/*`) mengembalikan `404`.

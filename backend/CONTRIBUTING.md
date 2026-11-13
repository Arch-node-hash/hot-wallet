# Alur & Panduan Ngoding Backend

Dokumen ini menjelaskan alur request di backend HotWallet dan file mana yang harus
diubah. Backend memakai **Fiber v3 + SQLite**. Handler/service masih berupa
placeholder, tinggal diisi logika nyata.

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
| `store/sqlite.go` | koneksi, Load/Save, transaksi | tahu soal Task/Contact/dll |
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
│   │   ├── activity/       handler.go, model.go, service.go
│   │   ├── contact/        handler.go, model.go, service.go
│   │   ├── portfolio/      handler.go, model.go, service.go
│   │   ├── setting/        handler.go, model.go, service.go
│   │   ├── transaction/    handler.go, model.go, service.go
│   │   └── wallet/         handler.go, model.go, service.go
│   ├── blockchain/
│   │   ├── wallet/         master.go, derivation.go, address.go, keystore.go
│   │   ├── evm/            client.go, balance.go, transfer.go, token.go, gas.go, transaction.go
│   │   ├── solana/         client.go, balance.go, transfer.go, token.go, transaction.go
│   │   └── network/        registry.go, network.go, rpc.go
│   ├── infrastructure/
│   │   ├── router/         router.go
│   │   ├── store/          sqlite.go, backup.go, migrations/
│   │   ├── security/       encryption.go, keymanager.go
│   │   └── health/         handler.go
│   └── shared/
│       ├── errors/         errors.go
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

| Domain | Struct | Koleksi SQLite | Route |
|---|---|---|---|
| `transaction` | `Task` | `tasks` | `/api/tasks` |
| `transaction` | `Project` | `projects` | `/api/projects` |
| `contact` | `Member` | `team` | `/api/team` |
| `activity` | `Event` | `events` | `/api/calendar/events` |
| `activity` | `Reminder` | `reminders` | `/api/reminders` |
| `wallet` | `TimeEntry` | `time_entries` | `/api/time-entries` |
| `wallet` | `Response` (agregat) | `dashboard` | `GET /api/dashboard` |
| `portfolio` | `Analytics` | `analytics` | `GET /api/analytics` |
| `setting` | `Settings` | `settings` | `GET/PUT /api/settings` |
| `auth` | `User` | `users` | `GET /api/auth/me` |
| `health` | — | — | `GET /api/health` |

---

## 4. Kontrak tiap lapisan

### `pkg/response/json.go`

```go
response.SendSuccess(c, fiber.StatusOK, data)
response.SendError(c, err)                       // status otomatis dari error
response.SendMessage(c, fiber.StatusOK, "ok")
response.SendPaginated(c, fiber.StatusOK, data, meta)
response.SendPlaceholder(c, fiber.StatusOK, "tasks")
```

### `pkg/crud/generic.go`

```go
repo := crud.NewRepo[transaction.Task](st, "tasks")

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

st.Load("tasks", &items)         // koleksi tidak ada → dst tetap kosong
st.Save("tasks", items)          // upsert
st.Delete("tasks")
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

### `pkg/auth/token.go`

```go
token, err := auth.Generate(secret, "user-id", 24*time.Hour)
claims, err := auth.Verify(secret, token)
auth.Bearer(c.Get(fiber.HeaderAuthorization))
hash := auth.HashPIN("123456")
auth.VerifyPIN("123456", hash)
```

Middleware `auth.Protect(secret, "/api/health", "/api/auth/...")` sudah ada,
tapi belum dipasang di router.

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
// internal/modules/transaction/model.go
type Task struct {
    // ...
    Description string `json:"description"`
}
```

Handler dan service tidak perlu diubah karena `c.Bind().JSON(&p)` mengikat
seluruh struct dan repo meneruskan apa adanya.

Bila field wajib, tambahkan validasi di **service**, bukan handler:

```go
func (s *Service) Create(t Task) (Task, error) {
    if t.Title == "" {
        return Task{}, crud.Invalid("title wajib diisi")
    }
    return s.repo.Create(t)
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
   app.Get("/api/tasks/summary", taskH.Summary)
   ```
   Letakkan **sebelum** route `/:id` agar tidak dianggap sebagai ID.

### D. Menambah resource CRUD baru

Buat tiga file di `internal/modules/<domain>/`:

```
model.go      struct + GetID/SetID
service.go    wrapper crud.Repo
handler.go    6 method CRUD
```

Lalu di `router.go`:

```go
repo := crud.NewRepo[<domain>.Item](st, "items")
svc := <domain>.NewService(repo)
h := <domain>.NewHandler(svc)
registerCRUD(app, "/api/items", h)
```

`registerCRUD` otomatis membuat:

```
GET    /api/items
POST   /api/items
GET    /api/items/:id
PUT    /api/items/:id
PATCH  /api/items/:id
DELETE /api/items/:id
```

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
    "/api/health",
    "/api/auth/login",
    "/api/auth/unlock",
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
curl -s http://localhost:5022/api/health
curl -s http://localhost:5022/api/tasks
curl -s -X POST http://localhost:5022/api/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Cek manual"}'
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
| Data tidak berubah | nama koleksi salah (`tasks` vs `projects`) |
| `database is locked` | ada proses lain memakai DB yang sama; hentikan dulu |
| `CGO_ENABLED=0` | driver SQLite butuh CGO; aktifkan dan sediakan GCC |
| Perubahan tidak muncul | server lama masih jalan di port yang sama; matikan dulu |

---

## 9. Referensi route

| Method | Path |
|---|---|
| GET | `/` |
| GET | `/api/health` |
| GET | `/api/auth/me` |
| POST | `/api/auth/logout` |
| GET | `/api/dashboard` |
| GET | `/api/analytics` |
| GET/PUT | `/api/settings` |
| POST | `/api/settings/avatar` |
| GET/POST | `/api/tasks`, `/api/projects`, `/api/team`, `/api/reminders`, `/api/calendar/events`, `/api/time-entries` |
| GET/PUT/PATCH/DELETE | `.../<resource>/:id` |
| GET | `/avatars/*` |

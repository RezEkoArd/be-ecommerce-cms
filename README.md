# be-ecommerce-cms

Backend API untuk **E-Commerce CMS** — layanan yang menyediakan REST API untuk
mengelola konten dan operasi toko online: produk, kategori, order, dan
pengguna. Dipakai oleh dashboard admin (CMS) dan storefront frontend
(`fe-cms-ecommerce`).

Dibangun dengan Go: **Gin + GORM v2 + PostgreSQL + Viper + zerolog**, dengan
autentikasi JWT (access + refresh token).

## Stack

- **Web Framework:** Gin (`github.com/gin-gonic/gin`)
- **ORM:** GORM v2 (`gorm.io/gorm`) + PostgreSQL driver (`gorm.io/driver/postgres`)
- **Config:** Viper (`github.com/spf13/viper`) — baca `.env` + system env
- **Logging:** zerolog via wrapper `pkg/logger`
- **Auth:** JWT (access token via header, refresh token via cookie HttpOnly)
- **Testing:** Testify (`github.com/stretchr/testify`)

## Fitur yang Direncanakan

Cakupan aplikasi yang ingin dibangun. Tanda status menandai progres saat ini.

| Domain | Deskripsi | Status |
|---|---|---|
| **Auth** | Register, login, refresh token, logout; JWT access + refresh dengan rotation | 🚧 Scaffolding |
| **User** | Manajemen akun & role (admin CMS vs customer) | 🚧 Scaffolding |
| **Product** | CRUD produk, harga, stok, gambar | 📋 Direncanakan |
| **Category** | CRUD kategori & relasi produk-kategori | 📋 Direncanakan |
| **Cart** | Keranjang belanja per customer | 📋 Direncanakan |
| **Order** | Checkout, riwayat order, status order | 📋 Direncanakan |

Legend: ✅ Selesai · 🚧 Sedang dikerjakan · 📋 Direncanakan

## Prasyarat

- Go 1.25+
- PostgreSQL 14+

## Setup

```bash
# 1. Salin env template dan isi nilainya
cp .env.example .env

# 2. Isi minimal: DB_* dan JWT_SECRET
#    Generate JWT secret: openssl rand -base64 32

# 3. Download dependency
go mod download

# 4. Jalankan aplikasi
go run cmd/main.go
```

Server default jalan di `http://localhost:8080`. Cek health check:

```bash
curl http://localhost:8080/health
# {"code":200,"message":"ok","data":null}
```

## Environment Variables

Lihat [.env.example](.env.example) untuk daftar lengkap.

| Variable | Deskripsi | Default |
|---|---|---|
| `APP_PORT` | Port HTTP server | `8080` |
| `APP_ENV` | `development` (log console berwarna) / `production` (log JSON) | `development` |
| `DB_HOST` | Host PostgreSQL | `localhost` |
| `DB_PORT` | Port PostgreSQL | `5432` |
| `DB_USER` | User database | — |
| `DB_PASSWORD` | Password database | — (wajib, tanpa default) |
| `DB_NAME` | Nama database | `ecommerce_cms` |
| `DB_SSLMODE` | Mode SSL koneksi DB | `disable` |
| `JWT_SECRET` | Secret untuk sign JWT | — (wajib, tanpa default) |

## Build & Test

- **Run app:** `go run cmd/main.go`
- **Build:** `go build -o bin/be-ecommerce-cms cmd/main.go`
- **Test all:** `go test ./... -v -cover`
- **Test single package:** `go test ./internal/auth/... -v`
- **Lint:** `golangci-lint run ./...`

## Struktur Project

```
cmd/main.go              → Entry point, dependency injection, graceful shutdown
internal/config/         → Load config via Viper
internal/database/       → Koneksi PostgreSQL (GORM)
internal/domain/         → Struct entity & error domain
internal/auth/           → Autentikasi: handler, service, JWT
internal/user/           → Manajemen user (repository, dst)
pkg/logger/              → Wrapper logging zerolog
pkg/response/            → Format response HTTP konsisten
migrations/              → File SQL migration (up/down)
```

## Arsitektur

Dependency mengalir satu arah: **Handler → Service → Repository**.

- **Handler** — hanya tangani HTTP: parse request, panggil service, return response
- **Service** — hanya business logic; tidak boleh import `gin`, `gorm`, atau `sql`
- **Repository** — hanya akses DB via GORM; tidak boleh ada business logic

Setiap layer didefinisikan sebagai **interface**, bukan concrete struct.

Detail konvensi coding, logging, dan keamanan ada di [CLAUDE.md](CLAUDE.md) dan
`.claude/rules/`.

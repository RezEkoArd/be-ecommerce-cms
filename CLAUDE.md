# <NAMA_PROJECT> — Go Backend

<DESKRIPSI_SINGKAT_PROJECT — satu kalimat: API untuk apa, dipakai siapa.>

## Stack
- **Web Framework:** Gin (`github.com/gin-gonic/gin`)
- **ORM:** GORM v2 (`gorm.io/gorm`) + PostgreSQL driver (`gorm.io/driver/postgres`)
- **Config:** godotenv (`github.com/joho/godotenv`) + os.Getenv
- **Logging:** zerolog via wrapper `pkg/logger`
- **Migration:** Golang-Migrate v4 (`github.com/golang-migrate/migrate/v4`)
- **Testing:** Testify (`github.com/stretchr/testify`)

## Build & Test Commands
- **Run app:** `go run cmd/main.go`
- **Build:** `go build -o bin/<NAMA_BINARY> cmd/main.go`
- **Test all:** `go test ./... -v -cover`
- **Test single package:** `go test ./internal/service/... -v`
- **Lint:** `golangci-lint run ./...`
- **Migration up:** `make migrate-up`
- **Migration down:** `make migrate-down`
- **Create migration:** `make migrate-create` (lalu ketik nama migration)

## Project Structure
```
cmd/main.go              → Entry point, dependency injection, graceful shutdown
internal/config/         → Load config via godotenv
internal/model/          → Struct entity (GORM model + business types)
internal/handler/        → HTTP handler Gin (HANYA parsing request/response)
internal/midleware/      → Gin middleware (auth, logger, rate limit)
internal/repository/     → Query database via GORM (HANYA akses DB)
internal/service/        → Business logic (tidak boleh tahu soal HTTP atau DB)
migrations/              → File SQL up/down
```

## Architecture Rules
- **Handler** → hanya tangani HTTP: parse request, call service, return response
- **Service** → hanya business logic: tidak boleh import `gin`, `gorm`, `sql`
- **Repository** → hanya akses DB: tidak boleh ada business logic di sini
- **Dependency arah:** handler → service → repository (satu arah, tidak bolak-balik)
- Selalu definisikan layer sebagai **interface**, bukan concrete struct

## Coding Conventions

### Error Handling
```go
// Selalu wrap error dengan konteks
return nil, fmt.Errorf("service.GetUser: %w", err)

// Jangan panic untuk error runtime biasa
// Jangan return error database mentah ke HTTP response
```

### Logging
- Selalu pakai `pkg/logger`, jangan import `log` standar atau `zerolog` langsung
- Log hanya di **handler** dan **repository**, tidak di service
- Sertakan field konteks yang relevan saat log error (`user_id`, `email`, dll)
- Jangan log data sensitif: password, token, secret
- Bedakan logger (server) dan response (client) — keduanya dipakai bersamaan di handler

```go
// Repository: log error lalu wrap dan return
logger.Errorf("userRepository.FindByID failed", err, map[string]any{"user_id": id})
return nil, fmt.Errorf("userRepository.FindByID: %w", err)

// Handler: log error + kirim response user-friendly ke client
logger.Errorf("handler.Login failed", err, map[string]any{"email": req.Email})
c.JSON(http.StatusUnauthorized, response.NewResponse(401, "email atau password salah", nil))
```

| Level | Kapan dipakai |
|---|---|
| `logger.Debug` | Detail internal, hanya development |
| `logger.Info` | Event normal: server start, DB connect |
| `logger.Warn` | Tidak ideal tapi app masih jalan |
| `logger.Error` | Error di proses bisnis, app tidak berhenti |
| `logger.Fatal` | Error fatal saat startup, app berhenti |

### Context
```go
// Selalu teruskan ctx dari handler ke repository
func (s *userService) GetByID(ctx context.Context, id uint) (*model.User, error)
func (r *userRepo) FindByID(ctx context.Context, id uint) (*model.User, error)
```

### HTTP Response
```go
// Format response konsisten pakai pkg/response
c.JSON(http.StatusOK, response.NewResponse(200, "berhasil", result))
c.JSON(http.StatusBadRequest, response.NewResponse(400, "pesan error yang user-friendly", nil))
// Jangan expose error internal ke client
```

### Config
- Semua config dari environment variable via godotenv
- File `.env` tidak boleh di-commit
- Gunakan `config.Load()` di main.go, inject ke dependency

### Database / GORM
- Selalu pakai `WithContext(ctx)` sebelum query
- Gunakan `SoftDelete` (field `DeletedAt gorm.DeletedAt`) untuk data penting
- Migration menggunakan file SQL, bukan `AutoMigrate` di production
- Index untuk semua foreign key dan kolom yang sering di-query

### Testing
- Setiap service wajib punya unit test dengan mock repository
- Mock dibuat manual menggunakan `testify/mock`, bukan generate tool
- Gunakan `require.NoError` jika langkah berikutnya bergantung pada hasil
- Gunakan `assert.Equal` untuk verifikasi nilai

## Rules Maintenance
- Detail rules per-area ada di `.claude/rules/` (dimuat otomatis sesuai path file yang dikerjakan): `backend.md`, `api-design.md`, `logging.md`, `init.md`, `auth.md`, `security.md`
- `/check-convention` — jalankan sebelum mulai coding untuk checklist konvensi yang relevan
- `/add-rule` — tambah rule/contoh baru saat agent menulis kode yang tidak sesuai pattern
- `/audit-rules` — audit berkala agar rules tetap lean (setiap selesai feature besar / 2 minggu)

## Git Conventions
- Branch: `feature/nama-fitur`, `fix/nama-bug`, `refactor/nama-area`
- Commit: `feat: tambah endpoint create user`, `fix: perbaiki query N+1 di endpoint list`
- Jangan commit file: `.env`, `bin/`, `*.log`

## Security Rules
- Jangan log password, token, atau data sensitif
- Selalu validasi input dengan binding tag Gin sebelum proses
- Validasi input pakai **allowlist** (format/karakter diizinkan), bukan blacklist pola XSS; input tak valid → 400, jangan disamarkan jadi 401 (detail: `api-design.md` §6)
- Gunakan `bcrypt` untuk hash password, bukan MD5/SHA1
- JWT secret wajib dari environment variable, tidak boleh hardcode
- Auth: access token via `Authorization: Bearer` (FE simpan di memory), refresh token via cookie `HttpOnly`+`Secure`+`SameSite` khusus endpoint refresh, dengan rotation + reuse detection (detail: `auth.md`)

## Hal yang JANGAN Dilakukan
- Jangan import `gorm` di layer service
- Jangan import `gin` di layer repository atau service
- Jangan return `gorm.ErrRecordNotFound` langsung ke HTTP handler
- Jangan pakai `AutoMigrate` di production
- Jangan simpan logic bisnis di handler
- Jangan gunakan `interface{}` jika bisa pakai type yang spesifik

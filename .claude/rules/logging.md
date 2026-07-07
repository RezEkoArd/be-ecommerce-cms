---
paths:
  - "internal/**/*.go"
  - "cmd/**/*.go"
---

# Rules: Logging dengan Zerolog

---

## 1. Selalu pakai `pkg/logger`, jangan import `log` standar atau `zerolog` langsung

Semua logging harus lewat wrapper `pkg/logger` agar format dan behavior konsisten di seluruh project.

### ✅ Prinsip
```go
import "github.com/rezekoard/larobas-api/pkg/logger"

logger.Info("server started")
logger.Error("service.GetUser failed", err)
```

### ❌ Anti-Pattern
```go
import "log"
import "github.com/rs/zerolog"

log.Println("server started")      // JANGAN: bypass wrapper
zerolog.New(os.Stdout).Info().Msg() // JANGAN: langsung pakai zerolog
fmt.Println("debug:", err)          // JANGAN: pakai fmt untuk logging
```

---

## 2. Pakai level log yang tepat sesuai konteks

Setiap level punya makna spesifik. Jangan semua pakai `Info` atau `Error`.

| Level | Fungsi | Contoh Penggunaan |
|---|---|---|
| `Debug` | Detail internal, hanya development | nilai variabel, langkah eksekusi |
| `Info` | Event normal yang penting | server start, koneksi DB berhasil |
| `Warn` | Tidak ideal tapi app masih jalan | file .env tidak ditemukan, fallback ke default |
| `Error` | Error yang perlu diperhatikan, app tidak berhenti | query gagal, validasi gagal di service |
| `Fatal` | Error fatal, app harus berhenti | gagal koneksi DB saat startup |

### ✅ Prinsip
```go
// Startup & infrastruktur → Info
logger.Info("Shutting down server...")
logger.Infof("Database connected", map[string]any{"host": cfg.DBHost, "name": cfg.DBName})

// Tidak ideal tapi tidak kritis → Warn
logger.Warn("No .env file found, using system env")

// Error di dalam proses bisnis → Error
logger.Errorf("userService.GetByID failed", err, map[string]any{"user_id": id})

// Gagal fatal saat startup → Fatal
logger.Fatal("Failed to connect to database", err)

// Detail untuk debugging → Debug
logger.Debugf("query executed", map[string]any{"table": "users", "rows": count})
```

### ❌ Anti-Pattern
```go
logger.Info("query failed: " + err.Error()) // JANGAN: salah level, pakai Error
logger.Fatal("user not found", err)          // JANGAN: user not found bukan fatal
logger.Error("server started", nil)          // JANGAN: salah level, pakai Info
```

---

## 3. Logging hanya di handler dan repository, tidak di service

Service tidak boleh tahu soal logging infrastruktur. Service cukup wrap dan return error, lalu biarkan handler atau repository yang log.

### ✅ Prinsip
```go
// Repository: log error DB, lalu wrap dan return
func (r *userRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
    var user model.User
    err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
    if err != nil {
        logger.Errorf("userRepository.FindByID failed", err, map[string]any{"user_id": id})
        return nil, fmt.Errorf("userRepository.FindByID: %w", err)
    }
    return &user, nil
}

// Service: cukup wrap error, tidak perlu log
func (s *userService) GetByID(ctx context.Context, id string) (*model.User, error) {
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("userService.GetByID: %w", err)
    }
    return user, nil
}

// Handler: log error final yang sampai ke HTTP layer
func (h *UserHandler) GetByID(c *gin.Context) {
    user, err := h.service.GetByID(ctx, id)
    if err != nil {
        logger.Errorf("handler.GetByID failed", err, map[string]any{"user_id": id})
        c.JSON(http.StatusInternalServerError, response.NewResponse(500, "terjadi kesalahan internal", nil))
        return
    }
    c.JSON(http.StatusOK, response.NewResponse(200, "berhasil", user))
}
```

### ❌ Anti-Pattern
```go
// Service melakukan logging — service tidak boleh tahu soal infrastruktur log
func (s *userService) GetByID(ctx context.Context, id string) (*model.User, error) {
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        logger.Error("gagal ambil user", err) // JANGAN: log di service
        return nil, err
    }
    return user, nil
}
```

---

## 4. Selalu sertakan field konteks yang relevan saat log error

Log tanpa konteks tidak membantu debugging. Sertakan identifier yang memudahkan penelusuran.

### ✅ Prinsip
```go
// Di repository: sertakan identifier data yang dicari
logger.Errorf("userRepository.FindByID failed", err, map[string]any{
    "user_id": id,
})

// Di handler: sertakan identifier request
logger.Errorf("handler.Login failed", err, map[string]any{
    "email": req.Email,
})

// Di repository saat create: sertakan data yang hendak disimpan
logger.Errorf("postRepository.Create failed", err, map[string]any{
    "title":     post.Title,
    "author_id": post.AuthorID,
})
```

### ❌ Anti-Pattern
```go
logger.Error("gagal", err)                    // JANGAN: tidak ada konteks sama sekali
logger.Error("userRepo.FindByID failed", err) // KURANG: tidak ada user_id-nya
```

---

## 5. Jangan log data sensitif

Password, token, dan data pribadi tidak boleh masuk ke log dalam bentuk apapun.

### ✅ Prinsip
```go
// Log hanya identifier, bukan data sensitif
logger.Errorf("handler.Login failed", err, map[string]any{
    "email": req.Email, // email boleh, karena bukan credential
})
```

### ❌ Anti-Pattern
```go
logger.Errorf("handler.Login failed", err, map[string]any{
    "password": req.Password, // JANGAN: password tidak boleh di log
    "token":    token,        // JANGAN: token tidak boleh di log
})
```

---

## 6. Bedakan log server (logger) dan response client (response.NewResponse)

Logger dan response adalah dua hal yang **berbeda dan tidak saling menggantikan**.

| | Logger | Response |
|---|---|---|
| Tujuan | Catat di server | Kirim ke client/frontend |
| Isi | Detail teknis internal | Pesan user-friendly |
| Dilihat oleh | Developer/DevOps | Frontend/user |

### ✅ Prinsip — keduanya dipakai bersamaan di handler
```go
func (h *UserHandler) Login(c *gin.Context) {
    user, err := h.service.Login(ctx, req)
    if err != nil {
        // Logger: catat detail teknis di server
        logger.Errorf("handler.Login failed", err, map[string]any{"email": req.Email})

        // Response: kirim pesan yang aman ke frontend
        c.JSON(http.StatusUnauthorized, response.NewResponse(401, "email atau password salah", nil))
        return
    }
    c.JSON(http.StatusOK, response.NewResponse(200, "login berhasil", user))
}
```

### ❌ Anti-Pattern
```go
// Kirim error internal ke client — bocorkan detail server
c.JSON(500, response.NewResponse(500, err.Error(), nil)) // JANGAN

// Tidak log sama sekali — susah debugging
c.JSON(500, response.NewResponse(500, "terjadi kesalahan", nil)) // KURANG: tidak ada log
```

---
paths:
  - "cmd/**/*.go"
  - "internal/config/**/*.go"
---

# Rules: Inisialisasi Project

Urutan inisialisasi di `cmd/main.go` harus selalu konsisten dan berurutan.
Setiap komponen bergantung pada komponen sebelumnya — jangan diubah urutannya.

---

## Urutan Wajib di main.go

```
1. config.Load()     → baca .env dan set semua config
2. logger.Init()     → inisialisasi logger (butuh AppEnv dari config)
3. database.Connect() → koneksi ke DB (butuh DB config)
4. setup router      → daftarkan semua route dan middleware
5. start server      → jalankan HTTP server dengan graceful shutdown
```

### ✅ Prinsip
```go
func main() {
    // 1. Load config — selalu pertama, semua komponen butuh ini
    cfg := config.Load()

    // 2. Init logger — harus sebelum komponen lain agar semua log tertangkap
    logger.Init(cfg.AppEnv)

    // 3. Connect database — setelah logger agar error koneksi bisa di-log
    db := database.Connect(cfg)

    // 4. Setup router — inject semua dependency ke handler
    r := setupRouter(db, cfg)

    // 5. Start server dengan graceful shutdown
    srv := &http.Server{Addr: ":" + cfg.AppPort, Handler: r}
    // ... graceful shutdown
}
```

### ❌ Anti-Pattern
```go
// Logger diinit setelah database — error koneksi DB tidak tertangkap logger
db := database.Connect(cfg)
logger.Init(cfg.AppEnv) // JANGAN: logger harus lebih awal

// Config tidak di-load di awal — komponen lain tidak punya config
logger.Init("development") // JANGAN: hardcode env, harus dari config
database.Connect(nil)      // JANGAN: config nil
```

---

## Rules Config (`internal/config/config.go`)

- Gunakan **Viper** untuk semua pembacaan config — tidak perlu godotenv lagi
- `viper.SetConfigFile(".env")` untuk baca file `.env`
- `viper.AutomaticEnv()` untuk baca system environment variable
- `viper.SetDefault()` hanya untuk nilai fallback yang aman
- Jangan beri default value untuk config sensitif (`JWT_SECRET`, `DB_PASSWORD`) — biarkan kosong agar mudah dideteksi kalau belum diset
- `config.Load()` harus return `*Config`, bukan global variable

### ✅ Prinsip
```go
func Load() *Config {
    viper.SetConfigFile(".env")
    viper.SetConfigType("env")
    viper.ReadInConfig()   // tidak perlu cek error — kalau .env tidak ada, fallback ke system env

    viper.AutomaticEnv()

    viper.SetDefault("APP_PORT", "8080")
    viper.SetDefault("APP_ENV", "development")
    // config sensitif tidak diberi default
    viper.SetDefault("JWT_SECRET", "")
    viper.SetDefault("DB_PASSWORD", "")

    return &Config{
        AppPort: viper.GetString("APP_PORT"),
        // ...
    }
}
```

### ❌ Anti-Pattern
```go
var cfg *Config // JANGAN: config sebagai global variable

func Load() {
    // JANGAN: tidak return *Config, tidak bisa di-inject ke dependency
}

viper.SetDefault("JWT_SECRET", "rahasia123") // JANGAN: hardcode secret sebagai default
```

---

## Rules Logger (`pkg/logger/logger.go`)

- `logger.Init()` dipanggil **sekali** di `main.go`, sebelum komponen lain
- Mode `development` → output console berwarna dan mudah dibaca manusia
- Mode `production` → output JSON untuk monitoring tools
- Setelah `Init()` dipanggil, semua package cukup import dan langsung pakai

### ✅ Prinsip
```go
// main.go
logger.Init(cfg.AppEnv) // panggil sekali di awal

// package lain — langsung pakai tanpa init lagi
logger.Info("Database connected")
logger.Errorf("repo.FindByID failed", err, map[string]any{"id": id})
```

### ❌ Anti-Pattern
```go
logger.Init("development") // JANGAN: hardcode env
logger.Init(cfg.AppEnv)    // JANGAN: dipanggil lebih dari sekali
```

---

## Rules Database (`internal/database/postgres.go`)

- `database.Connect()` harus **return `*gorm.DB`**, bukan simpan ke global variable
- `*gorm.DB` di-inject ke repository via constructor
- Jika koneksi gagal, gunakan `logger.Fatal()` — app tidak boleh jalan tanpa DB

### ✅ Prinsip
```go
// database.go — return *gorm.DB
func Connect(cfg *config.Config) *gorm.DB {
    db, err := gorm.Open(...)
    if err != nil {
        logger.Fatal("Failed to connect to database", err)
    }
    logger.Infof("Database connected", map[string]any{"host": cfg.DBHost})
    return db
}

// main.go — inject db ke repository
db := database.Connect(cfg)
userRepo := repository.NewUserRepository(db)
userService := service.NewUserService(userRepo)
userHandler := handler.NewUserHandler(userService)
```

### ❌ Anti-Pattern
```go
// Global variable DB — susah di-test dan tidak thread-safe
var DB *gorm.DB // JANGAN

func Connect(cfg *config.Config) {
    DB, _ = gorm.Open(...) // JANGAN: simpan ke global, tidak return
}
```

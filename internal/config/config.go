package config

import (
	"net/http"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppPort string
	AppEnv  string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration

	// CookieSecure = true di production (HTTPS). Di dev lokal (HTTP) = false.
	CookieSecure bool

	// CookieSameSite = mode SameSite untuk cookie refresh token.
	// "strict" aman kalau FE dan BE satu origin (lewat proxy Next.js).
	// "none" wajib kalau FE dan BE beda origin — dan "none" mensyaratkan Secure=true.
	CookieSameSite string

	// CORSAllowedOrigins = daftar origin frontend yang boleh akses API.
	// Kosong = CORS mati (cocok kalau FE mengakses lewat proxy same-origin).
	CORSAllowedOrigins []string

	// MinIO / S3 — penyimpanan gambar produk.
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool
	// URL publik untuk dipakai di <img>. Kosong = pakai MinioEndpoint.
	MinioPublicURL string

	// Seed admin default (opsional). Kalau salah satu kosong, seed di-skip.
	AdminEmail    string
	AdminPassword string
}

func Load() *Config {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	_ = viper.ReadInConfig()

	viper.AutomaticEnv()
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("ACCESS_TOKEN_TTL", "15m")
	viper.SetDefault("REFRESH_TOKEN_TTL", "168h") // 7 hari
	viper.SetDefault("COOKIE_SECURE", false)      // dev lokal HTTP; production set true
	viper.SetDefault("COOKIE_SAMESITE", "strict") // "strict" | "lax" | "none"
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "")  // kosong = CORS mati
	// Config sensitif TIDAK diberi default — biarkan kosong agar mudah dideteksi.
	viper.SetDefault("DB_PASSWORD", "")
	viper.SetDefault("JWT_SECRET", "")

	appEnv := viper.GetString("APP_ENV")

	cfg := &Config{
		AppPort: viper.GetString("APP_PORT"),
		AppEnv:  appEnv,

		DBHost:     viper.GetString("DB_HOST"),
		DBPort:     viper.GetString("DB_PORT"),
		DBUser:     viper.GetString("DB_USER"),
		DBPassword: viper.GetString("DB_PASSWORD"),
		DBName:     viper.GetString("DB_NAME"),
		DBSSLMode:  viper.GetString("DB_SSLMODE"),

		JWTSecret:    viper.GetString("JWT_SECRET"),
		AccessTTL:    viper.GetDuration("ACCESS_TOKEN_TTL"),
		RefreshTTL:   viper.GetDuration("REFRESH_TOKEN_TTL"),
		CookieSecure: viper.GetBool("COOKIE_SECURE"),

		CookieSameSite:     strings.ToLower(viper.GetString("COOKIE_SAMESITE")),
		CORSAllowedOrigins: parseOrigins(viper.GetString("CORS_ALLOWED_ORIGINS")),

		MinioEndpoint:  viper.GetString("MINIO_ENDPOINT"),
		MinioAccessKey: viper.GetString("MINIO_ACCESS_KEY"),
		MinioSecretKey: viper.GetString("MINIO_SECRET_KEY"),
		MinioBucket:    viper.GetString("MINIO_BUCKET"),
		MinioUseSSL:    viper.GetBool("MINIO_USE_SSL"),
		MinioPublicURL: strings.TrimSuffix(viper.GetString("MINIO_PUBLIC_URL"), "/"),

		AdminEmail:    viper.GetString("ADMIN_EMAIL"),
		AdminPassword: viper.GetString("ADMIN_PASSWORD"),
	}

	return cfg
}

// parseOrigins memecah daftar origin dipisah koma menjadi slice yang sudah bersih.
// Contoh: "http://localhost:3000, https://cms.example.com"
func parseOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			origins = append(origins, strings.TrimSuffix(p, "/"))
		}
	}
	return origins
}

// SameSiteMode menerjemahkan config string ke konstanta http.SameSite.
// Nilai tak dikenal jatuh ke Strict — pilihan paling aman.
func (c *Config) SameSiteMode() http.SameSite {
	switch c.CookieSameSite {
	case "none":
		return http.SameSiteNoneMode
	case "lax":
		return http.SameSiteLaxMode
	default:
		return http.SameSiteStrictMode
	}
}

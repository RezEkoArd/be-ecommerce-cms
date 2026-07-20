package config

import (
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
	}

	return cfg
}

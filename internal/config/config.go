package config

import "github.com/spf13/viper"


type Config struct {
	AppPort	string
	AppEnv 	string

	DBHost	string
	DBPort	string
	DBUser	string
	DBPassword	string
	DBName	string
	DBSSLMode	string

	JWTSecret	string
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
	// Config sensitif TIDAK diberi default — biarkan kosong agar mudah dideteksi.
	viper.SetDefault("DB_PASSWORD", "")
	viper.SetDefault("JWT_SECRET", "")
	
	return &Config{
		AppPort: viper.GetString("APP_PORT"),
		AppEnv:  viper.GetString("APP_ENV"),

		DBHost:     viper.GetString("DB_HOST"),
		DBPort:     viper.GetString("DB_PORT"),
		DBUser:     viper.GetString("DB_USER"),
		DBPassword: viper.GetString("DB_PASSWORD"),
		DBName:     viper.GetString("DB_NAME"),
		DBSSLMode:  viper.GetString("DB_SSLMODE"),

		JWTSecret: viper.GetString("JWT_SECRET"),
	}
}
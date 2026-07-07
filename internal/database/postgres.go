package database

import (
	"fmt"

	"github.com/rezekoard/be-cms-ecommerce/internal/config"
	"github.com/rezekoard/be-cms-ecommerce/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Fatal("Failed to connect to database", err)
	}

	logger.Infof("Database connected", map[string]any{
		"host": cfg.DBHost,
		"name": cfg.DBName,
	})

	return  db
}
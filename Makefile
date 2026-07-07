# Muat variabel dari .env agar bisa dipakai di perintah migrate
include .env
export

# URL koneksi untuk golang-migrate
DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

# Jalankan semua migration yang belum diterapkan
migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

# Rollback satu langkah terakhir
migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1

# Buat pasangan file migration baru. Contoh: make migrate-create name=create_users
migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

# Jalankan aplikasi
run:
	go run cmd/main.go

.PHONY: migrate-up migrate-down migrate-create run
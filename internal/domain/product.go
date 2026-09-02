package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Category struct {
	ID   uuid.UUID
	Name string
	Slug string
	// ImageURL = gambar sampul kategori (URL publik MinIO). Boleh kosong.
	ImageURL  string
	CreatedAt time.Time

	// ProductCount diisi saat list kategori — jumlah produk di kategori ini.
	ProductCount int64
}

type Product struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	Description string
	Price       decimal.Decimal
	Stock       int
	CategoryID  *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// Relasi diisi saat query detail produk
	Category *Category
	Images   []ProductImage
}

type ProductImage struct {
	ID        uuid.UUID
	ProductID uuid.UUID
	URL       string
	IsPrimary bool
	CreatedAt time.Time
}

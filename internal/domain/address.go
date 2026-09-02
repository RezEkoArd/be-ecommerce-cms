package domain

import (
	"time"

	"github.com/google/uuid"
)

// Address = alamat pengiriman milik user. Satu user boleh punya banyak,
// tapi hanya satu yang IsPrimary (dijaga unique index parsial di DB).
type Address struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Label      string // "Rumah", "Kantor", dst.
	Recipient  string
	Phone      string
	Street     string
	City       string
	PostalCode string
	IsPrimary  bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

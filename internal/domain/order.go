package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderStatus string

const (
	OrderStatusDraft     OrderStatus = "draft"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type DiscountType string

const (
	DiscountPercent DiscountType = "percent"
	DiscountFixed   DiscountType = "fixed"
)

type Coupon struct {
	ID            uuid.UUID
	Code          string
	DiscountType  DiscountType
	DiscountValue decimal.Decimal
	ExpiresAt     *time.Time
	IsActive      bool
}

type Order struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CouponID  *uuid.UUID
	Status    OrderStatus
	Subtotal  decimal.Decimal
	Tax       decimal.Decimal
	Discount  decimal.Decimal
	Total     decimal.Decimal
	Items     []OrderItem
	CreatedAt time.Time
	UpdatedAt time.Time

	// Relasi diisi saat query detail order (admin).
	User   *OrderUser
	Coupon *Coupon
}

// OrderUser = ringkasan pemesan. Sengaja bukan *User penuh agar
// PasswordHash tidak ikut ter-serialize ke response.
type OrderUser struct {
	ID    uuid.UUID
	Name  string
	Email string
}

type OrderItem struct {
	ID          uuid.UUID
	OrderID     uuid.UUID
	ProductID   *uuid.UUID
	ProductName string
	Price       decimal.Decimal
	Quantity    int
}

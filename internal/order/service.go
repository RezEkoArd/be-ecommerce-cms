package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
	"github.com/shopspring/decimal"
)

// CartReader = kebutuhan order terhadap cart, didefinisikan di sisi konsumen
// (bukan import package cart). cart.Service kebetulan memenuhi ini.
type CartReader interface {
	GetCart(ctx context.Context, userID uuid.UUID) (*domain.Cart, error)
}

// AddressReader = kebutuhan order terhadap alamat, didefinisikan di sisi
// konsumen. address.Service kebetulan memenuhi ini.
type AddressReader interface {
	Get(ctx context.Context, userID, id uuid.UUID) (*domain.Address, error)
}

type CreateCouponInput struct {
	Code          string
	DiscountType  domain.DiscountType
	DiscountValue decimal.Decimal
	ExpiresAt     *time.Time
	IsActive      bool
}

// UpdateCouponInput — semua field pointer supaya bisa update sebagian.
// nil berarti "jangan ubah", jadi toggle aktif cukup mengisi IsActive saja.
type UpdateCouponInput struct {
	Code          *string
	DiscountType  *domain.DiscountType
	DiscountValue *decimal.Decimal
	ExpiresAt     **time.Time // pointer ganda: bedakan "tidak diubah" dari "dikosongkan"
	IsActive      *bool
}

type Service interface {
	CreateCoupon(ctx context.Context, in CreateCouponInput) (*domain.Coupon, error)
	ListCoupons(ctx context.Context) ([]domain.Coupon, error)
	GetCoupon(ctx context.Context, id uuid.UUID) (*domain.Coupon, error)
	UpdateCoupon(ctx context.Context, id uuid.UUID, in UpdateCouponInput) (*domain.Coupon, error)
	DeleteCoupon(ctx context.Context, id uuid.UUID) error

	Checkout(ctx context.Context, userID uuid.UUID, in CheckoutInput) (*domain.Order, error)
	ListMyOrders(ctx context.Context, userID uuid.UUID) ([]domain.Order, error)
	GetOrder(ctx context.Context, userID, orderID uuid.UUID) (*domain.Order, error)

	// Admin — lintas user.
	ListAllOrders(ctx context.Context) ([]domain.Order, error)
	GetOrderDetail(ctx context.Context, orderID uuid.UUID) (*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status domain.OrderStatus) error
}

// CheckoutInput = pilihan user saat checkout.
type CheckoutInput struct {
	AddressID  uuid.UUID
	CouponCode string
}

type service struct {
	repo    Repository
	cart    CartReader
	address AddressReader
}

func NewService(repo Repository, cart CartReader, address AddressReader) Service {
	return &service{repo: repo, cart: cart, address: address}
}

func (s *service) CreateCoupon(ctx context.Context, in CreateCouponInput) (*domain.Coupon, error) {
	// Kode kupon unik — cek dulu agar duplikat jadi 409, bukan 500 dari DB.
	existing, err := s.repo.FindCouponByCode(ctx, in.Code)
	if err != nil && !errors.Is(err, domain.ErrCouponNotFound) {
		return nil, fmt.Errorf("orderService.CreateCoupon: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrCouponAlreadyExists
	}

	// Diskon persen tidak boleh melebihi 100.
	if in.DiscountType == domain.DiscountPercent &&
		in.DiscountValue.GreaterThan(decimal.NewFromInt(100)) {
		return nil, domain.ErrDiscountTooLarge
	}

	c := &domain.Coupon{
		Code:          in.Code,
		DiscountType:  in.DiscountType,
		DiscountValue: in.DiscountValue,
		ExpiresAt:     in.ExpiresAt,
		IsActive:      in.IsActive,
	}
	if err := s.repo.CreateCoupon(ctx, c); err != nil {
		return nil, fmt.Errorf("orderService.CreateCoupon: %w", err)
	}
	return c, nil
}

func (s *service) ListCoupons(ctx context.Context) ([]domain.Coupon, error) {
	coupons, err := s.repo.ListCoupons(ctx)
	if err != nil {
		return nil, fmt.Errorf("orderService.ListCoupons: %w", err)
	}
	return coupons, err
}

func (s *service) GetCoupon(ctx context.Context, id uuid.UUID) (*domain.Coupon, error) {
	c, err := s.repo.FindCouponByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("orderService.GetCoupon: %w", err)
	}
	return c, nil
}

func (s *service) UpdateCoupon(ctx context.Context, id uuid.UUID, in UpdateCouponInput) (*domain.Coupon, error) {
	c, err := s.repo.FindCouponByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("orderService.UpdateCoupon: %w", err)
	}

	// Kode berubah → pastikan tidak bentrok dengan kupon lain.
	if in.Code != nil && *in.Code != c.Code {
		existing, err := s.repo.FindCouponByCode(ctx, *in.Code)
		if err != nil && !errors.Is(err, domain.ErrCouponNotFound) {
			return nil, fmt.Errorf("orderService.UpdateCoupon: %w", err)
		}
		if existing != nil && existing.ID != id {
			return nil, domain.ErrCouponAlreadyExists
		}
		c.Code = *in.Code
	}

	if in.DiscountType != nil {
		c.DiscountType = *in.DiscountType
	}
	if in.DiscountValue != nil {
		c.DiscountValue = *in.DiscountValue
	}
	if in.ExpiresAt != nil {
		c.ExpiresAt = *in.ExpiresAt
	}
	if in.IsActive != nil {
		c.IsActive = *in.IsActive
	}

	// Diskon persen tidak boleh melebihi 100.
	if c.DiscountType == domain.DiscountPercent &&
		c.DiscountValue.GreaterThan(decimal.NewFromInt(100)) {
		return nil, domain.ErrDiscountTooLarge
	}

	if err := s.repo.UpdateCoupon(ctx, c); err != nil {
		return nil, fmt.Errorf("orderService.UpdateCoupon: %w", err)
	}
	return c, nil
}

func (s *service) DeleteCoupon(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteCoupon(ctx, id); err != nil {
		return fmt.Errorf("orderService.DeleteCoupon: %w", err)
	}
	return nil
}

func (s *service) Checkout(ctx context.Context, userID uuid.UUID, in CheckoutInput) (*domain.Order, error) {
	// 0. Alamat pengiriman wajib — barang harus punya tujuan.
	//    Get() menyaring per user, jadi alamat orang lain otomatis ditolak.
	addr, err := s.address.Get(ctx, userID, in.AddressID)
	if err != nil {
		return nil, fmt.Errorf("orderService.Checkout: %w", err)
	}

	// 1. Ambil cart user (GetCart auto-create, jadi selalu ada objeknya).
	cart, err := s.cart.GetCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("orderService.Checkout: %w", err)
	}
	if len(cart.Items) == 0 {
		return nil, domain.ErrCartEmpty
	}

	// 2. Snapshot tiap item + hitung subtotal.
	subtotal := decimal.Zero
	items := make([]domain.OrderItem, 0, len(cart.Items))
	for _, ci := range cart.Items {
		if ci.Product == nil {
			// Produk hilang di tengah jalan — batalkan.
			return nil, domain.ErrProductNotFound
		}
		lineTotal := ci.Product.Price.Mul(decimal.NewFromInt(int64(ci.Quantity)))
		subtotal = subtotal.Add(lineTotal)

		productID := ci.ProductID // salin ke var lokal untuk ambil alamatnya
		items = append(items, domain.OrderItem{
			ProductID:   &productID,
			ProductName: ci.Product.Name,  // snapshot nama
			Price:       ci.Product.Price, // snapshot harga
			Quantity:    ci.Quantity,
		})
	}

	// 3. Kupon (opsional). Kalau ada kode, validasi & hitung diskon.
	var couponID *uuid.UUID
	discount := decimal.Zero
	if in.CouponCode != "" {
		coupon, err := s.repo.FindCouponByCode(ctx, in.CouponCode)
		if err != nil {
			return nil, fmt.Errorf("orderService.Checkout: %w", err)
		}
		if err := validateCoupon(coupon); err != nil {
			return nil, err
		}
		discount = calcDiscount(coupon, subtotal)
		couponID = &coupon.ID
	}

	tax := decimal.Zero
	total := subtotal.Add(tax).Sub(discount)
	if total.IsNegative() {
		total = decimal.Zero
	}

	o := &domain.Order{
		UserID:   userID,
		CouponID: couponID,
		Status:   domain.OrderStatusPaid, // langsung 'paid' untuk MVP
		Subtotal: subtotal,
		Tax:      tax,
		Discount: discount,
		Total:    total,
		Items:    items,
		Shipping: domain.ShippingAddress{
			Recipient:  addr.Recipient,
			Phone:      addr.Phone,
			Street:     addr.Street,
			City:       addr.City,
			PostalCode: addr.PostalCode,
		},
	}
	if err := s.repo.CreateOrder(ctx, o, cart.ID); err != nil {
		return nil, fmt.Errorf("orderService.Checkout: %w", err)
	}
	return o, nil
}

func (s *service) ListMyOrders(ctx context.Context, userID uuid.UUID) ([]domain.Order, error) {
	orders, err := s.repo.ListOrdersByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("orderService.ListMyOrder: %w", err)
	}
	return orders, nil
}

// GetOrderDetail — admin: detail order lengkap dengan user & kupon,
// tanpa pengecekan kepemilikan seperti GetOrder milik customer.
func (s *service) GetOrderDetail(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	o, err := s.repo.FindOrderDetailByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("orderService.GetOrderDetail: %w", err)
	}
	return o, nil
}

func (s *service) ListAllOrders(ctx context.Context) ([]domain.Order, error) {
	orders, err := s.repo.ListAllOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("orderService.ListAllOrders: %w", err)
	}
	return orders, nil
}

// UpdateOrderStatus memvalidasi status sebelum menyimpan — hanya nilai
// yang dikenal domain yang boleh masuk DB.
func (s *service) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status domain.OrderStatus) error {
	if !isValidOrderStatus(status) {
		return domain.ErrInvalidOrderStatus
	}

	if err := s.repo.UpdateOrderStatus(ctx, orderID, status); err != nil {
		return fmt.Errorf("orderService.UpdateOrderStatus: %w", err)
	}
	return nil
}

func isValidOrderStatus(s domain.OrderStatus) bool {
	switch s {
	case domain.OrderStatusDraft, domain.OrderStatusPaid, domain.OrderStatusShipped,
		domain.OrderStatusCompleted, domain.OrderStatusCancelled:
		return true
	default:
		return false
	}
}

func (s *service) GetOrder(ctx context.Context, userID, orderID uuid.UUID) (*domain.Order, error) {
	o, err := s.repo.FindOrderByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("orderService.GetOrder: %w", err)
	}
	if o.UserID != userID {
		return nil, domain.ErrOrderNotFound
	}
	return o, nil
}

// helper
func validateCoupon(c *domain.Coupon) error {
	if !c.IsActive {
		return domain.ErrCouponInvalid
	}
	if c.ExpiresAt != nil && time.Now().After(*c.ExpiresAt) {
		return domain.ErrCouponInvalid
	}
	return nil
}

func calcDiscount(c *domain.Coupon, subtotal decimal.Decimal) decimal.Decimal {
	switch c.DiscountType {
	case domain.DiscountPercent:
		return subtotal.Mul(c.DiscountValue).Div(decimal.NewFromInt(100))
	case domain.DiscountFixed:
		return c.DiscountValue
	default:
		return decimal.Zero
	}
}

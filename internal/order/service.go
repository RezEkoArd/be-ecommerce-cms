package order

import (
	"context"
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

type CreateCouponInput struct {
	Code			string
	DiscountType	domain.DiscountType
	DiscountValue	decimal.Decimal
	ExpiresAt		*time.Time
	IsActive		bool
}

type Service interface {
	CreateCoupon(ctx context.Context, in CreateCouponInput) (*domain.Coupon, error)
	ListCoupons(ctx context.Context) ([]domain.Coupon, error)

	Checkout(ctx context.Context, userID uuid.UUID, couponCode string) (*domain.Order, error)
	ListMyOrders(ctx context.Context, userID uuid.UUID) ([]domain.Order, error)
	GetOrder(ctx context.Context, userID, orderID uuid.UUID) (*domain.Order, error)
}

type service struct {
	repo Repository
	cart CartReader
}

func NewService(repo Repository, cart CartReader) Service {
	return &service{repo: repo, cart: cart}
}

func (s *service) CreateCoupon(ctx context.Context, in CreateCouponInput) (*domain.Coupon, error) {
	c := &domain.Coupon {
		Code: in.Code,
		DiscountType: in.DiscountType,
		DiscountValue: in.DiscountValue,
		ExpiresAt: in.ExpiresAt,
		IsActive: in.IsActive,
	}
	if err := s.repo.CreateCoupon(ctx, c);err != nil {
		return nil, fmt.Errorf("orderService.CreateCoupon: %w", err)
	}
	return c,nil
}

func (s *service) ListCoupons(ctx context.Context) ([]domain.Coupon, error) {
	coupons, err := s.repo.ListCoupons(ctx)
	if err != nil {
		return nil, fmt.Errorf("orderService.ListCoupons: %w", err)
	}
	return  coupons, err
}


func (s *service) Checkout(ctx context.Context, userID uuid.UUID, couponCode string) (*domain.Order, error) {
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
			ProductName: ci.Product.Name, // snapshot nama
			Price:       ci.Product.Price, // snapshot harga
			Quantity:    ci.Quantity,
		})
	}

	// 3. Kupon (opsional). Kalau ada kode, validasi & hitung diskon.
	var couponID *uuid.UUID
	discount := decimal.Zero
	if couponCode != "" {
		coupon, err := s.repo.FindCouponByCode(ctx, couponCode)
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
	}
	if err := s.repo.CreateOrder(ctx, o, cart.ID); err != nil {
		return nil, fmt.Errorf("orderService.Checkout: %w", err)
	}
	return o, nil
 }

 func (s *service) ListMyOrders(ctx context.Context, userID uuid.UUID) ([]domain.Order, error) {
	orders, err := s.repo.ListOrdersByUser(ctx ,userID)
	if err != nil {
		return nil, fmt.Errorf("orderService.ListMyOrder: %w", err)
	}
	return orders, nil
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



//  helper 
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
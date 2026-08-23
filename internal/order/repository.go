package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
	"github.com/rezekoard/be-cms-ecommerce/pkg/logger"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Repository interface {
	CreateCoupon(ctx context.Context, c *domain.Coupon) error
	FindCouponByCode(ctx context.Context, code string) (*domain.Coupon, error)
	FindCouponByID(ctx context.Context, id uuid.UUID) (*domain.Coupon, error)
	ListCoupons(ctx context.Context) ([]domain.Coupon, error)
	UpdateCoupon(ctx context.Context, c *domain.Coupon) error
	DeleteCoupon(ctx context.Context, id uuid.UUID) error

	CreateOrder(ctx context.Context, o *domain.Order, cartID uuid.UUID) error
	FindOrderByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	ListOrdersByUser(ctx context.Context, userID uuid.UUID) ([]domain.Order, error)

	// Admin — tanpa filter user_id.
	ListAllOrders(ctx context.Context) ([]domain.Order, error)
	FindOrderDetailByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

type couponModel struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Code          string
	DiscountType  string
	DiscountValue decimal.Decimal `gorm:"type:numeric(12,2)"`
	ExpiresAt     *time.Time
	IsActive      bool
}

func (couponModel) TableName() string { return "coupons" }

type orderModel struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID  `gorm:"type:uuid"`
	CouponID  *uuid.UUID `gorm:"type:uuid"`
	Status    string
	Subtotal  decimal.Decimal `gorm:"type:numeric(12,2)"`
	Tax       decimal.Decimal `gorm:"type:numeric(12,2)"`
	Discount  decimal.Decimal `gorm:"type:numberic(12,2)"`
	Total     decimal.Decimal `gorm:"type:numeric(12.2)"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (orderModel) TableName() string { return "orders" }

type orderItemModel struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	OrderID     uuid.UUID  `gorm:"type:uuid"`
	ProductID   *uuid.UUID `gorm:"type:uuid"`
	ProductName string
	Price       decimal.Decimal `gorm:"type:numeric(12,2)"`
	Quantity    int
}

func (orderItemModel) TableName() string { return "order_items" }

func couponToDomain(m couponModel) domain.Coupon {
	return domain.Coupon{
		ID: m.ID, Code: m.Code,
		DiscountType:  domain.DiscountType(m.DiscountType),
		DiscountValue: m.DiscountValue,
		ExpiresAt:     m.ExpiresAt,
		IsActive:      m.IsActive,
	}
}

func (r *repository) CreateCoupon(ctx context.Context, c *domain.Coupon) error {
	m := &couponModel{
		Code: c.Code, DiscountType: string(c.DiscountType),
		DiscountValue: c.DiscountValue, ExpiresAt: c.ExpiresAt, IsActive: c.IsActive,
	}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		logger.Errorf("orderRepository.CreateCoupon failed", err, map[string]any{"code": c.Code})
		return fmt.Errorf("orderRepository.CreateCoupon: %w", err)
	}
	c.ID = m.ID
	return nil
}

func (r *repository) FindCouponByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	var m couponModel
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCouponNotFound
		}
		logger.Errorf("orderRepository.FindCouponByCode failed", err, map[string]any{"code": code})
		return nil, fmt.Errorf("orderRepository.FindCouponByCode: %w", err)
	}
	c := couponToDomain(m)
	return &c, nil
}

func (r *repository) FindCouponByID(ctx context.Context, id uuid.UUID) (*domain.Coupon, error) {
	var m couponModel
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCouponNotFound
		}
		logger.Errorf("orderRepository.FindCouponByID failed", err, map[string]any{"coupon_id": id})
		return nil, fmt.Errorf("orderRepository.FindCouponByID: %w", err)
	}
	c := couponToDomain(m)
	return &c, nil
}

func (r *repository) ListCoupons(ctx context.Context) ([]domain.Coupon, error) {
	var ms []couponModel
	if err := r.db.WithContext(ctx).Order("code ASC").Find(&ms).Error; err != nil {
		logger.Errorf("orderRepository.ListCoupons failed", err, nil)
		return nil, fmt.Errorf("orderRepository.ListCoupons: %w", err)
	}
	out := make([]domain.Coupon, 0, len(ms))
	for _, m := range ms {
		out = append(out, couponToDomain(m))
	}
	return out, nil
}

func (r *repository) UpdateCoupon(ctx context.Context, c *domain.Coupon) error {
	err := r.db.WithContext(ctx).Model(&couponModel{}).
		Where("id = ?", c.ID).
		Updates(map[string]any{
			"code":           c.Code,
			"discount_type":  string(c.DiscountType),
			"discount_value": c.DiscountValue,
			"expires_at":     c.ExpiresAt,
			"is_active":      c.IsActive,
		}).Error
	if err != nil {
		logger.Errorf("orderRepository.UpdateCoupons failed", err, map[string]any{"coupon_id": c.ID})
		return fmt.Errorf("orderRepository.UpdateCoupon: %w", err)
	}
	return nil
}

func (r *repository) DeleteCoupon(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&couponModel{}, "id = ?", id)
	if res.Error != nil {
		logger.Errorf("orderRepository.DeleteCoupon failed", res.Error, map[string]any{"coupon_id": id})
		return fmt.Errorf("orderRepository.DeleteCoupon: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrCouponNotFound
	}
	return nil
}

// create order
func (r *repository) CreateOrder(ctx context.Context, o *domain.Order, cartID uuid.UUID) error {
	// Semua operasi dibungkus 1 transaksi. Kalau ada yang return error,
	// GORM otomatis ROLLBACK. Kalau fungsi return nil, otomatis COMMIT.

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Insert order (header)
		om := &orderModel{
			UserID:   o.UserID,
			CouponID: o.CouponID,
			Status:   string(o.Status),
			Subtotal: o.Subtotal,
			Tax:      o.Tax,
			Discount: o.Discount,
			Total:    o.Total,
		}
		if err := tx.Create(om).Error; err != nil {
			return fmt.Errorf("insert order: %w", err)
		}
		o.ID = om.ID
		o.CreatedAt = om.CreatedAt
		o.UpdatedAt = om.UpdatedAt

		// 2. Insert tiap order_item (snapshot) + kurangi stok produk.
		for i := range o.Items {
			it := &o.Items[i]
			im := &orderItemModel{
				OrderID:     om.ID,
				ProductID:   it.ProductID,
				ProductName: it.ProductName,
				Price:       it.Price,
				Quantity:    it.Quantity,
			}
			if err := tx.Create(im).Error; err != nil {
				return fmt.Errorf("insert order_item: %w", err)
			}
			it.ID = im.ID
			it.OrderID = om.ID

			// Kurangi stok. Guard "stock >= qty" mencegah stok minus kalau ada
			// race — kalau tidak ada baris ter-update, berarti stok tak cukup.
			if it.ProductID != nil {
				res := tx.Exec(
					"UPDATE products SET stock = stock - ? WHERE id = ? AND stock >= ?",
					it.Quantity, *it.ProductID, it.Quantity,
				)
				if res.Error != nil {
					return fmt.Errorf("kurangi stok: %w", res.Error)
				}
				if res.RowsAffected == 0 {
					return domain.ErrInsufficientStock
				}
			}
		}

		if err := tx.Exec("DELETE FROM cart_items WHERE cart_id = ?", cartID).Error; err != nil {
			return fmt.Errorf("clear cart: %w", err)
		}

		return nil
	})

	if err != nil {
		logger.Errorf("orderRepository.CreateOrder failed", err, map[string]any{"user_id": o.UserID})
		return fmt.Errorf("orderRepository.CreateOrder: %w", err)
	}

	return nil
}

func (r *repository) FindOrderByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	var m orderModel
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrderNotFound
		}
		logger.Errorf("orderRepository.FindOrderByID failed", err, map[string]any{"order_id": id})
		return nil, fmt.Errorf("orderRepository.FindOrderByID: %w", err)
	}

	o := orderToDomain(m)

	var items []orderItemModel
	if err := r.db.WithContext(ctx).Where("order_id = ?", m.ID).Find(&items).Error; err != nil {
		logger.Errorf("orderRepository.FindOrderByID items failed", err, map[string]any{"order_id": id})
		return nil, fmt.Errorf("orderRepository.FindOrderByID: %w", err)
	}
	o.Items = make([]domain.OrderItem, 0, len(items))
	for _, im := range items {
		o.Items = append(o.Items, domain.OrderItem{
			ID: im.ID, OrderID: im.OrderID, ProductID: im.ProductID,
			ProductName: im.ProductName, Price: im.Price, Quantity: im.Quantity,
		})
	}
	return &o, nil
}

// userSummaryModel = kolom user yang aman dikirim ke admin.
// Sengaja tidak memakai model user penuh agar password_hash tidak ikut terbaca.
type userSummaryModel struct {
	ID    uuid.UUID
	Name  string
	Email string
}

// FindOrderDetailByID = FindOrderByID + relasi user & kupon, untuk modal detail admin.
func (r *repository) FindOrderDetailByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	o, err := r.FindOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Pemesan. Kalau gagal dibaca, detail order tetap dikembalikan
	// tanpa data user — relasi bukan alasan menggagalkan seluruh response.
	var u userSummaryModel
	err = r.db.WithContext(ctx).
		Table("users").
		Select("id", "name", "email").
		Where("id = ?", o.UserID).
		Take(&u).Error
	if err == nil {
		o.User = &domain.OrderUser{ID: u.ID, Name: u.Name, Email: u.Email}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Errorf("orderRepository.FindOrderDetailByID user failed", err, map[string]any{"order_id": id})
	}

	// Kupon (opsional — order bisa tanpa kupon).
	if o.CouponID != nil {
		coupon, err := r.FindCouponByID(ctx, *o.CouponID)
		if err == nil {
			o.Coupon = coupon
		} else if !errors.Is(err, domain.ErrCouponNotFound) {
			logger.Errorf("orderRepository.FindOrderDetailByID coupon failed", err, map[string]any{"order_id": id})
		}
	}

	return o, nil
}

func (r *repository) ListOrdersByUser(ctx context.Context, userID uuid.UUID) ([]domain.Order, error) {
	var ms []orderModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&ms).Error; err != nil {
		logger.Errorf("orderRepository.ListOrderByUser failed", err, map[string]any{"user_id": userID})
		return nil, fmt.Errorf("orderRepository.ListOrdersByUser: %w", err)
	}
	out := make([]domain.Order, 0, len(ms))
	for _, m := range ms {
		out = append(out, orderToDomain(m))
	}
	return out, nil
}

// ListAllOrders mengambil seluruh order tanpa filter user — khusus admin.
func (r *repository) ListAllOrders(ctx context.Context) ([]domain.Order, error) {
	var ms []orderModel
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&ms).Error; err != nil {
		logger.Errorf("orderRepository.ListAllOrders failed", err, nil)
		return nil, fmt.Errorf("orderRepository.ListAllOrders: %w", err)
	}
	out := make([]domain.Order, 0, len(ms))
	for _, m := range ms {
		out = append(out, orderToDomain(m))
	}
	return out, nil
}

// UpdateOrderStatus mengubah status satu order.
func (r *repository) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
	res := r.db.WithContext(ctx).
		Model(&orderModel{}).
		Where("id = ?", id).
		Update("status", string(status))

	if res.Error != nil {
		logger.Errorf("orderRepository.UpdateOrderStatus failed", res.Error, map[string]any{
			"order_id": id, "status": status,
		})
		return fmt.Errorf("orderRepository.UpdateOrderStatus: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrOrderNotFound
	}
	return nil
}

// Helper
func orderToDomain(m orderModel) domain.Order {
	return domain.Order{
		ID: m.ID, UserID: m.UserID, CouponID: m.CouponID,
		Status:   domain.OrderStatus(m.Status),
		Subtotal: m.Subtotal, Tax: m.Tax, Discount: m.Discount, Total: m.Total,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

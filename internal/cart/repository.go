package cart

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
	GetOrCreateCart(ctx context.Context, userID uuid.UUID) (*domain.Cart, error)
	FindCartByUserID(ctx context.Context, userID uuid.UUID) (*domain.Cart, error)
	FindItem(ctx context.Context, cartID, productID uuid.UUID) (*domain.CartItem, error)
	UpsertItem(ctx context.Context, item *domain.CartItem) error
	UpdateItemQuantity(ctx context.Context, cartID, productID uuid.UUID, quantity int) error
	DeleteItem(ctx context.Context, cartID, productID uuid.UUID) error
	FindProduct(ctx context.Context, id uuid.UUID) (*domain.Product, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

type cartModel struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (cartModel) TableName() string { return "carts" }

type cartItemModel struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	CartID    uuid.UUID `gorm:"type:uuid"`
	ProductID uuid.UUID `gorm:"type:uuid"`
	Quantity  int
	CreatedAt time.Time
}

func (cartItemModel) TableName() string { return "cart_items" }

func (r *repository) GetOrCreateCart(ctx context.Context, userID uuid.UUID) (*domain.Cart, error) {
	var m cartModel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error
	if err == nil {
		return &domain.Cart{ID: m.ID, UserID: m.UserID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Errorf("cartRepository.GetOrCreateCart find failed", err, map[string]any{"user_id": userID})
		return nil, fmt.Errorf("cartRepository.GetOrCreatedCart: %w", err)
	}
	m = cartModel{UserID: userID}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		logger.Errorf("cartRepository.GetOrCreateCart create failed", err, map[string]any{"user_id": userID})
		return nil, fmt.Errorf("cartRepository.GetOrCreateCart: %w", err)
	}
	return &domain.Cart{ID: m.ID, UserID: m.UserID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}, nil
}

func (r *repository) FindCartByUserID(ctx context.Context, userID uuid.UUID) (*domain.Cart, error) {
	var m cartModel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCartNotFound
		}
		logger.Errorf("cartRepository.FindCartByUserID failed", err, map[string]any{"user_id": userID})
		return nil, fmt.Errorf("cartRepository.FindCartByUserID: %w", err)
	}

	cart := &domain.Cart{ID: m.ID, UserID: m.UserID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}

	// Ambil semua item cart ini.
	var items []cartItemModel
	if err := r.db.WithContext(ctx).
		Where("cart_id = ?", m.ID).
		Order("created_at ASC").
		Find(&items).Error; err != nil {
		logger.Errorf("cartRepository.FindCartByUserID items failed", err, map[string]any{"cart_id": m.ID})
		return nil, fmt.Errorf("cartRepository.FindCartByUserID: %w", err)
	}

	cart.Items = make([]domain.CartItem, 0, len(items))
	for _, it := range items {
		ci := domain.CartItem{
			ID: it.ID, CartID: it.CartID, ProductID: it.ProductID,
			Quantity: it.Quantity, CreatedAt: it.CreatedAt,
		}
		// Isi info produk (nama/harga/stok) untuk item ini.
		if p, err := r.FindProduct(ctx, it.ProductID); err == nil {
			ci.Product = p
		}
		cart.Items = append(cart.Items, ci)
	}
	return cart, nil
}

func (r *repository) FindItem(ctx context.Context, cartID, productID uuid.UUID) (*domain.CartItem, error) {
	var it cartItemModel
	err := r.db.WithContext(ctx).
		Where("cart_id = ? AND product_id = ?", cartID, productID).
		First(&it).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCartItemNotFound
		}
		logger.Errorf("cartRepository.FindItem failed", err, map[string]any{"cart_id": cartID, "product_id": productID})
		return nil, fmt.Errorf("cartRepository.FindItem: %w", err)
	}
	return &domain.CartItem{
		ID: it.ID, CartID: it.CartID, ProductID: it.ProductID,
		Quantity: it.Quantity, CreatedAt: it.CreatedAt,
	}, nil
}

func (r *repository) UpsertItem(ctx context.Context, item *domain.CartItem) error {
	m := &cartItemModel{CartID: item.CartID, ProductID: item.ProductID, Quantity: item.Quantity}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		logger.Errorf("cartRepository.UpsertItem failed", err, map[string]any{"cart_id": item.CartID, "product_id": item.ProductID})
		return fmt.Errorf("cartRepository.UpsertItem: %w", err)
	}
	item.ID = m.ID
	item.CreatedAt = m.CreatedAt
	return nil
}

func (r *repository) UpdateItemQuantity(ctx context.Context, cartID, productID uuid.UUID, quantity int) error {
	res := r.db.WithContext(ctx).Model(&cartItemModel{}).
		Where("cart_id = ? AND product_id = ?", cartID, productID).
		Update("quantity", quantity)

	if res.Error != nil {
		logger.Errorf("cartRepository.UpdateItemQuantity failed", res.Error, map[string]any{"cart_id": cartID, "product_id": productID})
		return fmt.Errorf("cartRepository.UpdateItemQuantity: %w", res.Error)
	}

	if res.RowsAffected == 0 {
		return domain.ErrCartItemNotFound
	}
	return nil
}

func (r *repository) DeleteItem(ctx context.Context, cartID, productID uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("cart_id = ? AND product_id = ?", cartID, productID).
		Delete(&cartItemModel{})
	if res.Error != nil {
		logger.Errorf("cartRepository.DeleteItem failed", res.Error, map[string]any{"cart_id": cartID, "product_id": productID})
		return fmt.Errorf("cartRepository.DeleteItem: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrCartItemNotFound
	}
	return nil
}

// Helper
type productRow struct {
	ID    uuid.UUID
	Name  string
	Price decimal.Decimal
	Stock int
}

func (productRow) TableName() string { return "products" }

func (r *repository) FindProduct(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	var row productRow
	if err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("cartRepository.findProduct: %w", err)
	}
	return &domain.Product{ID: row.ID, Name: row.Name, Price: row.Price, Stock: row.Stock}, nil
}

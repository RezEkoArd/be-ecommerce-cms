package address

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
	"github.com/rezekoard/be-cms-ecommerce/pkg/logger"
)

type Repository interface {
	Create(ctx context.Context, a *domain.Address) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Address, error)
	FindByID(ctx context.Context, userID, id uuid.UUID) (*domain.Address, error)
	Update(ctx context.Context, a *domain.Address) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
	CountByUser(ctx context.Context, userID uuid.UUID) (int64, error)
	// ClearPrimary melepas status utama dari semua alamat user.
	ClearPrimary(ctx context.Context, userID uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

type addressModel struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID     uuid.UUID `gorm:"type:uuid"`
	Label      string
	Recipient  string
	Phone      string
	Street     string
	City       string
	PostalCode string
	IsPrimary  bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (addressModel) TableName() string { return "addresses" }

func (m *addressModel) toDomain() domain.Address {
	return domain.Address{
		ID: m.ID, UserID: m.UserID, Label: m.Label,
		Recipient: m.Recipient, Phone: m.Phone, Street: m.Street,
		City: m.City, PostalCode: m.PostalCode, IsPrimary: m.IsPrimary,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func (r *repository) Create(ctx context.Context, a *domain.Address) error {
	m := &addressModel{
		UserID: a.UserID, Label: a.Label, Recipient: a.Recipient,
		Phone: a.Phone, Street: a.Street, City: a.City,
		PostalCode: a.PostalCode, IsPrimary: a.IsPrimary,
	}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		logger.Errorf("addressRepository.Create failed", err, map[string]any{"user_id": a.UserID})
		return fmt.Errorf("addressRepository.Create: %w", err)
	}
	a.ID = m.ID
	a.CreatedAt = m.CreatedAt
	a.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Address, error) {
	var ms []addressModel
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		// Alamat utama selalu di atas.
		Order("is_primary DESC, created_at ASC").
		Find(&ms).Error
	if err != nil {
		logger.Errorf("addressRepository.ListByUser failed", err, map[string]any{"user_id": userID})
		return nil, fmt.Errorf("addressRepository.ListByUser: %w", err)
	}

	out := make([]domain.Address, 0, len(ms))
	for i := range ms {
		out = append(out, ms[i].toDomain())
	}
	return out, nil
}

// FindByID menyaring per user sekaligus — mencegah user membaca alamat orang lain.
func (r *repository) FindByID(ctx context.Context, userID, id uuid.UUID) (*domain.Address, error) {
	var m addressModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrAddressNotFound
		}
		logger.Errorf("addressRepository.FindByID failed", err, map[string]any{"address_id": id})
		return nil, fmt.Errorf("addressRepository.FindByID: %w", err)
	}
	a := m.toDomain()
	return &a, nil
}

func (r *repository) Update(ctx context.Context, a *domain.Address) error {
	res := r.db.WithContext(ctx).Model(&addressModel{}).
		Where("id = ? AND user_id = ?", a.ID, a.UserID).
		Updates(map[string]any{
			"label":       a.Label,
			"recipient":   a.Recipient,
			"phone":       a.Phone,
			"street":      a.Street,
			"city":        a.City,
			"postal_code": a.PostalCode,
			"is_primary":  a.IsPrimary,
			"updated_at":  time.Now(),
		})

	if res.Error != nil {
		logger.Errorf("addressRepository.Update failed", res.Error, map[string]any{"address_id": a.ID})
		return fmt.Errorf("addressRepository.Update: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrAddressNotFound
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&addressModel{})

	if res.Error != nil {
		logger.Errorf("addressRepository.Delete failed", res.Error, map[string]any{"address_id": id})
		return fmt.Errorf("addressRepository.Delete: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrAddressNotFound
	}
	return nil
}

func (r *repository) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&addressModel{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		logger.Errorf("addressRepository.CountByUser failed", err, map[string]any{"user_id": userID})
		return 0, fmt.Errorf("addressRepository.CountByUser: %w", err)
	}
	return count, nil
}

func (r *repository) ClearPrimary(ctx context.Context, userID uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&addressModel{}).
		Where("user_id = ? AND is_primary", userID).
		Update("is_primary", false).Error
	if err != nil {
		logger.Errorf("addressRepository.ClearPrimary failed", err, map[string]any{"user_id": userID})
		return fmt.Errorf("addressRepository.ClearPrimary: %w", err)
	}
	return nil
}

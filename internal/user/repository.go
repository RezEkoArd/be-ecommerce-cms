package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
	"github.com/rezekoard/be-cms-ecommerce/pkg/logger"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, u *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// userModel = representasi GORM tabel users. Dipisah dari domain.User
// agar tag GORM tidak bocor ke domain.
type userModel struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name         string
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt
}

func (userModel) TableName() string { return "users" }

func (m *userModel) toDomain() *domain.User {
	u := &domain.User{
		ID:           m.ID,
		Name:         m.Name,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		Role:         domain.Role(m.Role),
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
	if m.DeletedAt.Valid {
		t := m.DeletedAt.Time
		u.DeletedAt = &t
	}
	return u
}

func fromDomain(u *domain.User) *userModel {
	return &userModel{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         string(u.Role),
	}
}

func (r *repository) Create(ctx context.Context, u *domain.User) error {
	m := fromDomain(u)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		logger.Errorf("userRepository.Create failed", err, map[string]any{"email": u.Email})
		return fmt.Errorf("userRepository.Create: %w", err)
	}
	u.ID = m.ID
	u.CreatedAt = m.CreatedAt
	u.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *repository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var m userModel
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		logger.Errorf("userRepository.FindByEmail failed", err, map[string]any{"email": email})
		return nil, fmt.Errorf("userRepository.FindByEmail: %w", err)
	}
	return m.toDomain(), nil
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var m userModel
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		logger.Errorf("userRepository.FindByID failed", err, map[string]any{"user_id": id})
		return nil, fmt.Errorf("userRepository.FindByID: %w", err)
	}
	return m.toDomain(), nil
}

package auth

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

// RefreshToken = representasi domain satu baris refresh_tokens.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// RefreshRepository = kontrak akses DB untuk refresh_tokens.
type RefreshRepository interface {
	Save(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	FindByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type refreshRepository struct {
	db *gorm.DB
}

func NewRefreshRepository(db *gorm.DB) RefreshRepository {
	return &refreshRepository{db: db}
}

// refreshModel = representasi GORM tabel refresh_tokens.
type refreshModel struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid"`
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

func (refreshModel) TableName() string { return "refresh_tokens" }

func (r *refreshRepository) Save(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	m := &refreshModel{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		logger.Errorf("refreshRepository.Save failed", err, map[string]any{"user_id": userID})
		return fmt.Errorf("refreshRepository.Save: %w", err)
	}
	return nil
}

func (r *refreshRepository) FindByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	var m refreshModel
	err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrInvalidCredentials // token tidak dikenal
		}
		logger.Errorf("refreshRepository.FindByHash failed", err, nil)
		return nil, fmt.Errorf("refreshRepository.FindByHash: %w", err)
	}
	return &RefreshToken{
		ID:        m.ID,
		UserID:    m.UserID,
		TokenHash: m.TokenHash,
		ExpiresAt: m.ExpiresAt,
		RevokedAt: m.RevokedAt,
		CreatedAt: m.CreatedAt,
	}, nil
}

func (r *refreshRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	err := r.db.WithContext(ctx).Model(&refreshModel{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now).Error
	if err != nil {
		logger.Errorf("refreshRepository.Revoke failed", err, map[string]any{"token_id": id})
		return fmt.Errorf("refreshRepository.Revoke: %w", err)
	}
	return nil
}

func (r *refreshRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	err := r.db.WithContext(ctx).Model(&refreshModel{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
	if err != nil {
		logger.Errorf("refreshRepository.RevokeAllForUser failed", err, map[string]any{"user_id": userID})
		return fmt.Errorf("refreshRepository.RevokeAllForUser: %w", err)
	}
	return nil
}

package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
)

type UpdateProfileInput struct {
	Name      string
	Email     string
	Phone     string
	BirthDate *time.Time
}

type ChangePasswordInput struct {
	CurrentPassword string
	NewPassword     string
}

type Service interface {
	GetProfile(ctx context.Context, id uuid.UUID) (*domain.User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, in UpdateProfileInput) (*domain.User, error)
	ChangePassword(ctx context.Context, id uuid.UUID, in ChangePasswordInput) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetProfile(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("userService.GetProfile: %w", err)
	}
	return u, nil
}

func (s *service) UpdateProfile(ctx context.Context, id uuid.UUID, in UpdateProfileInput) (*domain.User, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("userService.UpdateProfile: %w", err)
	}

	email := strings.ToLower(strings.TrimSpace(in.Email))

	// Email berubah → pastikan belum dipakai akun lain.
	if email != u.Email {
		existing, err := s.repo.FindByEmail(ctx, email)
		if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
			return nil, fmt.Errorf("userService.UpdateProfile: %w", err)
		}
		if existing != nil && existing.ID != id {
			return nil, domain.ErrEmailAlreadyExists
		}
	}

	name := strings.TrimSpace(in.Name)
	phone := strings.TrimSpace(in.Phone)

	err = s.repo.UpdateProfile(ctx, id, UpdateProfileFields{
		Name:      name,
		Email:     email,
		Phone:     phone,
		BirthDate: in.BirthDate,
	})
	if err != nil {
		return nil, fmt.Errorf("userService.UpdateProfile: %w", err)
	}

	u.Name = name
	u.Email = email
	u.Phone = phone
	u.BirthDate = in.BirthDate
	return u, nil
}

// ChangePassword memverifikasi password lama sebelum menggantinya —
// mencegah penyalahgunaan sesi yang sedang terbuka.
func (s *service) ChangePassword(ctx context.Context, id uuid.UUID, in ChangePasswordInput) error {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("userService.ChangePassword: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.CurrentPassword))
	if err != nil {
		return domain.ErrInvalidCredentials
	}

	if in.NewPassword == in.CurrentPassword {
		return domain.ErrSamePassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("userService.ChangePassword: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, id, string(hash)); err != nil {
		return fmt.Errorf("userService.ChangePassword: %w", err)
	}
	return nil
}

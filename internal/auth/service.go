package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
	"github.com/rezekoard/be-cms-ecommerce/internal/user"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(ctx context.Context, name, email, password string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (access, refresh string, err error)
	Refresh(ctx context.Context, rawRefreshToken string) (access, refresh string, err error)
	Logout(ctx context.Context, rawRefreshToken string) error
}

type service struct {
	users   user.Repository
	refresh RefreshRepository
	tokens  *TokenManager
}

func NewService(users user.Repository, refresh RefreshRepository, tokens *TokenManager) Service {
	return &service{users: users, refresh: refresh, tokens: tokens}
}

func (s *service) Register(ctx context.Context, name, email, password string) (*domain.User, error) {
	// Cek email sudah dipakai atau belum.
	existing, err := s.users.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("authService.Register: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrEmailAlreadyExists
	}

	// Hash password dengan bcrypt (rules: wajib bcrypt, bukan MD5/SHA1).
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("authService.Register: %w", err)
	}

	u := &domain.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         domain.RoleCustomer,
	}

	if err := s.users.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("authService.Register: %w", err)
	}

	return u, nil
}

func (s *service) Login(ctx context.Context, email, password string) (access, refresh string, err error) {
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Samakan pesannya dgn password salah → jangan bocorkan email mana
			// yang terdaftar (lihat komentar di domain/errors.go).
			return "", "", domain.ErrInvalidCredentials
		}
		return "", "", fmt.Errorf("authService.Login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", "", domain.ErrInvalidCredentials
	}

	return s.issueTokenPair(ctx, u.ID, u.Role)
}

func (s *service) Refresh(ctx context.Context, rawRefresh string) (string, string, error) {
	hash := HashToken(rawRefresh)

	rt, err := s.refresh.FindByHash(ctx, hash)
	if err != nil {
		return "", "", fmt.Errorf("authService.Refresh: %w", err)
	}

	// Reuse detection: token sudah di-revoke tapi dipakai lagi → anggap dicuri.
	if rt.RevokedAt != nil {
		_ = s.refresh.RevokeAllForUser(ctx, rt.UserID)
		return "", "", domain.ErrTokenReuse
	}

	if time.Now().After(rt.ExpiresAt) {
		return "", "", domain.ErrTokenExpired
	}

	// Rotation: matikan token lama sebelum menerbitkan yang baru.
	if err := s.refresh.Revoke(ctx, rt.ID); err != nil {
		return "", "", fmt.Errorf("authService.Refresh: %w", err)
	}

	// Ambil role terbaru user untuk dimasukkan ke access token baru.
	u, err := s.users.FindByID(ctx, rt.UserID)
	if err != nil {
		return "", "", fmt.Errorf("authService.Refresh: %w", err)
	}

	return s.issueTokenPair(ctx, u.ID, u.Role)
}

func (s *service) Logout(ctx context.Context, rawRefresh string) error {
	hash := HashToken(rawRefresh)

	rt, err := s.refresh.FindByHash(ctx, hash)
	if err != nil {
		// Token tidak dikenal — logout tetap dianggap sukses dari sisi user.
		return nil
	}

	if err := s.refresh.Revoke(ctx, rt.ID); err != nil {
		return fmt.Errorf("authService.Logout: %w", err)
	}
	return nil
}

// issueTokenPair menerbitkan access + refresh token sekaligus,
// dan menyimpan hash refresh token ke DB.
func (s *service) issueTokenPair(ctx context.Context, userID uuid.UUID, role domain.Role) (string, string, error) {
	access, err := s.tokens.GenerateAccess(userID, role)
	if err != nil {
		return "", "", fmt.Errorf("authService.IssueTokenPair: %w", err)
	}

	raw, hash, err := s.tokens.GenerateRefresh()
	if err != nil {
		return "", "", fmt.Errorf("authService.IssueTokenPair: %w", err)
	}

	expiresAt := time.Now().Add(s.tokens.RefreshTTL())
	if err := s.refresh.Save(ctx, userID, hash, expiresAt); err != nil {
		return "", "", fmt.Errorf("authService.issueTokenPair: %w", err)
	}

	return access, raw, nil
}

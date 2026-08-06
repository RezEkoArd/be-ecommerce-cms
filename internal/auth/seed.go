package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/rezekoard/be-cms-ecommerce/internal/config"
	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
	"github.com/rezekoard/be-cms-ecommerce/internal/user"
	"github.com/rezekoard/be-cms-ecommerce/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

// SeedAdmin membuat akun admin default kalau belum ada.
// Idempotent: aman dipanggil setiap startup — kalau admin sudah ada, di-skip.
//
// Kredensial diambil dari config (ADMIN_EMAIL / ADMIN_PASSWORD). Endpoint
// register biasa selalu membuat RoleCustomer, jadi admin hanya bisa lahir dari sini.
func SeedAdmin(ctx context.Context, users user.Repository, cfg *config.Config) error {
	// Kredensial wajib dari env. Kalau kosong, jangan seed (Warn, bukan error).
	if cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		logger.Warn("SeedAdmin dilewati: ADMIN_EMAIL / ADMIN_PASSWORD belum diisi")
		return nil
	}

	// Sudah ada? → skip. FindByEmail balas ErrUserNotFound kalau belum ada.
	existing, err := users.FindByEmail(ctx, cfg.AdminEmail)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return fmt.Errorf("auth.SeedAdmin: %w", err)
	}
	if existing != nil {
		logger.Infof("Admin sudah ada, seed dilewati", map[string]any{"email": cfg.AdminEmail})
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth.SeedAdmin: %w", err)
	}

	admin := &domain.User{
		Name:         "Administrator",
		Email:        cfg.AdminEmail,
		PasswordHash: string(hash),
		Role:         domain.RoleAdmin, // <- kuncinya: role admin, bukan customer
	}
	if err := users.Create(ctx, admin); err != nil {
		return fmt.Errorf("auth.SeedAdmin: %w", err)
	}

	logger.Infof("Admin default berhasil dibuat", map[string]any{"email": cfg.AdminEmail})
	return nil
}

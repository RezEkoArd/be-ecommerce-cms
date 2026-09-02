package address

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
)

type AddressInput struct {
	Label      string
	Recipient  string
	Phone      string
	Street     string
	City       string
	PostalCode string
	IsPrimary  bool
}

type Service interface {
	List(ctx context.Context, userID uuid.UUID) ([]domain.Address, error)
	Get(ctx context.Context, userID, id uuid.UUID) (*domain.Address, error)
	Create(ctx context.Context, userID uuid.UUID, in AddressInput) (*domain.Address, error)
	Update(ctx context.Context, userID, id uuid.UUID, in AddressInput) (*domain.Address, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
	SetPrimary(ctx context.Context, userID, id uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) List(ctx context.Context, userID uuid.UUID) ([]domain.Address, error) {
	list, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("addressService.List: %w", err)
	}
	return list, nil
}

func (s *service) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Address, error) {
	a, err := s.repo.FindByID(ctx, userID, id)
	if err != nil {
		return nil, fmt.Errorf("addressService.Get: %w", err)
	}
	return a, nil
}

func (s *service) Create(ctx context.Context, userID uuid.UUID, in AddressInput) (*domain.Address, error) {
	count, err := s.repo.CountByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("addressService.Create: %w", err)
	}

	// Alamat pertama otomatis jadi utama — user selalu punya tujuan kirim.
	isPrimary := in.IsPrimary || count == 0

	if isPrimary {
		if err := s.repo.ClearPrimary(ctx, userID); err != nil {
			return nil, fmt.Errorf("addressService.Create: %w", err)
		}
	}

	a := &domain.Address{
		UserID:     userID,
		Label:      strings.TrimSpace(in.Label),
		Recipient:  strings.TrimSpace(in.Recipient),
		Phone:      strings.TrimSpace(in.Phone),
		Street:     strings.TrimSpace(in.Street),
		City:       strings.TrimSpace(in.City),
		PostalCode: strings.TrimSpace(in.PostalCode),
		IsPrimary:  isPrimary,
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("addressService.Create: %w", err)
	}
	return a, nil
}

func (s *service) Update(ctx context.Context, userID, id uuid.UUID, in AddressInput) (*domain.Address, error) {
	a, err := s.repo.FindByID(ctx, userID, id)
	if err != nil {
		return nil, fmt.Errorf("addressService.Update: %w", err)
	}

	// Menaikkan alamat ini jadi utama → turunkan yang lain dulu.
	if in.IsPrimary && !a.IsPrimary {
		if err := s.repo.ClearPrimary(ctx, userID); err != nil {
			return nil, fmt.Errorf("addressService.Update: %w", err)
		}
	}

	a.Label = strings.TrimSpace(in.Label)
	a.Recipient = strings.TrimSpace(in.Recipient)
	a.Phone = strings.TrimSpace(in.Phone)
	a.Street = strings.TrimSpace(in.Street)
	a.City = strings.TrimSpace(in.City)
	a.PostalCode = strings.TrimSpace(in.PostalCode)
	// Alamat utama tidak bisa "diturunkan" lewat update — user harus
	// menaikkan alamat lain, supaya selalu ada satu yang utama.
	if in.IsPrimary {
		a.IsPrimary = true
	}

	if err := s.repo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("addressService.Update: %w", err)
	}
	return a, nil
}

// Delete menolak penghapusan alamat utama selama masih ada alamat lain —
// mencegah user kehilangan tujuan kirim tanpa sadar.
func (s *service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	a, err := s.repo.FindByID(ctx, userID, id)
	if err != nil {
		return fmt.Errorf("addressService.Delete: %w", err)
	}

	if a.IsPrimary {
		count, err := s.repo.CountByUser(ctx, userID)
		if err != nil {
			return fmt.Errorf("addressService.Delete: %w", err)
		}
		if count > 1 {
			return domain.ErrCannotDeletePrimary
		}
	}

	if err := s.repo.Delete(ctx, userID, id); err != nil {
		return fmt.Errorf("addressService.Delete: %w", err)
	}
	return nil
}

func (s *service) SetPrimary(ctx context.Context, userID, id uuid.UUID) error {
	a, err := s.repo.FindByID(ctx, userID, id)
	if err != nil {
		return fmt.Errorf("addressService.SetPrimary: %w", err)
	}
	if a.IsPrimary {
		return nil
	}

	if err := s.repo.ClearPrimary(ctx, userID); err != nil {
		return fmt.Errorf("addressService.SetPrimary: %w", err)
	}

	a.IsPrimary = true
	if err := s.repo.Update(ctx, a); err != nil {
		return fmt.Errorf("addressService.SetPrimary: %w", err)
	}
	return nil
}

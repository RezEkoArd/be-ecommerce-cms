package cart

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
)

type AddItemInput struct {
	ProductID uuid.UUID
	Quantity  int
}

type Service interface {
	GetCart(ctx context.Context, userID uuid.UUID) (*domain.Cart, error)
	AddItem(ctx context.Context, userID uuid.UUID, in AddItemInput) (*domain.Cart, error)
	UpdateItem(ctx context.Context, userID, productID uuid.UUID, quantity int) (*domain.Cart, error)
	RemoveItem(ctx context.Context, userID, productID uuid.UUID) (*domain.Cart, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetCart(ctx context.Context, userID uuid.UUID) (*domain.Cart, error) {
	if _, err := s.repo.GetOrCreateCart(ctx, userID); err != nil {
		return nil, fmt.Errorf("cartService.GetCart: %w", err)
	}
	cart, err := s.repo.FindCartByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("cartService.GetCart: %w", err)
	}
	return cart, nil
}

func (s *service) AddItem(ctx context.Context, userID uuid.UUID, in AddItemInput) (*domain.Cart, error) {
	cart, err := s.repo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("cartService.AddItem: %w", err)
	}

	// Produk harus ada. FindProduct balas ErrProductNotFound kalau tidak ada.
	product, err := s.repo.FindProduct(ctx, in.ProductID)
	if err != nil {
		return nil, fmt.Errorf("cartService.AddItem: %w", err)
	}

	// Kalau item sudah ada di cart → qty final = lama + baru. Kalau belum → qty baru.
	finalQty := in.Quantity
	existing, err := s.repo.FindItem(ctx, cart.ID, in.ProductID)
	switch {
	case err == nil:
		finalQty = existing.Quantity + in.Quantity
	case errors.Is(err, domain.ErrCartItemNotFound):
		// belum ada — biarkan finalQty = in.Quantity
	default:
		return nil, fmt.Errorf("cartService.AddItem: %w", err)
	}

	// Cek stok terhadap qty FINAL.
	if finalQty > product.Stock {
		return nil, domain.ErrInsufficientStock
	}

	if existing != nil {
		if err := s.repo.UpdateItemQuantity(ctx, cart.ID, in.ProductID, finalQty); err != nil {
			return nil, fmt.Errorf("cartService.AddItem: %w", err)
		}
	} else {
		item := &domain.CartItem{CartID: cart.ID, ProductID: in.ProductID, Quantity: finalQty}
		if err := s.repo.UpsertItem(ctx, item); err != nil {
			return nil, fmt.Errorf("cartService.AddItem: %w", err)
		}
	}

	return s.repo.FindCartByUserID(ctx, userID)
}

func (s *service) UpdateItem(ctx context.Context, userID, productID uuid.UUID, quantity int) (*domain.Cart, error) {
	cart, err := s.repo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("cartService.Upadte: %w", err)
	}

	product, err := s.repo.FindProduct(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("cartService.Update: %w", err)
	}
	if quantity > product.Stock {
		return nil, domain.ErrInsufficientStock
	}

	if err := s.repo.UpdateItemQuantity(ctx, cart.ID, productID, quantity); err != nil {
		return nil, fmt.Errorf("cartService.UpdateItem: %w", err)
	}

	return s.repo.FindCartByUserID(ctx, userID)
}

func (s *service) RemoveItem(ctx context.Context, userID, productID uuid.UUID) (*domain.Cart, error) {
	cart, err := s.repo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("cartService.remove: %w", err)
	}

	if err := s.repo.DeleteItem(ctx, cart.ID, productID); err != nil {
		return nil, fmt.Errorf("cartService.RemoveItem: %w", err)
	}

	return s.repo.FindCartByUserID(ctx, userID)
}

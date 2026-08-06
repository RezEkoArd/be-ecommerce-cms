package catalog

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
)

// Input DTO = data yang sudah tervalidasi dari handler.
// Service tidak tahu soal HTTP/binding — cukup terima nilai apa adanya.

type CreateCategoryInput struct {
	Name string
}

type CreateProductInput struct {
	Name        string
	Description string
	Price       decimal.Decimal
	Stock       int
	CategoryID  *uuid.UUID
}

type UpdateProductInput struct {
	Name        string
	Description string
	Price       decimal.Decimal
	Stock       int
	CategoryID  *uuid.UUID
}

type Service interface {
	// Category
	CreateCategory(ctx context.Context, in CreateCategoryInput) (*domain.Category, error)
	ListCategories(ctx context.Context) ([]domain.Category, error)

	// Product
	CreateProduct(ctx context.Context, in CreateProductInput) (*domain.Product, error)
	UpdateProduct(ctx context.Context, id uuid.UUID, in UpdateProductInput) (*domain.Product, error)
	DeleteProduct(ctx context.Context, id uuid.UUID) error
	GetProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error)
	ListProducts(ctx context.Context, f ProductFilter) ([]domain.Product, int64, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// ---------- Category ----------

func (s *service) CreateCategory(ctx context.Context, in CreateCategoryInput) (*domain.Category, error) {
	c := &domain.Category{
		Name: strings.TrimSpace(in.Name),
		Slug: slugify(in.Name),
	}
	if err := s.repo.CreateCategory(ctx, c); err != nil {
		return nil, fmt.Errorf("catalogService.CreateCategory: %w", err)
	}
	return c, nil
}

func (s *service) ListCategories(ctx context.Context) ([]domain.Category, error) {
	cats, err := s.repo.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("catalogService.ListCategories: %w", err)
	}
	return cats, nil
}

// ---------- Product ----------

func (s *service) CreateProduct(ctx context.Context, in CreateProductInput) (*domain.Product, error) {
	// Kategori (opsional) harus valid kalau diisi.
	if err := s.ensureCategoryExists(ctx, in.CategoryID); err != nil {
		return nil, err
	}

	slug, err := s.uniqueSlug(ctx, in.Name, nil)
	if err != nil {
		return nil, err
	}

	p := &domain.Product{
		Name:        strings.TrimSpace(in.Name),
		Slug:        slug,
		Description: in.Description,
		Price:       in.Price,
		Stock:       in.Stock,
		CategoryID:  in.CategoryID,
	}
	if err := s.repo.CreateProduct(ctx, p); err != nil {
		return nil, fmt.Errorf("catalogService.CreateProduct: %w", err)
	}
	return p, nil
}

func (s *service) UpdateProduct(ctx context.Context, id uuid.UUID, in UpdateProductInput) (*domain.Product, error) {
	// Pastikan produk ada dulu — biar bisa balas 404 yang jelas.
	existing, err := s.repo.FindProductByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("catalogService.UpdateProduct: %w", err)
	}

	if err := s.ensureCategoryExists(ctx, in.CategoryID); err != nil {
		return nil, err
	}

	// Slug ikut nama; cek keunikan tapi abaikan produk ini sendiri.
	slug, err := s.uniqueSlug(ctx, in.Name, &id)
	if err != nil {
		return nil, err
	}

	existing.Name = strings.TrimSpace(in.Name)
	existing.Slug = slug
	existing.Description = in.Description
	existing.Price = in.Price
	existing.Stock = in.Stock
	existing.CategoryID = in.CategoryID

	if err := s.repo.UpdateProduct(ctx, existing); err != nil {
		return nil, fmt.Errorf("catalogService.UpdateProduct: %w", err)
	}
	return existing, nil
}

func (s *service) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteProduct(ctx, id); err != nil {
		return fmt.Errorf("catalogService.DeleteProduct: %w", err)
	}
	return nil
}

func (s *service) GetProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	p, err := s.repo.FindProductByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("catalogService.GetProductByID: %w", err)
	}
	return p, nil
}

func (s *service) GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	p, err := s.repo.FindProductBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("catalogService.GetProductBySlug: %w", err)
	}
	return p, nil
}

func (s *service) ListProducts(ctx context.Context, f ProductFilter) ([]domain.Product, int64, error) {
	products, total, err := s.repo.ListProducts(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("catalogService.ListProducts: %w", err)
	}
	return products, total, nil
}

// ---------- helpers ----------

// ensureCategoryExists memvalidasi category_id kalau diisi. Nil = tanpa kategori.
func (s *service) ensureCategoryExists(ctx context.Context, categoryID *uuid.UUID) error {
	if categoryID == nil {
		return nil
	}
	if _, err := s.repo.FindCategoryByID(ctx, *categoryID); err != nil {
		if errors.Is(err, domain.ErrCategoryNotFound) {
			return domain.ErrCategoryNotFound
		}
		return fmt.Errorf("catalogService.ensureCategoryExists: %w", err)
	}
	return nil
}

// uniqueSlug membuat slug dari nama lalu memastikan tidak bentrok.
// excludeID diisi saat update supaya produk tidak bentrok dengan dirinya sendiri.
func (s *service) uniqueSlug(ctx context.Context, name string, excludeID *uuid.UUID) (string, error) {
	slug := slugify(name)
	exists, err := s.repo.SlugExists(ctx, slug, excludeID)
	if err != nil {
		return "", fmt.Errorf("catalogService.uniqueSlug: %w", err)
	}
	if exists {
		return "", domain.ErrSlugAlreadyExists
	}
	return slug, nil
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// slugify mengubah nama jadi slug URL-friendly: lowercase, spasi/simbol → "-".
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonSlugChars.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

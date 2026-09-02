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
	"github.com/rezekoard/be-cms-ecommerce/pkg/logger"
)

// Input DTO = data yang sudah tervalidasi dari handler.
// Service tidak tahu soal HTTP/binding — cukup terima nilai apa adanya.

type CreateCategoryInput struct {
	Name     string
	ImageURL string
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
	GetCategory(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, in CreateCategoryInput) (*domain.Category, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
	PresignCategoryImage(ctx context.Context, filename string) (*PresignedUpload, error)

	// Product
	CreateProduct(ctx context.Context, in CreateProductInput) (*domain.Product, error)
	UpdateProduct(ctx context.Context, id uuid.UUID, in UpdateProductInput) (*domain.Product, error)
	DeleteProduct(ctx context.Context, id uuid.UUID) error
	GetProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error)
	ListProducts(ctx context.Context, f ProductFilter) ([]domain.Product, int64, error)

	// Product images
	PresignProductImage(ctx context.Context, productID uuid.UUID, filename string) (*PresignedUpload, error)
	ConfirmProductImage(ctx context.Context, productID uuid.UUID, objectKey string) (*domain.ProductImage, error)
	DeleteProductImage(ctx context.Context, productID, imageID uuid.UUID) error
}

// MaxProductImages membatasi jumlah gambar per produk.
const MaxProductImages = 5

// ImageStorage = kebutuhan catalog terhadap object storage, didefinisikan
// di sisi konsumen agar package ini tidak terikat implementasi MinIO.
type ImageStorage interface {
	PresignedUpload(ctx context.Context, prefix, filename string) (uploadURL, objectKey string, err error)
	PublicURL(objectKey string) string
	ObjectKeyFromURL(rawURL string) string
	Remove(ctx context.Context, objectKey string) error
}

// PresignedUpload = jawaban untuk frontend sebelum ia mengunggah file.
type PresignedUpload struct {
	UploadURL string `json:"upload_url"`
	ObjectKey string `json:"object_key"`
	PublicURL string `json:"public_url"`
}

type service struct {
	repo    Repository
	storage ImageStorage
}

func NewService(repo Repository, storage ImageStorage) Service {
	return &service{repo: repo, storage: storage}
}

// ---------- Category ----------

func (s *service) CreateCategory(ctx context.Context, in CreateCategoryInput) (*domain.Category, error) {
	slug := slugify(in.Name)

	exists, err := s.repo.CategorySlugExists(ctx, slug, nil)
	if err != nil {
		return nil, fmt.Errorf("catalogService.CreateCategory: %w", err)
	}
	if exists {
		return nil, domain.ErrSlugAlreadyExists
	}

	c := &domain.Category{
		Name:     strings.TrimSpace(in.Name),
		Slug:     slug,
		ImageURL: strings.TrimSpace(in.ImageURL),
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

func (s *service) GetCategory(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	c, err := s.repo.FindCategoryByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("catalogService.GetCategory: %w", err)
	}
	return c, nil
}

func (s *service) UpdateCategory(ctx context.Context, id uuid.UUID, in CreateCategoryInput) (*domain.Category, error) {
	c, err := s.repo.FindCategoryByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("catalogService.UpdateCategory: %w", err)
	}

	slug := slugify(in.Name)
	exists, err := s.repo.CategorySlugExists(ctx, slug, &id)
	if err != nil {
		return nil, fmt.Errorf("catalogService.UpdateCategory: %w", err)
	}
	if exists {
		return nil, domain.ErrSlugAlreadyExists
	}

	c.Name = strings.TrimSpace(in.Name)
	c.Slug = slug
	c.ImageURL = strings.TrimSpace(in.ImageURL)
	if err := s.repo.UpdateCategory(ctx, c); err != nil {
		return nil, fmt.Errorf("catalogService.UpdateCategory: %w", err)
	}
	return c, nil
}

// DeleteCategory menolak penghapusan kalau kategori masih dipakai produk.
// DB memang ON DELETE SET NULL, tapi melepas puluhan produk dari kategorinya
// tanpa sadar lebih merugikan daripada memaksa admin memindahkannya dulu.
func (s *service) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindCategoryByID(ctx, id); err != nil {
		return fmt.Errorf("catalogService.DeleteCategory: %w", err)
	}

	count, err := s.repo.CountProductsByCategory(ctx, id)
	if err != nil {
		return fmt.Errorf("catalogService.DeleteCategory: %w", err)
	}
	if count > 0 {
		return domain.ErrCategoryInUse
	}

	if err := s.repo.DeleteCategory(ctx, id); err != nil {
		return fmt.Errorf("catalogService.DeleteCategory: %w", err)
	}
	return nil
}

// PresignCategoryImage menerbitkan URL upload untuk gambar sampul kategori.
// Tidak butuh ID kategori — gambar boleh diunggah sebelum kategori dibuat,
// lalu URL-nya dikirim bersama form.
func (s *service) PresignCategoryImage(ctx context.Context, filename string) (*PresignedUpload, error) {
	if s.storage == nil {
		return nil, domain.ErrStorageUnavailable
	}

	uploadURL, objectKey, err := s.storage.PresignedUpload(ctx, "categories", filename)
	if err != nil {
		return nil, fmt.Errorf("catalogService.PresignCategoryImage: %w", err)
	}

	return &PresignedUpload{
		UploadURL: uploadURL,
		ObjectKey: objectKey,
		PublicURL: s.storage.PublicURL(objectKey),
	}, nil
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

// ---------- Product images ----------

// PresignProductImage menyiapkan URL upload. Baris DB baru dibuat setelah
// frontend memanggil ConfirmProductImage — supaya upload yang batal
// tidak meninggalkan baris tanpa file.
func (s *service) PresignProductImage(ctx context.Context, productID uuid.UUID, filename string) (*PresignedUpload, error) {
	if s.storage == nil {
		return nil, domain.ErrStorageUnavailable
	}

	// Pastikan produknya ada sebelum menerbitkan URL.
	if _, err := s.repo.FindProductByID(ctx, productID); err != nil {
		return nil, fmt.Errorf("catalogService.PresignProductImage: %w", err)
	}

	count, err := s.repo.CountProductImages(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("catalogService.PresignProductImage: %w", err)
	}
	if count >= MaxProductImages {
		return nil, domain.ErrTooManyProductImages
	}

	prefix := "products/" + productID.String()
	uploadURL, objectKey, err := s.storage.PresignedUpload(ctx, prefix, filename)
	if err != nil {
		return nil, fmt.Errorf("catalogService.PresignProductImage: %w", err)
	}

	return &PresignedUpload{
		UploadURL: uploadURL,
		ObjectKey: objectKey,
		PublicURL: s.storage.PublicURL(objectKey),
	}, nil
}

// ConfirmProductImage dipanggil frontend setelah file berhasil diunggah.
func (s *service) ConfirmProductImage(ctx context.Context, productID uuid.UUID, objectKey string) (*domain.ProductImage, error) {
	if s.storage == nil {
		return nil, domain.ErrStorageUnavailable
	}

	count, err := s.repo.CountProductImages(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("catalogService.ConfirmProductImage: %w", err)
	}
	if count >= MaxProductImages {
		return nil, domain.ErrTooManyProductImages
	}

	img := &domain.ProductImage{
		ProductID: productID,
		URL:       s.storage.PublicURL(objectKey),
		// Gambar pertama otomatis jadi gambar utama.
		IsPrimary: count == 0,
	}
	if err := s.repo.AddProductImage(ctx, img); err != nil {
		return nil, fmt.Errorf("catalogService.ConfirmProductImage: %w", err)
	}
	return img, nil
}

func (s *service) DeleteProductImage(ctx context.Context, productID, imageID uuid.UUID) error {
	deleted, err := s.repo.DeleteProductImage(ctx, productID, imageID)
	if err != nil {
		return fmt.Errorf("catalogService.DeleteProductImage: %w", err)
	}

	// Baris DB sudah hilang; kegagalan menghapus objek tidak dijadikan error
	// agar admin tidak melihat kegagalan padahal gambarnya sudah lenyap dari UI.
	if s.storage != nil {
		if key := s.storage.ObjectKeyFromURL(deleted.URL); key != "" {
			if err := s.storage.Remove(ctx, key); err != nil {
				logger.Errorf("catalogService.DeleteProductImage storage failed", err, map[string]any{
					"image_id": imageID, "object_key": key,
				})
			}
		}
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

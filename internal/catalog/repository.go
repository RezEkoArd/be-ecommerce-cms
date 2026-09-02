package catalog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
	"github.com/rezekoard/be-cms-ecommerce/pkg/logger"
)

// ProductFilter = kriteria pencarian daftar produk.
type ProductFilter struct {
	CategoryID *uuid.UUID
	Search     string // cari di nama produk
	Limit      int
	Offset     int
}

type Repository interface {
	// Category
	CreateCategory(ctx context.Context, c *domain.Category) error
	ListCategories(ctx context.Context) ([]domain.Category, error)
	FindCategoryByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	UpdateCategory(ctx context.Context, c *domain.Category) error
	DeleteCategory(ctx context.Context, id uuid.UUID) error
	CountProductsByCategory(ctx context.Context, categoryID uuid.UUID) (int64, error)
	CategorySlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error)

	// Product
	CreateProduct(ctx context.Context, p *domain.Product) error
	UpdateProduct(ctx context.Context, p *domain.Product) error
	DeleteProduct(ctx context.Context, id uuid.UUID) error
	FindProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	FindProductBySlug(ctx context.Context, slug string) (*domain.Product, error)
	ListProducts(ctx context.Context, f ProductFilter) ([]domain.Product, int64, error)
	SlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error)

	// Product images
	AddProductImage(ctx context.Context, img *domain.ProductImage) error
	DeleteProductImage(ctx context.Context, productID, imageID uuid.UUID) (*domain.ProductImage, error)
	CountProductImages(ctx context.Context, productID uuid.UUID) (int64, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// ---------- model GORM ----------

type categoryModel struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string
	Slug      string
	ImageURL  string
	CreatedAt time.Time
}

func (categoryModel) TableName() string { return "categories" }

type productModel struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	CategoryID  *uuid.UUID `gorm:"type:uuid"`
	Name        string
	Slug        string
	Description string
	Price       decimal.Decimal `gorm:"type:numeric(12,2)"`
	Stock       int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (productModel) TableName() string { return "products" }

type productImageModel struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProductID uuid.UUID `gorm:"type:uuid"`
	URL       string
	IsPrimary bool
	CreatedAt time.Time
}

func (productImageModel) TableName() string { return "product_images" }

// ---------- konversi ----------

func (m *categoryModel) toDomain() domain.Category {
	return domain.Category{ID: m.ID, Name: m.Name, Slug: m.Slug, ImageURL: m.ImageURL, CreatedAt: m.CreatedAt}
}

func (m *productModel) toDomain() domain.Product {
	return domain.Product{
		ID:          m.ID,
		CategoryID:  m.CategoryID,
		Name:        m.Name,
		Slug:        m.Slug,
		Description: m.Description,
		Price:       m.Price,
		Stock:       m.Stock,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// ---------- Category ----------

func (r *repository) CreateCategory(ctx context.Context, c *domain.Category) error {
	m := &categoryModel{Name: c.Name, Slug: c.Slug, ImageURL: c.ImageURL}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		logger.Errorf("catalogRepository.CreateCategory failed", err, map[string]any{"slug": c.Slug})
		return fmt.Errorf("catalogRepository.CreateCategory: %w", err)
	}
	c.ID = m.ID
	c.CreatedAt = m.CreatedAt
	return nil
}

func (r *repository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	var ms []categoryModel
	if err := r.db.WithContext(ctx).Order("name ASC").Find(&ms).Error; err != nil {
		logger.Errorf("catalogRepository.ListCategories failed", err, nil)
		return nil, fmt.Errorf("catalogRepository.ListCategories: %w", err)
	}
	out := make([]domain.Category, 0, len(ms))
	for i := range ms {
		out = append(out, ms[i].toDomain())
	}

	// Hitung produk per kategori dalam satu query agregat (hindari N+1).
	if len(out) > 0 {
		type countRow struct {
			CategoryID uuid.UUID
			Total      int64
		}
		var rows []countRow
		err := r.db.WithContext(ctx).
			Model(&productModel{}).
			Select("category_id", "COUNT(*) AS total").
			Where("category_id IS NOT NULL").
			Group("category_id").
			Scan(&rows).Error
		if err != nil {
			logger.Errorf("catalogRepository.ListCategories count failed", err, nil)
			return nil, fmt.Errorf("catalogRepository.ListCategories: %w", err)
		}

		byCategory := make(map[uuid.UUID]int64, len(rows))
		for _, row := range rows {
			byCategory[row.CategoryID] = row.Total
		}
		for i := range out {
			out[i].ProductCount = byCategory[out[i].ID]
		}
	}

	return out, nil
}

func (r *repository) UpdateCategory(ctx context.Context, c *domain.Category) error {
	res := r.db.WithContext(ctx).Model(&categoryModel{}).
		Where("id = ?", c.ID).
		Updates(map[string]any{"name": c.Name, "slug": c.Slug, "image_url": c.ImageURL})

	if res.Error != nil {
		logger.Errorf("catalogRepository.UpdateCategory failed", res.Error, map[string]any{"category_id": c.ID})
		return fmt.Errorf("catalogRepository.UpdateCategory: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

func (r *repository) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&categoryModel{}, "id = ?", id)
	if res.Error != nil {
		logger.Errorf("catalogRepository.DeleteCategory failed", res.Error, map[string]any{"category_id": id})
		return fmt.Errorf("catalogRepository.DeleteCategory: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

func (r *repository) CountProductsByCategory(ctx context.Context, categoryID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&productModel{}).
		Where("category_id = ?", categoryID).
		Count(&count).Error
	if err != nil {
		logger.Errorf("catalogRepository.CountProductsByCategory failed", err, map[string]any{"category_id": categoryID})
		return 0, fmt.Errorf("catalogRepository.CountProductsByCategory: %w", err)
	}
	return count, nil
}

// CategorySlugExists cek duplikasi slug kategori. excludeID diisi saat update.
func (r *repository) CategorySlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error) {
	q := r.db.WithContext(ctx).Model(&categoryModel{}).Where("slug = ?", slug)
	if excludeID != nil {
		q = q.Where("id <> ?", *excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		logger.Errorf("catalogRepository.CategorySlugExists failed", err, map[string]any{"slug": slug})
		return false, fmt.Errorf("catalogRepository.CategorySlugExists: %w", err)
	}
	return count > 0, nil
}

func (r *repository) FindCategoryByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	var m categoryModel
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCategoryNotFound
		}
		logger.Errorf("catalogRepository.FindCategoryByID failed", err, map[string]any{"category_id": id})
		return nil, fmt.Errorf("catalogRepository.FindCategoryByID: %w", err)
	}
	c := m.toDomain()
	return &c, nil
}

// ---------- Product ----------

func (r *repository) CreateProduct(ctx context.Context, p *domain.Product) error {
	m := &productModel{
		CategoryID:  p.CategoryID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Price:       p.Price,
		Stock:       p.Stock,
	}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		logger.Errorf("catalogRepository.CreateProduct failed", err, map[string]any{"slug": p.Slug})
		return fmt.Errorf("catalogRepository.CreateProduct: %w", err)
	}
	p.ID = m.ID
	p.CreatedAt = m.CreatedAt
	p.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *repository) UpdateProduct(ctx context.Context, p *domain.Product) error {
	err := r.db.WithContext(ctx).Model(&productModel{}).
		Where("id = ?", p.ID).
		Updates(map[string]any{
			"category_id": p.CategoryID,
			"name":        p.Name,
			"slug":        p.Slug,
			"description": p.Description,
			"price":       p.Price,
			"stock":       p.Stock,
			"updated_at":  time.Now(),
		}).Error
	if err != nil {
		logger.Errorf("catalogRepository.UpdateProduct failed", err, map[string]any{"product_id": p.ID})
		return fmt.Errorf("catalogRepository.UpdateProduct: %w", err)
	}
	return nil
}

func (r *repository) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&productModel{}, "id = ?", id)
	if res.Error != nil {
		logger.Errorf("catalogRepository.DeleteProduct failed", res.Error, map[string]any{"product_id": id})
		return fmt.Errorf("catalogRepository.DeleteProduct: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

// --- Product images ---

func (r *repository) AddProductImage(ctx context.Context, img *domain.ProductImage) error {
	m := &productImageModel{
		ProductID: img.ProductID,
		URL:       img.URL,
		IsPrimary: img.IsPrimary,
	}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		logger.Errorf("catalogRepository.AddProductImage failed", err, map[string]any{
			"product_id": img.ProductID,
		})
		return fmt.Errorf("catalogRepository.AddProductImage: %w", err)
	}
	img.ID = m.ID
	img.CreatedAt = m.CreatedAt
	return nil
}

// DeleteProductImage mengembalikan baris yang dihapus agar pemanggil
// tahu object key mana yang perlu dibuang dari MinIO.
func (r *repository) DeleteProductImage(ctx context.Context, productID, imageID uuid.UUID) (*domain.ProductImage, error) {
	var m productImageModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND product_id = ?", imageID, productID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProductImageNotFound
		}
		logger.Errorf("catalogRepository.DeleteProductImage find failed", err, map[string]any{
			"image_id": imageID,
		})
		return nil, fmt.Errorf("catalogRepository.DeleteProductImage: %w", err)
	}

	if err := r.db.WithContext(ctx).Delete(&productImageModel{}, "id = ?", imageID).Error; err != nil {
		logger.Errorf("catalogRepository.DeleteProductImage failed", err, map[string]any{
			"image_id": imageID,
		})
		return nil, fmt.Errorf("catalogRepository.DeleteProductImage: %w", err)
	}

	return &domain.ProductImage{
		ID: m.ID, ProductID: m.ProductID, URL: m.URL,
		IsPrimary: m.IsPrimary, CreatedAt: m.CreatedAt,
	}, nil
}

func (r *repository) CountProductImages(ctx context.Context, productID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&productImageModel{}).
		Where("product_id = ?", productID).
		Count(&count).Error
	if err != nil {
		logger.Errorf("catalogRepository.CountProductImages failed", err, map[string]any{
			"product_id": productID,
		})
		return 0, fmt.Errorf("catalogRepository.CountProductImages: %w", err)
	}
	return count, nil
}

func (r *repository) FindProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	var m productModel
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProductNotFound
		}
		logger.Errorf("catalogRepository.FindProductByID failed", err, map[string]any{"product_id": id})
		return nil, fmt.Errorf("catalogRepository.FindProductByID: %w", err)
	}
	p := m.toDomain()
	if err := r.attachRelations(ctx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) FindProductBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	var m productModel
	err := r.db.WithContext(ctx).First(&m, "slug = ?", slug).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProductNotFound
		}
		logger.Errorf("catalogRepository.FindProductBySlug failed", err, map[string]any{"slug": slug})
		return nil, fmt.Errorf("catalogRepository.FindProductBySlug: %w", err)
	}
	p := m.toDomain()
	if err := r.attachRelations(ctx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// attachRelations mengisi Category & Images untuk satu produk (dipakai di detail).
func (r *repository) attachRelations(ctx context.Context, p *domain.Product) error {
	if p.CategoryID != nil {
		var cm categoryModel
		if err := r.db.WithContext(ctx).First(&cm, "id = ?", *p.CategoryID).Error; err == nil {
			c := cm.toDomain()
			p.Category = &c
		}
	}

	var ims []productImageModel
	if err := r.db.WithContext(ctx).
		Where("product_id = ?", p.ID).
		Order("is_primary DESC, created_at ASC").
		Find(&ims).Error; err != nil {
		logger.Errorf("catalogRepository.attachRelations failed", err, map[string]any{"product_id": p.ID})
		return fmt.Errorf("catalogRepository.attachRelations: %w", err)
	}
	p.Images = make([]domain.ProductImage, 0, len(ims))
	for _, im := range ims {
		p.Images = append(p.Images, domain.ProductImage{
			ID: im.ID, ProductID: im.ProductID, URL: im.URL,
			IsPrimary: im.IsPrimary, CreatedAt: im.CreatedAt,
		})
	}
	return nil
}

func (r *repository) ListProducts(ctx context.Context, f ProductFilter) ([]domain.Product, int64, error) {
	q := r.db.WithContext(ctx).Model(&productModel{})

	if f.CategoryID != nil {
		q = q.Where("category_id = ?", *f.CategoryID)
	}
	if f.Search != "" {
		// ILIKE = case-insensitive (khas PostgreSQL).
		q = q.Where("name ILIKE ?", "%"+f.Search+"%")
	}

	// Hitung total SEBELUM limit/offset — untuk info paginasi.
	var total int64
	if err := q.Count(&total).Error; err != nil {
		logger.Errorf("catalogRepository.ListProducts count failed", err, nil)
		return nil, 0, fmt.Errorf("catalogRepository.ListProducts: %w", err)
	}

	var ms []productModel
	if err := q.Order("created_at DESC").
		Limit(f.Limit).Offset(f.Offset).
		Find(&ms).Error; err != nil {
		logger.Errorf("catalogRepository.ListProducts failed", err, nil)
		return nil, 0, fmt.Errorf("catalogRepository.ListProducts: %w", err)
	}

	out := make([]domain.Product, 0, len(ms))
	for i := range ms {
		out = append(out, ms[i].toDomain())
	}

	// Ambil gambar untuk semua produk sekaligus (hindari N+1).
	if len(out) > 0 {
		ids := make([]uuid.UUID, 0, len(out))
		for i := range out {
			ids = append(ids, out[i].ID)
		}

		var imgs []productImageModel
		err := r.db.WithContext(ctx).
			Where("product_id IN ?", ids).
			Order("is_primary DESC, created_at ASC").
			Find(&imgs).Error
		if err != nil {
			logger.Errorf("catalogRepository.ListProducts images failed", err, nil)
			return nil, 0, fmt.Errorf("catalogRepository.ListProducts: %w", err)
		}

		byProduct := make(map[uuid.UUID][]domain.ProductImage, len(out))
		for _, im := range imgs {
			byProduct[im.ProductID] = append(byProduct[im.ProductID], domain.ProductImage{
				ID: im.ID, ProductID: im.ProductID, URL: im.URL,
				IsPrimary: im.IsPrimary, CreatedAt: im.CreatedAt,
			})
		}
		for i := range out {
			out[i].Images = byProduct[out[i].ID]
		}
	}

	return out, total, nil
}

// SlugExists cek duplikasi slug. excludeID diisi saat update (abaikan diri sendiri).
func (r *repository) SlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error) {
	q := r.db.WithContext(ctx).Model(&productModel{}).Where("slug = ?", slug)
	if excludeID != nil {
		q = q.Where("id <> ?", *excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		logger.Errorf("catalogRepository.SlugExists failed", err, map[string]any{"slug": slug})
		return false, fmt.Errorf("catalogRepository.SlugExists: %w", err)
	}
	return count > 0, nil
}

package catalog

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
	"github.com/rezekoard/be-cms-ecommerce/pkg/logger"
	"github.com/rezekoard/be-cms-ecommerce/pkg/response"
	"github.com/shopspring/decimal"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

type CreateCategoryRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
}

type ProductRequest struct {
	Name        string  `json:"name"        binding:"required,min=2,max=200"`
	Description string  `json:"description" binding:"max=5000"`
	Price       string  `json:"price"       binding:"required"` // string → parse ke decimal
	Stock       int     `json:"stock"       binding:"gte=0"`
	CategoryID  *string `json:"category_id" binding:"omitempty,uuid"`
}

func (h *Handler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	cat, err := h.svc.CreateCategory(c.Request.Context(), CreateCategoryInput{Name: req.Name})
	if err != nil {
		logger.Errorf("handler.CreateCategory failed", err, map[string]any{"name": req.Name})
		respondProductError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response.NewResponse(201, "berhasil membuat kategori", cat))
}

func (h *Handler) ListCategories(c *gin.Context) {
	cats, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		logger.Errorf("handler.ListCategories failed", err, nil)
		c.JSON(http.StatusInternalServerError, response.NewResponse(500, "terjadi kesalahan internal", nil))
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil mengambil kategori", cats))
}
func (h *Handler) GetCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	cat, err := h.svc.GetCategory(c.Request.Context(), id)
	if err != nil {
		logger.Errorf("handler.GetCategory failed", err, map[string]any{"category_id": id})
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil mengambil kategori", cat))
}

func (h *Handler) UpdateCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	cat, err := h.svc.UpdateCategory(c.Request.Context(), id, CreateCategoryInput{Name: req.Name})
	if err != nil {
		logger.Errorf("handler.UpdateCategory failed", err, map[string]any{"category_id": id})
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil memperbarui kategori", cat))
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	if err := h.svc.DeleteCategory(c.Request.Context(), id); err != nil {
		logger.Errorf("handler.DeleteCategory failed", err, map[string]any{"category_id": id})
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil menghapus kategori", nil))
}

func (h *Handler) ListProducts(c *gin.Context) {
	f := ProductFilter{
		Search: c.Query("search"),
		Limit:  parseIntDefault(c.Query("limit"), 10),
		Offset: parseIntDefault(c.Query("offset"), 0),
	}
	if f.Limit > 100 {
		f.Limit = 100
	}

	if raw := c.Query("category_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.NewResponse(400, "category_id tidak valid", nil))
			return
		}
		f.CategoryID = &id
	}

	products, total, err := h.svc.ListProducts(c.Request.Context(), f)
	if err != nil {
		logger.Errorf("handler.ListProducts failed", err, nil)
		c.JSON(http.StatusInternalServerError, response.NewResponse(500, "terjadi kesalahan internal", nil))
		return
	}

	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil mengambil produk", gin.H{
		"items":  products,
		"total":  total,
		"limit":  f.Limit,
		"offset": f.Offset,
	}))
}

func (h *Handler) CreateProduct(c *gin.Context) {
	var req ProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	price, categoryID, err := parseProductRequest(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "price atau category_id tidak valid", nil))
		return
	}

	p, err := h.svc.CreateProduct(c.Request.Context(), CreateProductInput{
		Name:        req.Name,
		Description: req.Description,
		Price:       price,
		Stock:       req.Stock,
		CategoryID:  categoryID,
	})
	if err != nil {
		logger.Errorf("handler.CreateProduct failed", err, map[string]any{"name": req.Name})
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.NewResponse(201, "berhasil membuat produk", p))
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	// bind body
	var req ProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	price, categoryID, err := parseProductRequest(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "price atau category_id tidak valid", nil))
		return
	}

	p, err := h.svc.UpdateProduct(c.Request.Context(), id, UpdateProductInput{
		Name:        req.Name,
		Description: req.Description,
		Price:       price,
		Stock:       req.Stock,
		CategoryID:  categoryID,
	})
	if err != nil {
		logger.Errorf("handler.UpdateProduct failed", err, map[string]any{"product_id": id})
		respondProductError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil memperbarui produk", p))
}

func (h *Handler) DeleteProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	if err := h.svc.DeleteProduct(c.Request.Context(), id); err != nil {
		logger.Errorf("handler.DeleteProduct failed", err, map[string]any{"product_id": id})
		respondProductError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.NewResponse(200, "barhasil menghapus product", nil))
}

// --- Product images ---

type PresignImageRequest struct {
	Filename string `json:"filename" binding:"required,max=255"`
}

// PresignProductImage — admin: minta URL upload bertanda tangan.
func (h *Handler) PresignProductImage(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	var req PresignImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	out, err := h.svc.PresignProductImage(c.Request.Context(), productID, req.Filename)
	if err != nil {
		logger.Errorf("handler.PresignProductImage failed", err, map[string]any{"product_id": productID})
		respondProductError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil membuat url upload", out))
}

type ConfirmImageRequest struct {
	ObjectKey string `json:"object_key" binding:"required,max=500"`
}

// ConfirmProductImage — admin: catat gambar setelah upload selesai.
func (h *Handler) ConfirmProductImage(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	var req ConfirmImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	img, err := h.svc.ConfirmProductImage(c.Request.Context(), productID, req.ObjectKey)
	if err != nil {
		logger.Errorf("handler.ConfirmProductImage failed", err, map[string]any{"product_id": productID})
		respondProductError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response.NewResponse(201, "berhasil menambah gambar", img))
}

func (h *Handler) DeleteProductImage(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id produk tidak valid", nil))
		return
	}
	imageID, err := uuid.Parse(c.Param("imageId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id gambar tidak valid", nil))
		return
	}

	if err := h.svc.DeleteProductImage(c.Request.Context(), productID, imageID); err != nil {
		logger.Errorf("handler.DeleteProductImage failed", err, map[string]any{"image_id": imageID})
		respondProductError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil menghapus gambar", nil))
}

func (h *Handler) GetProductBySlug(c *gin.Context) {
	slug := c.Param("slug")

	p, err := h.svc.GetProductBySlug(c.Request.Context(), slug)
	if err != nil {
		logger.Errorf("handler.GetProductBySlug failed", err, map[string]any{"slug": slug})
		respondProductError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil mengambil product", p))
}

// Utils Helper
func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return def
	}
	return n
}

func respondProductError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrProductNotFound):
		c.JSON(http.StatusNotFound, response.NewResponse(404, "produk tidak ditemukan", nil))
	case errors.Is(err, domain.ErrCategoryNotFound):
		c.JSON(http.StatusNotFound, response.NewResponse(404, "kategori tidak ditemukan", nil))
	case errors.Is(err, domain.ErrCategoryInUse):
		c.JSON(http.StatusConflict, response.NewResponse(409,
			"kategori masih dipakai produk, pindahkan produknya terlebih dahulu", nil))
	case errors.Is(err, domain.ErrSlugAlreadyExists):
		c.JSON(http.StatusConflict, response.NewResponse(409, "nama serupa sudah digunakan", nil))
	case errors.Is(err, domain.ErrProductImageNotFound):
		c.JSON(http.StatusNotFound, response.NewResponse(404, "gambar tidak ditemukan", nil))
	case errors.Is(err, domain.ErrTooManyProductImages):
		c.JSON(http.StatusConflict, response.NewResponse(409,
			fmt.Sprintf("maksimal %d gambar per produk", MaxProductImages), nil))
	case errors.Is(err, domain.ErrStorageUnavailable):
		c.JSON(http.StatusServiceUnavailable, response.NewResponse(503, "penyimpanan gambar belum dikonfigurasi", nil))
	default:
		c.JSON(http.StatusInternalServerError, response.NewResponse(500, "terjadi kesalahan internal", nil))
	}
}

func parseProductRequest(req ProductRequest) (decimal.Decimal, *uuid.UUID, error) {
	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		return decimal.Decimal{}, nil, err
	}
	if price.IsNegative() {
		return decimal.Decimal{}, nil, errors.New("price tidak boleh negatif")
	}

	var categoryID *uuid.UUID
	if req.CategoryID != nil && *req.CategoryID != "" {
		id, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			return decimal.Decimal{}, nil, err
		}
		categoryID = &id
	}
	return price, categoryID, nil
}

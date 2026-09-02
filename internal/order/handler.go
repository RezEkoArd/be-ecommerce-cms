package order

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
	"github.com/rezekoard/be-cms-ecommerce/internal/middleware"
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

func respondOrderError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		c.JSON(http.StatusNotFound, response.NewResponse(404, "pesanan tidak ditemukan", nil))
	case errors.Is(err, domain.ErrAddressNotFound):
		c.JSON(http.StatusNotFound, response.NewResponse(404, "alamat pengiriman tidak ditemukan", nil))
	case errors.Is(err, domain.ErrCartEmpty):
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "keranjang kosong", nil))
	case errors.Is(err, domain.ErrCouponNotFound):
		c.JSON(http.StatusNotFound, response.NewResponse(404, "kupon tidak ditemukan", nil))
	case errors.Is(err, domain.ErrCouponInvalid):
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "kupon tidak valid atau kadaluarsa", nil))
	case errors.Is(err, domain.ErrCouponAlreadyExists):
		c.JSON(http.StatusConflict, response.NewResponse(409, "kode kupon sudah digunakan", nil))
	case errors.Is(err, domain.ErrDiscountTooLarge):
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "diskon persentase tidak boleh lebih dari 100%", nil))
	case errors.Is(err, domain.ErrInsufficientStock):
		c.JSON(http.StatusConflict, response.NewResponse(409, "stok produk tidak mencukupi", nil))
	case errors.Is(err, domain.ErrProductNotFound):
		c.JSON(http.StatusNotFound, response.NewResponse(404, "produk tidak ditemukan", nil))
	case errors.Is(err, domain.ErrInvalidOrderStatus):
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "status pesanan tidak valid", nil))
	default:
		c.JSON(http.StatusInternalServerError, response.NewResponse(500, "terjadi kesalahan internal", nil))
	}
}

type CheckoutRequest struct {
	AddressID  string `json:"address_id"  binding:"required,uuid"`
	CouponCode string `json:"coupon_code" binding:"omitempty,max=50"`
}

func (h *Handler) Checkout(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	addressID, err := uuid.Parse(req.AddressID)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "address_id tidak valid", nil))
		return
	}

	o, err := h.svc.Checkout(c.Request.Context(), userID, CheckoutInput{
		AddressID:  addressID,
		CouponCode: req.CouponCode,
	})
	if err != nil {
		logger.Errorf("handler.Checkout failed", err, map[string]any{"user_id": userID})
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.NewResponse(201, "berhasil checkout", o))
}

func (h *Handler) ListMyOrders(c *gin.Context) {
	userID := middleware.GetUserID(c)

	orders, err := h.svc.ListMyOrders(c.Request.Context(), userID)
	if err != nil {
		logger.Errorf("handler.ListMyOrders failed", err, map[string]any{"user_id": userID})
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil mengambil order", orders))
}

// ListAllOrders — admin: seluruh pesanan lintas user.
func (h *Handler) ListAllOrders(c *gin.Context) {
	orders, err := h.svc.ListAllOrders(c.Request.Context())
	if err != nil {
		logger.Errorf("handler.ListAllOrders failed", err, nil)
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil mengambil semua order", orders))
}

// GetOrderDetail — admin: detail satu pesanan lintas user.
func (h *Handler) GetOrderDetail(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	o, err := h.svc.GetOrderDetail(c.Request.Context(), orderID)
	if err != nil {
		logger.Errorf("handler.GetOrderDetail failed", err, map[string]any{"order_id": orderID})
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil mengambil detail order", o))
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=draft paid shipped completed cancelled"`
}

// UpdateOrderStatus — admin: ubah status pesanan.
func (h *Handler) UpdateOrderStatus(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	if err := h.svc.UpdateOrderStatus(c.Request.Context(), orderID, domain.OrderStatus(req.Status)); err != nil {
		logger.Errorf("handler.UpdateOrderStatus failed", err, map[string]any{
			"order_id": orderID, "status": req.Status,
		})
		respondOrderError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil memperbarui status pesanan", nil))
}

func (h *Handler) GetOrder(c *gin.Context) {
	userID := middleware.GetUserID(c)

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	o, err := h.svc.GetOrder(c.Request.Context(), userID, orderID)
	if err != nil {
		logger.Errorf("handler.Getorder failed", err, map[string]any{"user_id": userID})
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil mengambil order", o))
}

// handler coupon admin

type CreateCouponRequest struct {
	Code          string  `json:"code" binding:"required,min=3,max=50"`
	DiscountType  string  `json:"discount_type" binding:"required,oneof=percent fixed"`
	DiscountValue string  `json:"discount_value" binding:"required,gt=0"`
	ExpiresAt     *string `json:"expires_at" binding:"omitempty"`
	IsActive      *bool   `json:"is_active" binding:"omitempty"`
}

func (h *Handler) CreateCoupon(c *gin.Context) {
	var req CreateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	value, err := decimal.NewFromString(req.DiscountValue)
	if err != nil || value.IsNegative() {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "discount_value tidak valid", nil))
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.NewResponse(400, "expires_at harus format RFC3339", nil))
			return
		}
		expiresAt = &t
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	coupon, err := h.svc.CreateCoupon(c.Request.Context(), CreateCouponInput{
		Code:          req.Code,
		DiscountType:  domain.DiscountType(req.DiscountType),
		DiscountValue: value,
		ExpiresAt:     expiresAt,
		IsActive:      isActive,
	})
	if err != nil {
		logger.Errorf("handler.CreateCoupon failed", err, map[string]any{"code": req.Code})
		respondOrderError(c, err)
		return
	}
	// c.JSON(http.StatusCreated, response.NewResponse(201, "berhasil membuat kupon", coupon))
	c.JSON(http.StatusCreated, response.NewResponse(201, "berhasil membuat kupon", coupon))

}

func (h *Handler) GetCoupon(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	coupon, err := h.svc.GetCoupon(c.Request.Context(), id)
	if err != nil {
		logger.Errorf("handler.GetCoupon failed", err, map[string]any{"coupon_id": id})
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil mengambil kupon", coupon))
}

// Semua field opsional — hanya yang dikirim yang diubah.
type UpdateCouponRequest struct {
	Code          *string `json:"code" binding:"omitempty,min=3,max=50"`
	DiscountType  *string `json:"discount_type" binding:"omitempty,oneof=percent fixed"`
	DiscountValue *string `json:"discount_value" binding:"omitempty"`
	ExpiresAt     *string `json:"expires_at" binding:"omitempty"`
	IsActive      *bool   `json:"is_active" binding:"omitempty"`
}

func (h *Handler) UpdateCoupon(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	var req UpdateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	in := UpdateCouponInput{Code: req.Code, IsActive: req.IsActive}

	if req.DiscountType != nil {
		dt := domain.DiscountType(*req.DiscountType)
		in.DiscountType = &dt
	}

	if req.DiscountValue != nil {
		value, err := decimal.NewFromString(*req.DiscountValue)
		if err != nil || value.IsNegative() {
			c.JSON(http.StatusBadRequest, response.NewResponse(400, "discount_value tidak valid", nil))
			return
		}
		in.DiscountValue = &value
	}

	// String kosong berarti hapus masa berlaku.
	if req.ExpiresAt != nil {
		var t *time.Time
		if *req.ExpiresAt != "" {
			parsed, err := time.Parse(time.RFC3339, *req.ExpiresAt)
			if err != nil {
				c.JSON(http.StatusBadRequest, response.NewResponse(400, "expires_at harus format RFC3339", nil))
				return
			}
			t = &parsed
		}
		in.ExpiresAt = &t
	}

	coupon, err := h.svc.UpdateCoupon(c.Request.Context(), id, in)
	if err != nil {
		logger.Errorf("handler.UpdateCoupon failed", err, map[string]any{"coupon_id": id})
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil memperbarui kupon", coupon))
}

func (h *Handler) DeleteCoupon(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	if err := h.svc.DeleteCoupon(c.Request.Context(), id); err != nil {
		logger.Errorf("handler.DeleteCoupon failed", err, map[string]any{"coupon_id": id})
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil menghapus kupon", nil))
}

func (h *Handler) ListCoupons(c *gin.Context) {
	coupons, err := h.svc.ListCoupons(c.Request.Context())
	if err != nil {
		logger.Errorf("handler.ListCoupons failed", err, nil)
		respondOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil mengambil kupon", coupons))
}

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
		c.JSON(http.StatusNotFound, response.NewResponse(404, "ordert tidak ditemukan", nil))
	case errors.Is(err, domain.ErrCartEmpty):
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "keranjang kosong", nil))
	case errors.Is(err, domain.ErrCouponNotFound):
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "kupon tidak valid atau kadaluarsa", nil))
	case errors.Is(err, domain.ErrInsufficientStock):
		c.JSON(http.StatusConflict, response.NewResponse(409, "stok produk tidak mencukupi", nil))
	case errors.Is(err, domain.ErrProductNotFound):
		c.JSON(http.StatusNotFound, response.NewResponse(404, "produk tidak ditemukan", nil))
	default:
		c.JSON(http.StatusInternalServerError, response.NewResponse(500, "terjadi kesalahan internal", nil))
	}
}

type CheckoutRequest struct {
	CouponCode string `json:"coupon_code" binding:"omitempty,max=50"`
}

func (h *Handler) Checkout(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.CouponCode = ""
	}

	o, err := h.svc.Checkout(c.Request.Context(), userID, req.CouponCode)
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
	DiscountValue string `json:"discount_value" binding:"required,gt=0"`
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
		Code:		req.Code,
		DiscountType: domain.DiscountType(req.DiscountType),
		DiscountValue: value,
		ExpiresAt: expiresAt,
		IsActive: isActive,
	})	
	if err != nil {
		logger.Errorf("handler.CreateCoupon failed", err, map[string]any{"code": req.Code})
		respondOrderError(c, err)
		return
	}
	// c.JSON(http.StatusCreated, response.NewResponse(201, "berhasil membuat kupon", coupon))
	c.JSON(http.StatusCreated, response.NewResponse(201, "berhasil membuat kupon", coupon))

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
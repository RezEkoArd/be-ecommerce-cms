package cart

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
	"github.com/rezekoard/be-cms-ecommerce/internal/middleware"
	"github.com/rezekoard/be-cms-ecommerce/pkg/logger"
	"github.com/rezekoard/be-cms-ecommerce/pkg/response"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// respondCartError menerjemahkan sentinel error domain → HTTP status.
func respondCartError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrProductNotFound):
		c.JSON(http.StatusNotFound, response.NewResponse(404, "product tidak ditemukan", nil))
	case errors.Is(err, domain.ErrCartItemNotFound):
		c.JSON(http.StatusNotFound, response.NewResponse(404, "item tidak ditemukan", nil))
	case errors.Is(err, domain.ErrInsufficientStock):
		c.JSON(http.StatusConflict, response.NewResponse(409, "stok product tidak mencukupi", nil))
	default:
		c.JSON(http.StatusInternalServerError, response.NewResponse(500, "terjadi kesalahan internal", nil))
	}
}

func (h *Handler) GetCart(c *gin.Context) {
	userID := middleware.GetUserID(c)

	cart, err := h.svc.GetCart(c.Request.Context(), userID)
	if err != nil {
		logger.Errorf("handlerCart.GetCart failed", err, map[string]any{"user_id": userID})
		respondCartError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil mengambil keranjang", cart))
}

type AddItemRequest struct {
	ProductID string `json:"product_id" binding:"required,uuid"`
	Quantity  int    `json:"quantity" binding:"required,gte=1"`
}

func (h *Handler) AddItem(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "product_id tidak valid", nil))
		return
	}

	cart, err := h.svc.AddItem(c.Request.Context(), userID, AddItemInput{
		ProductID: productID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		logger.Errorf("handler.Additem failed", err, map[string]any{"user_id": userID, "product_id": productID})
		respondCartError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil menambahkan item", cart))
}

type UpdateItemRequest struct {
	Quantity int `json:"quantity" binding:"required,gte=1"`
}

func (h *Handler) UpdateItem(c *gin.Context) {
	userID := middleware.GetUserID(c)

	productID, err := uuid.Parse(c.Param("productId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "product_id tidak valid", nil))
		return
	}

	var req UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
	}

	cart, err := h.svc.UpdateItem(c.Request.Context(), userID, productID, req.Quantity)
	if err != nil {
		logger.Errorf("handler.UpdateItem failed", err, map[string]any{"user_id": userID, "product_id": productID})
		respondCartError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil memperbarui item", cart))
}

func (h *Handler) RemoveItem(c *gin.Context) {
	userID := middleware.GetUserID(c)

	productID, err := uuid.Parse(c.Param("productId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "product_id tidak valid", nil))
		return
	}

	cart, err := h.svc.RemoveItem(c.Request.Context(), userID, productID)
	if err != nil {
		logger.Errorf("handler.RemoveItem failed", err, map[string]any{"user_id": userID, "product_id": productID})
		respondCartError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil menghapus item", cart))
}

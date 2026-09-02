package address

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
	"github.com/rezekoard/be-cms-ecommerce/pkg/logger"
	"github.com/rezekoard/be-cms-ecommerce/pkg/response"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// currentUserID membaca klaim yang disetel JWTAuth. Sengaja tidak memakai
// middleware.GetUserID agar tidak terjadi import cycle.
func currentUserID(c *gin.Context) uuid.UUID {
	v, _ := c.Get("user_id")
	id, _ := v.(uuid.UUID)
	return id
}

func respondAddressError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrAddressNotFound):
		c.JSON(http.StatusNotFound, response.NewResponse(404, "alamat tidak ditemukan", nil))
	case errors.Is(err, domain.ErrCannotDeletePrimary):
		c.JSON(http.StatusConflict, response.NewResponse(409,
			"alamat utama tidak bisa dihapus, jadikan alamat lain sebagai utama terlebih dahulu", nil))
	default:
		c.JSON(http.StatusInternalServerError, response.NewResponse(500, "terjadi kesalahan internal", nil))
	}
}

type AddressRequest struct {
	Label      string `json:"label"       binding:"required,min=2,max=50"`
	Recipient  string `json:"recipient"   binding:"required,min=2,max=100"`
	Phone      string `json:"phone"       binding:"required,min=8,max=20"`
	Street     string `json:"street"      binding:"required,min=5,max=500"`
	City       string `json:"city"        binding:"required,min=2,max=100"`
	PostalCode string `json:"postal_code" binding:"required,min=4,max=10"`
	IsPrimary  bool   `json:"is_primary"`
}

func (r AddressRequest) toInput() AddressInput {
	return AddressInput{
		Label: r.Label, Recipient: r.Recipient, Phone: r.Phone,
		Street: r.Street, City: r.City, PostalCode: r.PostalCode,
		IsPrimary: r.IsPrimary,
	}
}

func (h *Handler) List(c *gin.Context) {
	userID := currentUserID(c)

	list, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		logger.Errorf("handler.ListAddresses failed", err, map[string]any{"user_id": userID})
		respondAddressError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil mengambil alamat", list))
}

func (h *Handler) Create(c *gin.Context) {
	userID := currentUserID(c)

	var req AddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	a, err := h.svc.Create(c.Request.Context(), userID, req.toInput())
	if err != nil {
		logger.Errorf("handler.CreateAddress failed", err, map[string]any{"user_id": userID})
		respondAddressError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.NewResponse(201, "berhasil menambah alamat", a))
}

func (h *Handler) Update(c *gin.Context) {
	userID := currentUserID(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	var req AddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	a, err := h.svc.Update(c.Request.Context(), userID, id, req.toInput())
	if err != nil {
		logger.Errorf("handler.UpdateAddress failed", err, map[string]any{"address_id": id})
		respondAddressError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil memperbarui alamat", a))
}

func (h *Handler) Delete(c *gin.Context) {
	userID := currentUserID(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), userID, id); err != nil {
		logger.Errorf("handler.DeleteAddress failed", err, map[string]any{"address_id": id})
		respondAddressError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil menghapus alamat", nil))
}

func (h *Handler) SetPrimary(c *gin.Context) {
	userID := currentUserID(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "id tidak valid", nil))
		return
	}

	if err := h.svc.SetPrimary(c.Request.Context(), userID, id); err != nil {
		logger.Errorf("handler.SetPrimaryAddress failed", err, map[string]any{"address_id": id})
		respondAddressError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil menjadikan alamat utama", nil))
}

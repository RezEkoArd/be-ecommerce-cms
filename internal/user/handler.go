package user

import (
	"errors"
	"net/http"
	"time"

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

// profileResponse = bentuk user yang aman dikirim ke klien.
// PasswordHash sengaja tidak disertakan.
type profileResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
	Phone string `json:"phone"`
	// Format YYYY-MM-DD agar cocok dengan <input type="date">.
	BirthDate string `json:"birth_date"`
}

func toProfileResponse(u *domain.User) profileResponse {
	birthDate := ""
	if u.BirthDate != nil {
		birthDate = u.BirthDate.Format("2006-01-02")
	}
	return profileResponse{
		ID:        u.ID.String(),
		Name:      u.Name,
		Email:     u.Email,
		Role:      string(u.Role),
		Phone:     u.Phone,
		BirthDate: birthDate,
	}
}

func respondUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		c.JSON(http.StatusNotFound, response.NewResponse(404, "pengguna tidak ditemukan", nil))
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		c.JSON(http.StatusConflict, response.NewResponse(409, "email sudah digunakan", nil))
	case errors.Is(err, domain.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, response.NewResponse(401, "password saat ini salah", nil))
	case errors.Is(err, domain.ErrSamePassword):
		c.JSON(http.StatusBadRequest, response.NewResponse(400, "password baru harus berbeda dari yang lama", nil))
	default:
		c.JSON(http.StatusInternalServerError, response.NewResponse(500, "terjadi kesalahan internal", nil))
	}
}

// currentUserID membaca klaim yang disetel JWTAuth. Sengaja tidak memakai
// middleware.GetUserID agar tidak terjadi import cycle (middleware → auth → user).
func currentUserID(c *gin.Context) uuid.UUID {
	v, _ := c.Get("user_id")
	id, _ := v.(uuid.UUID)
	return id
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID := currentUserID(c)

	u, err := h.svc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		logger.Errorf("handler.GetProfile failed", err, map[string]any{"user_id": userID})
		respondUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil mengambil profil", toProfileResponse(u)))
}

type UpdateProfileRequest struct {
	Name  string `json:"name"  binding:"required,min=2,max=100"`
	Email string `json:"email" binding:"required,email"`
	Phone string `json:"phone" binding:"omitempty,max=20"`
	// String kosong berarti tanggal lahir dikosongkan.
	BirthDate string `json:"birth_date" binding:"omitempty,datetime=2006-01-02"`
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userID := currentUserID(c)

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	var birthDate *time.Time
	if req.BirthDate != "" {
		t, err := time.Parse("2006-01-02", req.BirthDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.NewResponse(400, "tanggal lahir tidak valid", nil))
			return
		}
		birthDate = &t
	}

	u, err := h.svc.UpdateProfile(c.Request.Context(), userID, UpdateProfileInput{
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		BirthDate: birthDate,
	})
	if err != nil {
		logger.Errorf("handler.UpdateProfile failed", err, map[string]any{"user_id": userID})
		respondUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil memperbarui profil", toProfileResponse(u)))
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,min=8,max=72"`
	NewPassword     string `json:"new_password"     binding:"required,min=8,max=72"`
}

func (h *Handler) ChangePassword(c *gin.Context) {
	userID := currentUserID(c)

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	err := h.svc.ChangePassword(c.Request.Context(), userID, ChangePasswordInput{
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		// Password tidak pernah ikut ter-log — hanya user_id.
		logger.Errorf("handler.ChangePassword failed", err, map[string]any{"user_id": userID})
		respondUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil mengubah password", nil))
}

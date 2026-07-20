package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rezekoard/be-cms-ecommerce/internal/config"
	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
	"github.com/rezekoard/be-cms-ecommerce/pkg/logger"
	"github.com/rezekoard/be-cms-ecommerce/pkg/response"
)

const refreshCookieName = "refresh_token"
const refreshCookiePath = "/api/auth/refresh"

type Handler struct {
	svc Service
	cfg *config.Config
}

func NewHandler(svc Service, cfg *config.Config) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

// Validasi allowlist lewat binding tag (api-design.md §6).
type RegisterRequest struct {
	Name     string `json:"name"     binding:"required,min=2,max=100"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

func (h *Handler) Register(c *gin.Context) {
	// Implementation for registration
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	u, err := h.svc.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		logger.Errorf("handler.Register failed", err, map[string]any{"email": req.Email})

		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, response.NewResponse(409, "email sudah digunakan", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, response.NewResponse(500, "terjadi kesalahan internal", nil))
		return
	}

	c.JSON(http.StatusCreated, response.NewResponse(201, "berhasi register", gin.H{
		"id":    u.ID,
		"name":  u.Name,
		"email": u.Email,
		"role":  u.Role,
	}))
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
		return
	}

	access, refresh, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		logger.Errorf("handler.Login failed", err, map[string]any{"email": req.Email})
		c.JSON(http.StatusUnauthorized, response.NewResponse(401, "email atau password salah", nil))
		return
	}

	h.setRefreshCookie(c, refresh)
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil login", gin.H{
		"access_token": access,
		"token_type":   "Bearer",
		"expires_in":   int(h.cfg.AccessTTL.Seconds()),
	}))
}

func (h *Handler) Refresh(c *gin.Context) {
	raw, err := c.Cookie(refreshCookieName)
	if err != nil {
		c.JSON(http.StatusUnauthorized, response.NewResponse(401, "sesi todal valid silahkan login kembali", nil))
		return
	}

	access, refresh, err := h.svc.Refresh(c.Request.Context(), raw)
	if err != nil {
		logger.Errorf("handler.Refresj failed", err, nil)
		// Token dicuri / kadaluarsa / tidak dikenal → hapus cookie & minta login ulang.
		h.clearRefreshCookie(c)
		c.JSON(http.StatusUnauthorized, response.NewResponse(401, "sesi tidak valid, silakan login kembali", nil))
		return
	}

	h.setRefreshCookie(c, refresh)

	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil memperbarui sesi", gin.H{
		"access_token": access,
		"token_type":   "Bearer",
		"expires_in":   int(h.cfg.AccessTTL.Seconds()),
	}))
}

func (h *Handler) Logout(c *gin.Context) {
	if raw, err := c.Cookie(refreshCookieName); err == nil {
		if err := h.svc.Logout(c.Request.Context(), raw); err != nil {
			logger.Errorf("handler.logout failed", err, nil)
		}
	}
	h.clearRefreshCookie(c)
	c.JSON(http.StatusOK, response.NewResponse(200, "berhasil logout", nil))
}

func (h *Handler) setRefreshCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteStrictMode) // WAJIB dipanggil sebelum SetCookie
	c.SetCookie(
		refreshCookieName,
		token,
		int(h.cfg.RefreshTTL.Seconds()),
		refreshCookiePath,  // path dibatasi ke endpoint refresh saja
		"",                 // domain
		h.cfg.CookieSecure, // true di production (HTTPS)
		true,               // httpOnly — JS tidak bisa baca
	)
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, "", -1, refreshCookiePath, "", h.cfg.CookieSecure, true)
}

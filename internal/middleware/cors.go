package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CORS mengizinkan browser mengakses API dari origin frontend yang terdaftar.
//
// Origin di-cek lewat allowlist eksplisit — wildcard "*" tidak dipakai karena
// endpoint auth mengirim cookie (credentials), dan spesifikasi CORS melarang
// kombinasi "Access-Control-Allow-Origin: *" dengan credentials.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	// Set lookup dibuat sekali saat startup, bukan tiap request.
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o = strings.TrimSpace(o); o != "" {
			allowed[strings.TrimSuffix(o, "/")] = struct{}{}
		}
	}

	allowMethods := strings.Join([]string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	}, ", ")

	allowHeaders := strings.Join([]string{
		"Authorization",
		"Content-Type",
		"Accept",
		"Origin",
		"X-Requested-With",
	}, ", ")

	maxAge := strconv.Itoa(int((12 * time.Hour).Seconds()))

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Bukan request lintas origin (curl, Postman, server-to-server) — lewatkan.
		if origin == "" {
			c.Next()
			return
		}

		if _, ok := allowed[strings.TrimSuffix(origin, "/")]; !ok {
			// Origin tidak dikenal: jangan kirim header CORS sama sekali,
			// biar browser sendiri yang memblokir. Preflight dihentikan di sini
			// agar tidak diteruskan ke handler.
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		// Response berbeda per origin — beri tahu cache agar tidak tertukar.
		c.Header("Vary", "Origin")

		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Methods", allowMethods)
			c.Header("Access-Control-Allow-Headers", allowHeaders)
			c.Header("Access-Control-Max-Age", maxAge)
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

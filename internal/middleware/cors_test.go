package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newCORSRouter(origins []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(origins))
	r.GET("/api/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestCORS_AllowedOrigin(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:3000"})

	req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("Allow-Origin = %q, mau %q", got, "http://localhost:3000")
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Allow-Credentials = %q, mau %q", got, "true")
	}
	if got := w.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, mau %q", got, "Origin")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, mau %d", w.Code, http.StatusOK)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:3000"})

	req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Tanpa header CORS, browser yang memblokir responsnya.
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, mau kosong untuk origin tak dikenal", got)
	}
}

func TestCORS_PreflightAllowed(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:3000"})

	req := httptest.NewRequest(http.MethodOptions, "/api/products", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status preflight = %d, mau %d", w.Code, http.StatusNoContent)
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("Allow-Headers kosong di preflight")
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Allow-Methods kosong di preflight")
	}
}

func TestCORS_PreflightDisallowed(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:3000"})

	req := httptest.NewRequest(http.MethodOptions, "/api/products", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, mau %d untuk preflight origin tak dikenal", w.Code, http.StatusForbidden)
	}
}

// Request tanpa header Origin (curl, Postman, server-to-server) harus tetap jalan.
func TestCORS_NoOriginHeader(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:3000"})

	req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, mau %d", w.Code, http.StatusOK)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, mau kosong saat tidak ada Origin", got)
	}
}

// Trailing slash pada config maupun header Origin tidak boleh bikin gagal cocok.
func TestCORS_TrailingSlashNormalized(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:3000/"})

	req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("Allow-Origin = %q, mau %q", got, "http://localhost:3000")
	}
}

func TestCORS_MultipleOrigins(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:3000", "https://cms.example.com"})

	for _, origin := range []string{"http://localhost:3000", "https://cms.example.com"} {
		req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("Allow-Origin = %q, mau %q", got, origin)
		}
	}
}

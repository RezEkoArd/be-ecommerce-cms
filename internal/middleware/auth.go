package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rezekoard/be-cms-ecommerce/internal/auth"
	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
	"github.com/rezekoard/be-cms-ecommerce/pkg/response"
)

const (
	ctxUserID = "user_id"
	ctxRole   = "role"
)

// JWTAuth memverifikasi Authorization: Bearer <access_token>.
// Stateless — tidak menyentuh DB sama sekali.
func JWTAuth(tokens *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Implementation for JWT authentication
		header := c.GetHeader("Authorization")
		raw := strings.TrimPrefix(header, "Bearer ")
		if raw == "" || raw == header {
			c.JSON(http.StatusUnauthorized, response.NewResponse(401, "silahkan login terlebih dahulu", nil))
			c.Abort()
			return
		}

		claims, err := tokens.ParseAccess(raw)
		if err != nil {
			c.JSON(http.StatusUnauthorized, response.NewResponse(401, "sesi tidak valid atau kadaluarsa", nil))
			c.Abort()
			return
		}

		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxRole, claims.Role)
		c.Next()
	}
}

func GetUserID(c *gin.Context) uuid.UUID {
	v, _ := c.Get(ctxUserID)
	id, _ := v.(uuid.UUID)
	return id
}

func GetRole(c *gin.Context) domain.Role {
	v, _ := c.Get(ctxRole)
	r, _ := v.(domain.Role)
	return r
}

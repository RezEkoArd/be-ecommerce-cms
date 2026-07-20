package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
)

// AccessClaims = isi (payload) JWT access token.
// JANGAN taruh data sensitif di sini — JWT itu base64, bukan enkripsi.

type AccessClaims struct {
	UserID uuid.UUID   `json:"user_id"`
	Role   domain.Role `json:"role"`
	jwt.RegisteredClaims
}

// Token Manager
type TokenManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenManager(secret string, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{
		secret:     []byte{},
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Geneerate access membuat token JWT
func (t *TokenManager) GenerateAccess(userID uuid.UUID, role domain.Role) (string, error) {
	now := time.Now()
	claims := AccessClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.accessTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("tokenmanager.GenerateAccess: %w", err)
	}
	return signed, nil
}

// ParseAccess memverifikasi signature + exp, lalu kembalikan claims.
// Dipakai di middleware JWTAuth (stateless, tanpa query DB).
func (t *TokenManager) ParseAccess(raw string) (*AccessClaims, error) {
	var claims AccessClaims
	_, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		// Pastikan algoritma sesuai yg kita pakai — cegah algorithm confusion attack.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return t.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("tokenManager.ParseAccess: %w", err)
	}
	return &claims, nil
}

// GenerateRefresh membuat refresh token = string acak (bukan JWT).
// Kembalikan token mentah (dikirim ke cookie) DAN hash-nya (disimpan di DB).
func (t *TokenManager) GenerateRefresh() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("tokenManager.GenerateRefresh: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashToken(raw), nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (t TokenManager) RefreshTTL() time.Duration {
	return t.refreshTTL
}

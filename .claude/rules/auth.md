---
paths:
  - "internal/handler/**/*.go"
  - "internal/midleware/**/*.go"
  - "internal/service/**/*.go"
---

# Rules: Authentication (implementasi Go/Gin)

> **Prinsip, keputusan strategi token, dan alasan keamanan ada di [`security.md`](security.md).**
> File ini fokus ke **cara mengimplementasikannya di Go/Gin** — contoh middleware, handler, service, dan migration. Ikuti keputusan di `security.md`; di sini adalah polanya dalam kode.

---

## 1. Login: access token di body, refresh token via cookie HttpOnly

Implementasi keputusan penyimpanan token dari `security.md` (access → body/memory, refresh → cookie `HttpOnly; Secure; SameSite=Strict`, path dibatasi ke endpoint refresh, hash disimpan di DB).

### ✅ Prinsip
```go
func (h *AuthHandler) Login(c *gin.Context) {
    var req LoginRequest // divalidasi allowlist (lihat api-design.md §6)
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

    // refresh token HANYA lewat cookie HttpOnly, path dibatasi ke endpoint refresh
    c.SetSameSite(http.SameSiteStrictMode)
    c.SetCookie("refresh_token", refresh, int(cfg.RefreshTTL.Seconds()),
        "/api/auth/refresh", "", true /*secure*/, true /*httpOnly*/)

    // access token di body → FE simpan di memory
    c.JSON(http.StatusOK, response.NewResponse(200, "berhasil login",
        gin.H{"access_token": access, "token_type": "Bearer", "expires_in": 900}))
}
```

### ❌ Anti-Pattern
```go
// JANGAN: refresh token di response body → rawan XSS (bisa dibaca JS)
c.JSON(200, gin.H{"access_token": access, "refresh_token": refresh})

// JANGAN: access token di cookie non-HttpOnly → percuma, tetap kena XSS
c.SetCookie("access_token", access, 900, "/", "", true, false) // httpOnly=false

// JANGAN: cookie tanpa Secure/SameSite atau path terlalu luas
c.SetCookie("refresh_token", refresh, ttl, "/", "", false, true) // path "/" + secure=false
```

---

## 2. Middleware JWTAuth — verifikasi Bearer, stateless

Middleware hanya memverifikasi signature + `exp` (tidak sentuh DB), lalu inject `user_id` ke context. Gunakan helper `GetUserID(c)` di handler, jangan baca `c.Get` mentah berulang. (Aturan "verifikasi tanda tangan tanpa query DB" → `security.md` § Aturan backend.)

### ✅ Prinsip
```go
func JWTAuth(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
        if raw == "" || raw == c.GetHeader("Authorization") { // tidak ada / format salah
            c.JSON(http.StatusUnauthorized, response.NewResponse(401, "silakan login terlebih dahulu", nil))
            c.Abort()
            return
        }
        claims, err := token.ParseAccess(raw, secret)
        if err != nil {
            c.JSON(http.StatusUnauthorized, response.NewResponse(401, "sesi tidak valid atau kadaluarsa", nil))
            c.Abort()
            return
        }
        c.Set("user_id", claims.UserID)
        c.Next()
    }
}

// helper — dipakai di handler
func GetUserID(c *gin.Context) uint { v, _ := c.Get("user_id"); id, _ := v.(uint); return id }
```

### ❌ Anti-Pattern
```go
// JANGAN: query DB tiap request untuk validasi access token (harusnya stateless)
user, _ := repo.FindByToken(c, raw) // bikin tiap request nembak DB

// JANGAN: taruh data sensitif di JWT claims — JWT itu base64, BUKAN enkripsi
claims := Claims{UserID: id, PasswordHash: hash, Role: role, Email: email} // password bocor
```

---

## 3. Refresh: rotation + reuse detection

Implementasi aturan rotasi + revocation dari `security.md`. Setiap `POST /api/auth/refresh`: terbitkan refresh token **baru** dan revoke yang lama (rotation). Jika refresh token yang sudah di-revoke dipakai lagi → anggap **dicuri** → revoke **semua** sesi user (reuse detection).

### Tabel DB (migration SQL)
```sql
CREATE TABLE refresh_tokens (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    token_hash  TEXT NOT NULL UNIQUE,   -- simpan HASH, bukan token mentah
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
```

### ✅ Prinsip (service)
```go
func (s *authService) Refresh(ctx context.Context, rawRefresh string) (access, refresh string, err error) {
    hash := token.Hash(rawRefresh)
    rt, err := s.repo.FindRefreshByHash(ctx, hash)
    if err != nil {
        return "", "", fmt.Errorf("authService.Refresh: %w", err)
    }

    // reuse detection: token sudah di-revoke tapi dipakai lagi → kompromi
    if rt.RevokedAt != nil {
        _ = s.repo.RevokeAllForUser(ctx, rt.UserID) // matikan semua sesi user
        return "", "", ErrTokenReuse
    }

    s.repo.Revoke(ctx, rt.ID)                 // rotation: matikan yang lama
    return s.issueTokenPair(ctx, rt.UserID)   // terbitkan pasangan baru
}
```

### ❌ Anti-Pattern
```go
// JANGAN: simpan refresh token mentah di DB → bocor kalau DB dibobol
repo.SaveRefresh(ctx, userID, rawRefresh) // simpan token.Hash(rawRefresh)

// JANGAN: refresh tanpa revoke yang lama → token lama tetap valid selamanya
return s.issueTokenPair(ctx, userID) // tanpa Revoke → tidak ada rotation
```

---

## 4. Logout — revoke server-side, jangan hanya hapus cookie

```go
func (h *AuthHandler) Logout(c *gin.Context) {
    if raw, err := c.Cookie("refresh_token"); err == nil {
        _ = h.svc.RevokeRefresh(c.Request.Context(), raw) // matikan di DB
    }
    c.SetCookie("refresh_token", "", -1, "/api/auth/refresh", "", true, true) // hapus cookie
    c.JSON(http.StatusOK, response.NewResponse(200, "berhasil logout", nil))
}
```

> ❌ Hanya clear cookie tanpa revoke di DB = refresh token masih valid kalau sempat dicuri sebelumnya.

---

## 5. Config auth wajib dari env

```
JWT_SECRET          → wajib, min 32 char, tidak boleh hardcode
ACCESS_TOKEN_TTL    → mis. 15m
REFRESH_TOKEN_TTL   → mis. 168h (7 hari)
```
Di production `Secure` cookie wajib `true` (HTTPS). Untuk dev lokal (HTTP) boleh `false`, kendalikan lewat config — jangan hardcode.

---
paths:
  - "internal/handler/**/*.go"
---

# Rules: API Design

---

## 1. Selalu gunakan ShouldBindJSON, bukan BindJSON

`ShouldBindJSON` mengembalikan error tanpa langsung abort request, memberi kamu kontrol penuh atas response yang dikirim ke client.

### ✅ Prinsip
```go
func (h *UserHandler) Create(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
        return
    }
}

type CreateUserRequest struct {
    Name     string `json:"name"     binding:"required,min=2,max=100"`
    Email    string `json:"email"    binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
}
```

### ❌ Anti-Pattern
```go
// BindJSON otomatis abort — kamu kehilangan kontrol response
if err := c.BindJSON(&req); err != nil {
    return // JANGAN
}
```

---

## 2. Format response harus konsisten di seluruh project

Semua response wajib menggunakan envelope `response.NewResponse` dari `pkg/response`.

### Struktur envelope
```json
{
  "data": { ... },
  "meta_data": {
    "status": 200,
    "message": "berhasil login"
  }
}
```

### ✅ Prinsip
```go
import "github.com/rezekoard/larobas-api/pkg/response"

c.JSON(http.StatusOK, response.NewResponse(200, "berhasil ambil data", result))
c.JSON(http.StatusCreated, response.NewResponse(201, "berhasil dibuat", result))
c.JSON(http.StatusOK, response.NewResponse(200, "berhasil dihapus", nil))
c.JSON(http.StatusUnauthorized, response.NewResponse(401, "email atau password salah", nil))
c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
c.JSON(http.StatusInternalServerError, response.NewResponse(500, "terjadi kesalahan internal", nil))
```

### ❌ Anti-Pattern
```go
c.JSON(200, gin.H{"message": "success", "user": user}) // JANGAN: format tidak konsisten
c.JSON(500, response.NewResponse(500, err.Error(), nil)) // JANGAN: bocorkan detail internal
```

---

## 3. Gunakan HTTP status code yang tepat

```go
// 201 untuk resource baru dibuat
c.JSON(http.StatusCreated, response.NewResponse(201, "berhasil register", ...))

// 404 untuk resource tidak ditemukan
c.JSON(http.StatusNotFound, response.NewResponse(404, "data tidak ditemukan", nil))

// 409 untuk konflik data
c.JSON(http.StatusConflict, response.NewResponse(409, "email sudah digunakan", nil))

// 401 vs 403
c.JSON(http.StatusUnauthorized, response.NewResponse(401, "silakan login terlebih dahulu", nil))
c.JSON(http.StatusForbidden, response.NewResponse(403, "tidak punya akses", nil))
```

---

## 4. Organisasikan route dengan grouping yang bersih

### ✅ Prinsip
```go
api := r.Group("/api")

// Public — tidak perlu auth
auth := api.Group("/auth")
{
    auth.POST("/login", h.Login)
    auth.POST("/register", h.Register)
}

// Protected — wajib JWT
protected := api.Group("")
protected.Use(middleware.JWTAuth())
{
    protected.GET("/users", h.GetAll)
    protected.GET("/users/:id", h.GetByID)
}
```

### ❌ Anti-Pattern
```go
// Middleware dipasang manual per route — mudah terlewat
r.GET("/api/users", middleware.JWTAuth(), h.GetAll)

// Middleware global padahal tidak semua route butuh
r.Use(middleware.JWTAuth()) // JANGAN: endpoint /login jadi butuh token
```

---

## 5. Konvensi URL yang seragam

```
GET    /api/resources        → ambil semua
GET    /api/resources/:id    → ambil satu
POST   /api/resources        → buat baru
PUT    /api/resources/:id    → update semua field
PATCH  /api/resources/:id    → update sebagian field
DELETE /api/resources/:id    → hapus
```

### ❌ Anti-Pattern
```
GET  /api/getUser      // JANGAN: verb di URL
POST /api/user/create  // JANGAN: verb "create" tidak perlu
GET  /api/User         // JANGAN: kapital dan singular
```

---

## 6. Validasi input dengan allowlist, bukan blacklist pola XSS

Input pengguna (params/payload) divalidasi pakai **allowlist** (format & karakter yang diizinkan per field) via binding tag Gin + helper terpusat. Input tak valid → **400**, terpisah dari kegagalan auth (**401**).

XSS adalah masalah **output** (escape saat render), bukan input — jadi **jangan** blokir string yang "menyerupai `<script>`". Blacklist pola gampang di-bypass (`<scr<script>ipt>`, encoding) dan sering false-positive (password `P@ss<w0rd`, nama `O'Brien`). API JSON ini tidak mengeksekusi HTML; data tervalidasi disimpan apa adanya.

Aturan status code:
- Format/isi input salah → **400 Bad Request** (`response.ValidationMessage`).
- Kredensial salah → **401 Unauthorized** (`"email atau password salah"`).
- **Jangan** samarkan kegagalan validasi sebagai 401 — menyembunyikan bug validasi & menghapus sinyal keamanan.

### ✅ Prinsip
```go
// helper terpusat: allowlist per field
type RegisterRequest struct {
    Username string `json:"username" binding:"required,min=3,max=30,alphanum"`
    Email    string `json:"email"    binding:"required,email"`
    Password string `json:"password" binding:"required,min=8,max=72"`
}

func (h *AuthHandler) Register(c *gin.Context) {
    var req RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        // input tak valid → 400, bukan 401
        c.JSON(http.StatusBadRequest, response.NewResponse(400, response.ValidationMessage(err), nil))
        return
    }
    // req sudah tervalidasi allowlist → teruskan ke service apa adanya
}
```

### ❌ Anti-Pattern
```go
// JANGAN: blacklist pola XSS — rapuh & false-positive
if strings.Contains(req.Username, "<script>") || xssRe.MatchString(req.Password) {
    // JANGAN: samarkan sebagai auth gagal
    c.JSON(http.StatusUnauthorized, response.NewResponse(401, "email atau password salah", nil))
    return
}

// JANGAN: sanitasi diam-diam mengubah data user tanpa dia tahu
req.Name = stripTags(req.Name) // hilangkan sebagian input tanpa peringatan
```

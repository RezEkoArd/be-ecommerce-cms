---
paths:
  - "internal/**/*.go"
  - "cmd/**/*.go"
---

# Rules: Go Backend Code

---

## 1. Dependency hanya boleh mengalir satu arah

Handler → Service → Repository. Tidak boleh terbalik atau melompat layer.

### ✅ Prinsip
```go
// handler hanya tahu service (lewat interface)
type UserHandler struct {
    service UserService
}

// service hanya tahu repository (lewat interface)
type UserService struct {
    repo UserRepository
}

// repository hanya tahu *gorm.DB
type UserRepository struct {
    db *gorm.DB
}
```

### ❌ Anti-Pattern
```go
// handler langsung akses database — melompat layer
type UserHandler struct {
    db *gorm.DB // JANGAN: handler tidak boleh tahu DB
}

// service import gin — bocor ke layer HTTP
import "github.com/gin-gonic/gin" // JANGAN di service
```

---

## 2. Setiap layer wajib didefinisikan sebagai interface

Interface memungkinkan layer di atasnya tidak terikat pada implementasi konkret, sehingga mudah di-test dan diganti.

### ✅ Prinsip
```go
type UserRepository interface {
    FindByID(ctx context.Context, id uint) (*model.User, error)
    Create(ctx context.Context, user *model.User) error
}

type UserService interface {
    GetByID(ctx context.Context, id uint) (*model.User, error)
    Create(ctx context.Context, req CreateUserRequest) (*model.User, error)
}

// Implementasi memenuhi interface secara implisit
type userRepository struct { db *gorm.DB }
type userService struct { repo UserRepository }
```

### ❌ Anti-Pattern
```go
// Langsung pakai struct konkret sebagai dependency
type UserService struct {
    repo *userRepository // JANGAN: ikat ke concrete struct
}
```

---

## 3. Selalu wrap error dengan konteks lokasi

Error yang di-wrap memudahkan debugging karena kamu tahu dari mana error berasal.

### ✅ Prinsip
```go
// Format: "namaPackage.namaFungsi: %w"
func (r *userRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
    var user model.User
    err := r.db.WithContext(ctx).First(&user, id).Error
    if err != nil {
        return nil, fmt.Errorf("userRepository.FindByID: %w", err)
    }
    return &user, nil
}

func (s *userService) GetByID(ctx context.Context, id uint) (*model.User, error) {
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("userService.GetByID: %w", err)
    }
    return user, nil
}
```

### ❌ Anti-Pattern
```go
// Return error mentah tanpa konteks
return nil, err // tidak tahu error ini dari mana

// Pakai panic untuk error biasa
if err != nil {
    panic(err) // JANGAN
}

// Expose error database langsung ke HTTP response
c.JSON(500, gin.H{"error": err.Error()}) // JANGAN: bocorkan detail internal
```

---

## 4. Selalu teruskan context dari handler ke repository

Context membawa informasi timeout dan cancellation. Tanpa ini, query database tidak bisa dibatalkan meski request HTTP-nya sudah timeout.

### ✅ Prinsip
```go
// Handler: ambil ctx dari request
func (h *UserHandler) GetByID(c *gin.Context) {
    ctx := c.Request.Context()
    user, err := h.service.GetByID(ctx, id)
}

// Service: teruskan ctx
func (s *userService) GetByID(ctx context.Context, id uint) (*model.User, error) {
    return s.repo.FindByID(ctx, id)
}

// Repository: pakai WithContext sebelum query
func (r *userRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
    err := r.db.WithContext(ctx).First(&user, id).Error
}
```

### ❌ Anti-Pattern
```go
// Query tanpa context
r.db.First(&user, id) // JANGAN: tanpa WithContext

// Buat context baru di tengah chain
ctx := context.Background() // JANGAN: buang ctx dari request
```

---

## 5. Gunakan constructor function untuk inisialisasi struct

### ✅ Prinsip
```go
func NewUserRepository(db *gorm.DB) UserRepository {
    return &userRepository{db: db}
}

func NewUserService(repo UserRepository) UserService {
    return &userService{repo: repo}
}

func NewUserHandler(s UserService) *UserHandler {
    return &UserHandler{service: s}
}
```

### ❌ Anti-Pattern
```go
// Inisialisasi langsung — struct bisa dalam kondisi tidak valid
handler := UserHandler{} // JANGAN: service-nya nil
```

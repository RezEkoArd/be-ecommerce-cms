package domain

import (
	"errors"
)

// Sentinel error = error bernama yang dimiliki domain.
// Dipakai layer lain lewat errors.Is(err, domain.ErrXxx) untuk membedakan
// jenis kegagalan (mis. di handler: pilih 401 vs 404 vs 500).
var (
	// ErrUserNotFound: user dengan id/email yang dicari tidak ada.
	ErrUserNotFound = errors.New("user not found")

	// ErrEmailAlreadyExists: email sudah dipakai user lain (saat register).
	ErrEmailAlreadyExists = errors.New("email already exists")

	// ErrInvalidCredentials: email ada tapi password tidak cocok, ATAU email
	// tidak ada. Sengaja disamakan supaya tidak membocorkan email mana yang
	// terdaftar (lihat security.md: jangan beri tahu penyerang email valid).
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrTokenReuse: refresh token yang sudah di-revoke dipakai lagi → indikasi
	// token dicuri. Respons: revoke SEMUA sesi user (lihat auth.md §3).
	ErrTokenReuse = errors.New("refresh token reuse detected")

	// ErrTokenExpired: refresh token sudah lewat masa berlakunya.
	ErrTokenExpired = errors.New("refresh token expired")

	// Product error Product list
	ErrProductNotFound = errors.New("product not found")

	ErrCategoryNotFound = errors.New("category not found")

	ErrSlugAlreadyExists = errors.New("slug already exists")

	ErrInsufficientStock = errors.New("insufficient stock")

	// error Cart
	ErrCartNotFound     = errors.New("cart not found")
	ErrCartItemNotFound = errors.New("cart item not found")

	ErrOrderNotFound       = errors.New("order not found")
	ErrInvalidOrderStatus  = errors.New("invalid order status")
	ErrCouponNotFound      = errors.New("coupon not found")
	ErrCouponInvalid       = errors.New("coupon invalid or expired")
	ErrCouponAlreadyExists = errors.New("coupon code already exists")
	ErrDiscountTooLarge    = errors.New("percent discount cannot exceed 100")
	ErrCartEmpty           = errors.New("cart is empty")
)

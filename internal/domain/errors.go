package domain

import "errors"

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
)

package response

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// ValidationMessage mengubah error dari ShouldBindJSON menjadi satu pesan
// user-friendly. Dipakai di handler saat binding gagal (input tak valid → 400).
//
// Tujuannya: jangan bocorkan struktur internal error ke client, tapi tetap
// beri petunjuk field mana yang salah.
func ValidationMessage(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		// Ambil error field pertama agar pesan ringkas dan jelas.
		fe := ve[0]
		switch fe.Tag() {
		case "required":
			return fmt.Sprintf("field '%s' wajib diisi", fe.Field())
		case "email":
			return "format email tidak valid"
		case "min":
			return fmt.Sprintf("field '%s' minimal %s karakter", fe.Field(), fe.Param())
		case "max":
			return fmt.Sprintf("field '%s' maksimal %s karakter", fe.Field(), fe.Param())
		case "alphanum":
			return fmt.Sprintf("field '%s' hanya boleh huruf dan angka", fe.Field())
		default:
			return fmt.Sprintf("field '%s' tidak valid", fe.Field())
		}
	}
	// Bukan validation error (mis. JSON malformed) → pesan generik.
	return "request tidak valid"
}

package auth

// File ini HANYA berisi anotasi Swagger (swaggo) untuk endpoint auth.
// Sengaja dipisah dari handler.go agar handler tetap bersih. Fungsi-fungsi di
// sini adalah stub kosong yang tidak pernah dipanggil — swaggo hanya membaca
// komentar di atasnya. Response body sebenarnya memakai response.Response.

// swaggerRegister godoc
// @Summary      Register akun customer baru
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterRequest  true  "Data registrasi"
// @Success      201      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      409      {object}  response.Response
// @Router       /auth/register [post]
func swaggerRegister() {}

// swaggerLogin godoc
// @Summary      Login (customer atau admin)
// @Description  Mengembalikan access_token (Bearer) & menyetel refresh_token via cookie HttpOnly.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "Kredensial login"
// @Success      200      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Router       /auth/login [post]
func swaggerLogin() {}

// swaggerRefresh godoc
// @Summary      Perbarui sesi (rotate refresh token)
// @Description  Membaca refresh_token dari cookie, menerbitkan access_token baru + rotasi refresh token.
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/refresh [post]
func swaggerRefresh() {}

// swaggerLogout godoc
// @Summary      Logout (revoke refresh token & hapus cookie)
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  response.Response
// @Router       /auth/logout [post]
func swaggerLogout() {}

package order

// File ini HANYA berisi anotasi Swagger (swaggo) untuk endpoint order & coupon.
// Dipisah dari handler.go; fungsi di sini stub kosong yang tak pernah dipanggil.

// swaggerCheckout godoc
// @Summary      Checkout: buat order dari keranjang
// @Description  Snapshot harga/nama, kurangi stok, kosongkan cart (transaksi). coupon_code opsional.
// @Tags         Order
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      CheckoutRequest  false  "Kode kupon (opsional)"
// @Success      201      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Failure      409      {object}  response.Response
// @Router       /orders [post]
func swaggerCheckout() {}

// swaggerListMyOrders godoc
// @Summary      Riwayat order milik sendiri
// @Tags         Order
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /orders [get]
func swaggerListMyOrders() {}

// swaggerGetOrder godoc
// @Summary      Detail order (hanya milik sendiri)
// @Tags         Order
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Order ID (UUID)"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /orders/{id} [get]
func swaggerGetOrder() {}

// swaggerCreateCoupon godoc
// @Summary      Buat kupon (admin)
// @Tags         Coupon
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      CreateCouponRequest  true  "Data kupon (discount_type: percent|fixed)"
// @Success      201      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Failure      403      {object}  response.Response
// @Router       /coupons [post]
func swaggerCreateCoupon() {}

// swaggerListCoupons godoc
// @Summary      Daftar kupon (admin)
// @Tags         Coupon
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /coupons [get]
func swaggerListCoupons() {}

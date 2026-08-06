package cart

// File ini HANYA berisi anotasi Swagger (swaggo) untuk endpoint cart.
// Dipisah dari handler.go; fungsi di sini stub kosong yang tak pernah dipanggil.

// swaggerGetCart godoc
// @Summary      Lihat keranjang milik sendiri
// @Tags         Cart
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /cart [get]
func swaggerGetCart() {}

// swaggerAddItem godoc
// @Summary      Tambah item ke keranjang (qty digabung jika sudah ada)
// @Tags         Cart
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      AddItemRequest  true  "Produk & jumlah"
// @Success      200      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Failure      404      {object}  response.Response
// @Failure      409      {object}  response.Response
// @Router       /cart/items [post]
func swaggerAddItem() {}

// swaggerUpdateItem godoc
// @Summary      Ubah jumlah item (set absolut)
// @Tags         Cart
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        productId  path      string             true  "Product ID (UUID)"
// @Param        request    body      UpdateItemRequest  true  "Jumlah baru"
// @Success      200        {object}  response.Response
// @Failure      400        {object}  response.Response
// @Failure      401        {object}  response.Response
// @Failure      404        {object}  response.Response
// @Failure      409        {object}  response.Response
// @Router       /cart/items/{productId} [put]
func swaggerUpdateItem() {}

// swaggerRemoveItem godoc
// @Summary      Hapus satu item dari keranjang
// @Tags         Cart
// @Security     BearerAuth
// @Produce      json
// @Param        productId  path      string  true  "Product ID (UUID)"
// @Success      200        {object}  response.Response
// @Failure      400        {object}  response.Response
// @Failure      401        {object}  response.Response
// @Failure      404        {object}  response.Response
// @Router       /cart/items/{productId} [delete]
func swaggerRemoveItem() {}

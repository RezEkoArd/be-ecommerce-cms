package catalog

// File ini HANYA berisi anotasi Swagger (swaggo) untuk endpoint catalog.
// Dipisah dari handler.go; fungsi di sini stub kosong yang tak pernah dipanggil.

// swaggerCreateCategory godoc
// @Summary      Buat kategori (admin)
// @Tags         Category
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      CreateCategoryRequest  true  "Data kategori"
// @Success      201      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Failure      403      {object}  response.Response
// @Router       /categories [post]
func swaggerCreateCategory() {}

// swaggerListCategories godoc
// @Summary      Daftar kategori (public)
// @Tags         Category
// @Produce      json
// @Success      200  {object}  response.Response
// @Router       /categories [get]
func swaggerListCategories() {}

// swaggerListProducts godoc
// @Summary      Daftar produk dengan filter & paginasi (public)
// @Tags         Product
// @Produce      json
// @Param        search       query     string  false  "Cari nama produk"
// @Param        category_id  query     string  false  "Filter per kategori (UUID)"
// @Param        limit        query     int     false  "Batas item (default 10, maks 100)"
// @Param        offset       query     int     false  "Offset paginasi (default 0)"
// @Success      200          {object}  response.Response
// @Failure      400          {object}  response.Response
// @Router       /products [get]
func swaggerListProducts() {}

// swaggerGetProductBySlug godoc
// @Summary      Detail produk by slug (public)
// @Tags         Product
// @Produce      json
// @Param        slug  path      string  true  "Slug produk"
// @Success      200   {object}  response.Response
// @Failure      404   {object}  response.Response
// @Router       /products/{slug} [get]
func swaggerGetProductBySlug() {}

// swaggerCreateProduct godoc
// @Summary      Buat produk (admin)
// @Tags         Product
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      ProductRequest  true  "Data produk (price sebagai string)"
// @Success      201      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Failure      403      {object}  response.Response
// @Failure      404      {object}  response.Response
// @Failure      409      {object}  response.Response
// @Router       /products [post]
func swaggerCreateProduct() {}

// swaggerUpdateProduct godoc
// @Summary      Update produk (admin)
// @Tags         Product
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id       path      string          true  "Product ID (UUID)"
// @Param        request  body      ProductRequest  true  "Data produk"
// @Success      200      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Failure      403      {object}  response.Response
// @Failure      404      {object}  response.Response
// @Failure      409      {object}  response.Response
// @Router       /products/{id} [put]
func swaggerUpdateProduct() {}

// swaggerDeleteProduct godoc
// @Summary      Hapus produk (admin)
// @Tags         Product
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Product ID (UUID)"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /products/{id} [delete]
func swaggerDeleteProduct() {}

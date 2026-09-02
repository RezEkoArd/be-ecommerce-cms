package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/rezekoard/be-cms-ecommerce/docs" // spec swagger hasil `swag init`
	"github.com/rezekoard/be-cms-ecommerce/internal/address"
	"github.com/rezekoard/be-cms-ecommerce/internal/auth"
	"github.com/rezekoard/be-cms-ecommerce/internal/cart"
	"github.com/rezekoard/be-cms-ecommerce/internal/catalog"
	"github.com/rezekoard/be-cms-ecommerce/internal/config"
	"github.com/rezekoard/be-cms-ecommerce/internal/database"
	"github.com/rezekoard/be-cms-ecommerce/internal/domain"
	"github.com/rezekoard/be-cms-ecommerce/internal/middleware"
	"github.com/rezekoard/be-cms-ecommerce/internal/order"
	"github.com/rezekoard/be-cms-ecommerce/internal/user"
	"github.com/rezekoard/be-cms-ecommerce/pkg/logger"
	"github.com/rezekoard/be-cms-ecommerce/pkg/response"
	"github.com/rezekoard/be-cms-ecommerce/pkg/storage"
)

// @title           E-Commerce CMS API
// @version         1.0
// @description     REST API untuk E-Commerce CMS: auth, katalog produk, keranjang, dan order.
// @description     Endpoint tulis katalog & kupon hanya untuk admin.

// @host      localhost:8080
// @BasePath  /api

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Masukkan: "Bearer {access_token}" yang didapat dari /auth/login.
func main() {

	// Load Config
	cfg := config.Load()

	logger.Init(cfg.AppEnv)

	// SameSite=None mensyaratkan Secure=true — browser modern menolak cookie
	// None tanpa Secure, dan refresh token jadi tidak pernah terkirim.
	if cfg.CookieSameSite == "none" && !cfg.CookieSecure {
		logger.Warn("COOKIE_SAMESITE=none tanpa COOKIE_SECURE=true — browser akan menolak cookie refresh token")
	}

	db := database.Connect(cfg)
	// _ = db

	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo)
	userHandler := user.NewHandler(userSvc)

	addressRepo := address.NewRepository(db)
	addressSvc := address.NewService(addressRepo)
	addressHandler := address.NewHandler(addressSvc)

	// Seed admin default (idempotent) — kalau ADMIN_* kosong, otomatis di-skip.
	if err := auth.SeedAdmin(context.Background(), userRepo, cfg); err != nil {
		logger.Fatal("Gagal seed admin", err)
	}
	refreshRepo := auth.NewRefreshRepository(db)
	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL)
	authService := auth.NewService(userRepo, refreshRepo, tokenManager)
	authHandler := auth.NewHandler(authService, cfg)

	// Catalog: repo → service → handler
	catalogRepo := catalog.NewRepository(db)
	// Object storage untuk gambar produk. Opsional — kalau belum
	// dikonfigurasi, app tetap jalan dan endpoint gambar membalas 503.
	var imageStorage catalog.ImageStorage
	if cfg.MinioEndpoint != "" && cfg.MinioBucket != "" {
		s, err := storage.New(storage.Config{
			Endpoint:  cfg.MinioEndpoint,
			AccessKey: cfg.MinioAccessKey,
			SecretKey: cfg.MinioSecretKey,
			Bucket:    cfg.MinioBucket,
			UseSSL:    cfg.MinioUseSSL,
			PublicURL: cfg.MinioPublicURL,
		})
		if err != nil {
			logger.Fatal("Gagal inisialisasi object storage", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		exists, err := s.BucketExists(ctx)
		cancel()
		if err != nil {
			logger.Errorf("Gagal cek bucket", err, map[string]any{"bucket": cfg.MinioBucket})
		} else if !exists {
			logger.Warn("Bucket tidak ditemukan: " + cfg.MinioBucket)
		} else {
			logger.Infof("Object storage siap", map[string]any{"bucket": cfg.MinioBucket})
		}
		imageStorage = s
	} else {
		logger.Warn("MINIO_ENDPOINT/MINIO_BUCKET kosong — fitur gambar produk nonaktif")
	}

	catalogSvc := catalog.NewService(catalogRepo, imageStorage)
	catalogHandler := catalog.NewHandler(catalogSvc)

	// Cart repo
	cartRepo := cart.NewRepository(db)
	cartSvc := cart.NewService(cartRepo)
	cartHandler := cart.NewHandler(cartSvc)

	// order repo
	orderRepo := order.NewRepository(db)
	orderSvc := order.NewService(orderRepo, cartSvc, addressSvc)
	orderHandler := order.NewHandler(orderSvc)

	r := setupRouter(authHandler, catalogHandler, cartHandler, orderHandler, userHandler, addressHandler, tokenManager, cfg)

	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: r,
	}

	go func() {
		logger.Infof("Server starting", map[string]any{"port": cfg.AppPort})

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Failed to start server", err)
		}
	}()

	//? Tunggu sinyal interrupt (Ctrl+C / SIGTERM) untuk shutdown rapi.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", err)
	}

	logger.Info("Server exited")
}

func setupRouter(authHandler *auth.Handler, catalogHandler *catalog.Handler, cartHandler *cart.Handler, orderHandler *order.Handler, userHandler *user.Handler, addressHandler *address.Handler, tokens *auth.TokenManager, cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	if len(cfg.CORSAllowedOrigins) > 0 {
		r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
		logger.Infof("CORS enabled", map[string]any{"origins": cfg.CORSAllowedOrigins})
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, response.NewResponse(200, "ok", nil))
	})

	// Swagger UI — http://localhost:8080/swagger/index.html
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api")
	// Public — tidak perlu auth
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.Refresh)
		authGroup.POST("/logout", authHandler.Logout)
	}

	// Public — catalog (baca): storefront tidak perlu login
	api.GET("/products", catalogHandler.ListProducts)
	api.GET("/products/:slug", catalogHandler.GetProductBySlug)
	api.GET("/categories", catalogHandler.ListCategories)

	// Protected — wajib JWT
	protected := api.Group("")
	protected.Use(middleware.JWTAuth(tokens))
	{
		protected.GET("/me", userHandler.GetProfile)
		protected.PUT("/me", userHandler.UpdateProfile)
		protected.PUT("/me/password", userHandler.ChangePassword)

		// Alamat pengiriman milik user sendiri.
		protected.GET("/addresses", addressHandler.List)
		protected.POST("/addresses", addressHandler.Create)
		protected.PUT("/addresses/:id", addressHandler.Update)
		protected.DELETE("/addresses/:id", addressHandler.Delete)
		protected.PUT("/addresses/:id/primary", addressHandler.SetPrimary)

		// cart
		protected.GET("/cart", cartHandler.GetCart)
		protected.POST("/cart/items", cartHandler.AddItem)
		protected.PUT("/cart/items/:productId", cartHandler.UpdateItem)
		protected.DELETE("/cart/items/:productId", cartHandler.RemoveItem)

		// Order — wajib login (customer)
		protected.POST("/orders", orderHandler.Checkout)
		protected.GET("/orders", orderHandler.ListMyOrders)
		protected.GET("/orders/:id", orderHandler.GetOrder)

	}

	// Admin only — catalog (tulis): wajib JWT + role admin
	admin := api.Group("")
	admin.Use(middleware.JWTAuth(tokens), middleware.RequireRole(domain.RoleAdmin))
	{
		admin.POST("/categories", catalogHandler.CreateCategory)
		admin.GET("/categories/:id", catalogHandler.GetCategory)
		admin.PUT("/categories/:id", catalogHandler.UpdateCategory)
		admin.DELETE("/categories/:id", catalogHandler.DeleteCategory)
		admin.POST("/categories/images/presign", catalogHandler.PresignCategoryImage)
		admin.POST("/products", catalogHandler.CreateProduct)
		admin.PUT("/products/:id", catalogHandler.UpdateProduct)
		admin.DELETE("/products/:id", catalogHandler.DeleteProduct)

		// Gambar produk — presigned upload, lalu konfirmasi.
		admin.POST("/products/:id/images/presign", catalogHandler.PresignProductImage)
		admin.POST("/products/:id/images", catalogHandler.ConfirmProductImage)
		admin.DELETE("/products/:id/images/:imageId", catalogHandler.DeleteProductImage)

		admin.POST("/coupons", orderHandler.CreateCoupon)
		admin.GET("/coupons", orderHandler.ListCoupons)
		admin.GET("/coupons/:id", orderHandler.GetCoupon)
		admin.PUT("/coupons/:id", orderHandler.UpdateCoupon)
		admin.DELETE("/coupons/:id", orderHandler.DeleteCoupon)

		// Order — admin melihat semua pesanan & mengubah statusnya.
		admin.GET("/admin/orders", orderHandler.ListAllOrders)
		admin.GET("/admin/orders/:id", orderHandler.GetOrderDetail)
		admin.PATCH("/admin/orders/:id/status", orderHandler.UpdateOrderStatus)
	}

	return r
}

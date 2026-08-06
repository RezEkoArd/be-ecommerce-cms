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

	db := database.Connect(cfg)
	// _ = db

	userRepo := user.NewRepository(db)

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
	catalogSvc := catalog.NewService(catalogRepo)
	catalogHandler := catalog.NewHandler(catalogSvc)

	// Cart repo
	cartRepo := cart.NewRepository(db)
	cartSvc := cart.NewService(cartRepo)
	cartHandler := cart.NewHandler(cartSvc)

	// order repo
	orderRepo := order.NewRepository(db)
	orderSvc := order.NewService(orderRepo, cartSvc)
	orderHandler := order.NewHandler(orderSvc)

	r := setupRouter(authHandler, catalogHandler, cartHandler, orderHandler, tokenManager)

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

func setupRouter(authHandler *auth.Handler, catalogHandler *catalog.Handler, cartHandler *cart.Handler, orderHandler *order.Handler, tokens *auth.TokenManager) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

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
		protected.GET("/me", func(c *gin.Context) {
			c.JSON(http.StatusOK, response.NewResponse(200, "berhasil", gin.H{
				"user_id": middleware.GetUserID(c),
				"role":    middleware.GetRole(c),
			}))
		})

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
		admin.POST("/products", catalogHandler.CreateProduct)
		admin.PUT("/products/:id", catalogHandler.UpdateProduct)
		admin.DELETE("/products/:id", catalogHandler.DeleteProduct)

		admin.POST("/coupons", orderHandler.CreateCoupon)
		admin.GET("/coupons", orderHandler.ListCoupons)
	}

	return r
}

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
	"github.com/rezekoard/be-cms-ecommerce/internal/auth"
	"github.com/rezekoard/be-cms-ecommerce/internal/config"
	"github.com/rezekoard/be-cms-ecommerce/internal/database"
	"github.com/rezekoard/be-cms-ecommerce/internal/middleware"
	"github.com/rezekoard/be-cms-ecommerce/internal/user"
	"github.com/rezekoard/be-cms-ecommerce/pkg/logger"
	"github.com/rezekoard/be-cms-ecommerce/pkg/response"
)

func main() {

	// Load Config
	cfg := config.Load()

	logger.Init(cfg.AppEnv)

	db := database.Connect(cfg)
	// _ = db

	userRepo := user.NewRepository(db)
	refreshRepo := auth.NewRefreshRepository(db)
	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL)
	authService := auth.NewService(userRepo, refreshRepo, tokenManager)
	authHandler := auth.NewHandler(authService, cfg)

	r := setupRouter(authHandler, tokenManager)

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

func setupRouter(authHandler *auth.Handler, tokens *auth.TokenManager) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, response.NewResponse(200, "ok", nil))
	})

	api := r.Group("/api")
	// Public — tidak perlu auth
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.Refresh)
		authGroup.POST("/logout", authHandler.Logout)
	}

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
	}

	return r
}

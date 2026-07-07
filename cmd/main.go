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
	"github.com/rezekoard/be-cms-ecommerce/internal/config"
	"github.com/rezekoard/be-cms-ecommerce/internal/database"
	"github.com/rezekoard/be-cms-ecommerce/pkg/logger"
	"github.com/rezekoard/be-cms-ecommerce/pkg/response"
)

func main() {

	// Load Config
	cfg := config.Load()

	logger.Init(cfg.AppEnv)

	db := database.Connect(cfg)
	_ = db
	r := setupRouter()

	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: r,
	}

	go func () {
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

func setupRouter() *gin.Engine {
	 r := gin.New()
	 r.Use(gin.Recovery())

	 r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, response.NewResponse(200, "ok", nil))
		
	})

	return r
}
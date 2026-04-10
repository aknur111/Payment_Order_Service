package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"order-service/internal/app"
	"order-service/internal/client"
	"order-service/internal/repository"
	httptransport "order-service/internal/transport/http"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	_ "order-service/docs"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	cfg := app.LoadConfig()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	db, err := sql.Open("postgres", cfg.DBDSN)
	if err != nil {
		slog.Error("failed to open db", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		slog.Error("ping db", "error", err)
		os.Exit(1)
	}

	orderRepo := repository.NewPostgresOrderRepository(db)
	if err := orderRepo.EnsureSchema(ctx); err != nil {
		slog.Error("ensure schema", "error", err)
		os.Exit(1)
	}

	httpClient := &http.Client{
		Timeout: time.Duration(cfg.HTTPTimeoutSeconds) * time.Second,
	}

	paymentClient := client.NewPaymentClient(cfg.PaymentBaseURL, httpClient)

	orderUC := usecase.NewOrderUsecase(orderRepo, paymentClient)
	handler := httptransport.NewHandler(orderUC)

	r := gin.New()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(httptransport.RequestIDMiddleware())

	handler.Register(r)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: r,
	}

	go func() {
		slog.Info("order service started", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	slog.Info("shutting down server...")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		slog.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}

	slog.Info("server exited properly")
}

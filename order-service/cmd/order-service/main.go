package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"order-service/internal/app"
	"order-service/internal/client"
	"order-service/internal/repository"
	grpctransport "order-service/internal/transport/grpc"
	httptransport "order-service/internal/transport/http"
	"order-service/internal/usecase"

	orderv1 "github.com/aknur111/my-user-service-gen/service/order/v1"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"google.golang.org/grpc"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found, using OS environment variables")
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := app.LoadConfig()

	db, err := sql.Open("postgres", cfg.DBDSNorder)
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

	paymentGRPCClient, err := client.NewGRPCPaymentClient(cfg.PaymentGRPCAddr)
	if err != nil {
		slog.Error("failed to create gRPC payment client", "error", err)
		os.Exit(1)
	}
	defer paymentGRPCClient.Close()

	orderUC := usecase.NewOrderUsecase(orderRepo, paymentGRPCClient)
	grpcHandler := grpctransport.NewOrderGRPCHandler(db, cfg.StreamPollIntervalMs)

	grpcServer := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(grpcServer, grpcHandler)

	grpcLis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		slog.Error("failed to listen gRPC", "addr", cfg.GRPCAddr, "error", err)
		os.Exit(1)
	}

	go func() {
		slog.Info("order gRPC server started", "addr", cfg.GRPCAddr)
		if err := grpcServer.Serve(grpcLis); err != nil {
			slog.Error("gRPC server error", "error", err)
			os.Exit(1)
		}
	}()

	httpHandler := httptransport.NewHandler(orderUC)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(httptransport.RequestIDMiddleware())

	httpHandler.Register(r)

	httpSrv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: r,
	}

	go func() {
		slog.Info("order HTTP server started", "addr", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP listen error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down order service...")
	grpcServer.GracefulStop()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := httpSrv.Shutdown(ctxShutdown); err != nil {
		slog.Error("HTTP shutdown failed", "error", err)
	}

	slog.Info("order service exited properly")
}

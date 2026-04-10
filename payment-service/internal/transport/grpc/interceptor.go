package grpctransport

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

func LoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	slog.Info("gRPC request started",
		"method", info.FullMethod,
	)

	resp, err := handler(ctx, req)

	duration := time.Since(start)

	if err != nil {
		slog.Error("gRPC request failed",
			"method", info.FullMethod,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)
	} else {
		slog.Info("gRPC request completed",
			"method", info.FullMethod,
			"duration_ms", duration.Milliseconds(),
		)
	}

	return resp, err
}

package app

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr              string
	GRPCAddr              string
	DBDSNorder            string
	PaymentGRPCAddr       string
	HTTPTimeoutSeconds    int
	StreamPollIntervalMs  int
}

func LoadConfig() Config {
	timeoutSeconds := 5
	if v := os.Getenv("HTTP_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			timeoutSeconds = parsed
		}
	}

	pollIntervalMs := 500
	if v := os.Getenv("STREAM_POLL_INTERVAL_MS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			pollIntervalMs = parsed
		}
	}

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	grpcAddr := os.Getenv("GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":50052"
	}

	return Config{
		HTTPAddr:             httpAddr,
		GRPCAddr:             grpcAddr,
		DBDSNorder:           os.Getenv("DB_DSN"),
		PaymentGRPCAddr:      os.Getenv("PAYMENT_GRPC_ADDR"),
		HTTPTimeoutSeconds:   timeoutSeconds,
		StreamPollIntervalMs: pollIntervalMs,
	}
}

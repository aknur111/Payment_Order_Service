package app

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr           string
	DBDSN              string
	PaymentBaseURL     string
	HTTPTimeoutSeconds int
}

func LoadConfig() Config {
	timeoutSeconds := 2
	if v := os.Getenv("HTTP_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			timeoutSeconds = parsed
		}
	}

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	return Config{
		HTTPAddr:           addr,
		DBDSN:              os.Getenv("DB_DSN"),
		PaymentBaseURL:     os.Getenv("PAYMENT_BASE_URL"),
		HTTPTimeoutSeconds: timeoutSeconds,
	}
}

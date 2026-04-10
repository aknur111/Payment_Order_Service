package app

import "os"

type Config struct {
	HTTPAddr string
	DBDSN    string
}

func LoadConfig() Config {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	return Config{
		HTTPAddr: addr,
		DBDSN:    os.Getenv("DB_DSN"),
	}
}

package utils

import (
	"log/slog"
	"os"
)

func GetEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func MustEnv(key string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		slog.Error("Value must be provided", slog.String("key", key))
		os.Exit(1)
	}
	return value
}

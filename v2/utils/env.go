package utils

import (
	"log"
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
		log.Fatal("Value must be provided")
	}
	return value
}

package config

import (
    "os"
)

type Config struct {
    DBUrl       string
    GoogleClientID string
    GoogleSecret   string
	JWT_SECRET string 
}

func Load() *Config {
    return &Config{
        DBUrl:          getEnv("DATABASE_URL", "postgres://lol:pass@localhost:5432/db"),
        GoogleClientID: getEnv("GOOGLE_CLIENT_ID", ""),
        GoogleSecret:   getEnv("GOOGLE_CLIENT_SECRET", ""),
		JWT_SECRET: getEnv("JWT_SECRET", ""),
    }
}

func getEnv(key, fallback string) string {
    if value, ok := os.LookupEnv(key); ok {
        return value
    }
    return fallback
}
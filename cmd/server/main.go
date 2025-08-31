package main

import (
    "log"

    "github.com/in-nis/cnis-back/internal/api"
    "github.com/in-nis/cnis-back/internal/config"
    "github.com/in-nis/cnis-back/internal/db"
    "github.com/in-nis/cnis-back/internal/cron"
    "github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
        log.Println("⚠️ No .env file found, using system env")
    }

    cfg := config.Load()

	db.InitDB(cfg.DBUrl)

    r := api.SetupRouter(cfg)

    // Start cron jobs
    cron.StartJobs()

    log.Println("Server running on :8080")
    r.Run(":8000")
}
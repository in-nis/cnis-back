package cron

import (
	"context"
	"log"

	"github.com/in-nis/cnis-back/internal/excel"
	"github.com/robfig/cron/v3"
	"github.com/in-nis/cnis-back/internal/db"
)

func StartJobs() {
	c := cron.New()

	c.AddFunc("@daily", func() {
		log.Println("Running Excel parser job...")

		// path, err := excel.GetExcel()
		// if err != nil {
		// 	log.Println("❌ Failed to download Excel:", err)
		// 	return
		// }

		path := "sheet.xlsx"

		lessons, err := excel.ParseExcel(path)
		if err != nil {
			log.Println("❌ Failed to parse Excel:", err)
			return
		}

		if err := db.SaveLessons(context.Background(), lessons); err != nil {
			log.Println("❌ Failed to save lessons:", err)
			return
		}

		log.Printf("✅ Saved %d lessons\n", len(lessons))
	})

	c.Start()
}
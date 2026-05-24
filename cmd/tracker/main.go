package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/DanChass/uptime-tracker/internal/checker"
	"github.com/DanChass/uptime-tracker/internal/config"
	"github.com/DanChass/uptime-tracker/internal/database"
	"github.com/DanChass/uptime-tracker/internal/models"
)

func main() {
	fmt.Println("Uptime Tracker запускается (Стадия 4: База данных)...")

	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	// 1. Инициализируем БД
	db, err := database.InitDB("tracker.db")
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer db.Close()

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	siteChecker := checker.NewChecker(timeout)

	results := make(chan models.CheckResult, len(cfg.Sites))
	var wg sync.WaitGroup

	for _, url := range cfg.Sites {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			results <- siteChecker.CheckSite(u)
		}(url)
	}

	wg.Wait()
	close(results)

	fmt.Println("\n--- Результаты ---")
	for result := range results {
		status := "UP"
		if !result.IsUp {
			status = "DOWN"
		}

		sslAlert := ""
		if result.SSLError {
			sslAlert = " ⚠️ [SSL ОШИБКА]"
		}

		fmt.Printf("[%s] %s | Статус: %d | Время: %v%s\n",
			status, result.URL, result.StatusCode, result.ResponseTime, sslAlert)

		// 2. Сохраняем в базу
		if err := db.SaveResult(result); err != nil {
			fmt.Printf("Ошибка записи в БД для %s: %v\n", result.URL, err)
		}
	}
}

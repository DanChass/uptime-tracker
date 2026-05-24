package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/DanChass/uptime-tracker/internal/checker"
	"github.com/DanChass/uptime-tracker/internal/config"
	"github.com/DanChass/uptime-tracker/internal/models"
)

func main() {
	fmt.Println("Uptime Tracker запускается (Стадия 3.5: Конфигурация)...")

	// 1. Загружаем конфигурацию
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		// log.Fatalf выведет ошибку и мгновенно завершит программу (код 1)
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	fmt.Printf("Загружено сайтов: %d, Таймаут: %d сек.\n\n", len(cfg.Sites), cfg.TimeoutSeconds)

	// 2. Переводим секунды из конфига в тип time.Duration и создаем чекер
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	siteChecker := checker.NewChecker(timeout)

	// Дальше логика остается почти без изменений, только используем cfg.Sites
	results := make(chan models.CheckResult, len(cfg.Sites))
	var wg sync.WaitGroup
	startAll := time.Now()

	for _, url := range cfg.Sites {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			results <- siteChecker.CheckSite(u)
		}(url)
	}

	wg.Wait()
	close(results)

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
	}

	fmt.Printf("\nВсе сайты проверены за: %v\n", time.Since(startAll))
}

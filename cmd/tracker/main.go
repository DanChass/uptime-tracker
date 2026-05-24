package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/DanChass/uptime-tracker/internal/api"
	"github.com/DanChass/uptime-tracker/internal/checker"
	"github.com/DanChass/uptime-tracker/internal/config"
	"github.com/DanChass/uptime-tracker/internal/database"
	"github.com/DanChass/uptime-tracker/internal/models"
)

func main() {
	fmt.Println("Uptime Tracker запускается (Стадия 5: REST API и Демон)...")

	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	db, err := database.InitDB("tracker.db")
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer db.Close()

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	siteChecker := checker.NewChecker(timeout)

	// 1. Запускаем бесконечный цикл проверок в отдельной горутине (в фоне)
	go func() {
		for {
			fmt.Printf("\n[%s] Начинаем фоновую проверку сайтов...\n", time.Now().Format("15:04:05"))
			runChecks(cfg.Sites, siteChecker, db)

			// Спим 30 секунд до следующей проверки (можно поменять)
			time.Sleep(30 * time.Second)
		}
	}()

	// 2. Запускаем веб-сервер на главном потоке
	// http.ListenAndServe "заморозит" выполнение main, поэтому программа больше не закроется сама
	fmt.Println("🌐 REST API сервер запущен на http://localhost:8080")
	apiServer := api.NewServer(db)
	if err := apiServer.Start(":8080"); err != nil {
		log.Fatalf("Ошибка сервера: %v", err)
	}
}

// runChecks делает ровно то, что раньше делал main: проверяет сайты и пишет в БД
func runChecks(sites []string, siteChecker *checker.Checker, db *database.DB) {
	results := make(chan models.CheckResult, len(sites))
	var wg sync.WaitGroup

	for _, url := range sites {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			results <- siteChecker.CheckSite(u)
		}(url)
	}

	wg.Wait()
	close(results)

	for result := range results {
		if err := db.SaveResult(result); err != nil {
			fmt.Printf("Ошибка записи в БД для %s: %v\n", result.URL, err)
		}
	}
	fmt.Println("✅ Проверка завершена, данные в БД обновлены.")
}

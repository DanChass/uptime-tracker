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
	"github.com/DanChass/uptime-tracker/internal/telegram"
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
			// Передаем весь конфиг cfg целиком, чтобы внутри был доступ к токену
			runChecks(cfg, siteChecker, db)
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
func runChecks(cfg *config.Config, siteChecker *checker.Checker, db *database.DB) {
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

	for result := range results {
		// 1. Пишем в базу
		if err := db.SaveResult(result); err != nil {
			fmt.Printf("Ошибка записи в БД для %s: %v\n", result.URL, err)
		}

		// 2. Если сайт упал или проблема с SSL — бьем тревогу в Telegram!
		if !result.IsUp || result.SSLError {
			alertMsg := fmt.Sprintf("🚨 <b>АЛЕРТ! Проблема с сайтом:</b>\n🌐 %s\nСтатус: %d\nВремя ответа: %v",
				result.URL, result.StatusCode, result.ResponseTime)

			if result.SSLError {
				alertMsg += "\n⚠️ <i>Ошибка SSL сертификата!</i>"
			}

			// Отправляем сообщение
			err := telegram.SendAlert(cfg.TelegramToken, cfg.TelegramChatID, alertMsg)
			if err != nil {
				fmt.Printf("❌ Ошибка отправки в ТГ для %s: %v\n", result.URL, err)
			} else {
				fmt.Printf("📩 Уведомление в ТГ отправлено для %s\n", result.URL)
			}
		}
	}
	fmt.Println("✅ Проверка завершена, данные в БД обновлены.")
}

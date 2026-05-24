package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/DanChass/uptime-tracker/internal/api"
	"github.com/DanChass/uptime-tracker/internal/checker"
	"github.com/DanChass/uptime-tracker/internal/config"
	"github.com/DanChass/uptime-tracker/internal/database"
	"github.com/DanChass/uptime-tracker/internal/models"
	"github.com/DanChass/uptime-tracker/internal/telegram" // Не забудь про импорт ТГ!
)

func main() {
	fmt.Println("Uptime Tracker запускается (Стадия 7: Graceful Shutdown)...")

	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	db, err := database.InitDB("tracker.db")
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	// УБРАЛИ defer db.Close() отсюда, будем закрывать БД вручную в самом конце!

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	siteChecker := checker.NewChecker(timeout)

	// 1. Создаем контекст. Когда мы вызовем cancel(), все процессы, слушающие этот контекст, поймут, что пора закругляться.
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// 2. Фоновый воркер проверок
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// Проверяем, не поступила ли команда на отмену
			select {
			case <-ctx.Done():
				fmt.Println("🛑 Фоновый воркер получил команду на остановку...")
				return
			default:
				fmt.Printf("\n[%s] Начинаем фоновую проверку сайтов...\n", time.Now().Format("15:04:05"))
				runChecks(cfg, siteChecker, db)

				// Умный Sleep. Он ждет 30 секунд, НО если во время ожидания прилетит сигнал отмены (ctx.Done), он моментально прервется.
				select {
				case <-time.After(30 * time.Second):
				case <-ctx.Done():
					fmt.Println("🛑 Воркер прервал ожидание и останавливается...")
					return
				}
			}
		}
	}()

	// 3. Запускаем REST API в отдельной горутине
	apiServer := api.NewServer(db)
	srv := apiServer.SetupServer(":8080")

	go func() {
		fmt.Println("🌐 REST API сервер запущен на http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Критическая ошибка сервера: %v", err)
		}
	}()

	// 4. ПЕРЕХВАТ СИГНАЛОВ ОС
	// Создаем канал, который будет ловить сигналы завершения от Windows/Linux (Ctrl+C или закрытие окна)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit // Программа будет висеть на этой строчке вечно, пока в канал quit не прилетит сигнал от ОС

	// ====== ЕСЛИ МЫ ДОШЛИ СЮДА, ЗНАЧИТ НАЖАЛИ CTRL+C ======
	fmt.Println("\n⚠️ Получен сигнал на выключение. Начинаем Graceful Shutdown...")

	// Говорим фоновому воркеру остановиться
	cancel()

	// Даем веб-серверу 5 секунд, чтобы доотвечать текущим пользователям
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("❌ Ошибка при остановке REST API: %v\n", err)
	} else {
		fmt.Println("✅ REST API остановлен.")
	}

	// Ждем, пока фоновый воркер допишет данные в базу (ждет вызова wg.Done)
	wg.Wait()
	fmt.Println("✅ Все фоновые процессы завершены.")

	// И только теперь, когда никто ничего не пишет, безопасно закрываем базу!
	if err := db.Close(); err != nil {
		fmt.Printf("❌ Ошибка при закрытии БД: %v\n", err)
	} else {
		fmt.Println("✅ Соединение с базой данных безопасно закрыто.")
	}

	fmt.Println("👋 Uptime Tracker успешно и безопасно выключен. Пока!")
}

// runChecks остается БЕЗ ИЗМЕНЕНИЙ, такой же, как в Этапе 6
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
		if err := db.SaveResult(result); err != nil {
			fmt.Printf("Ошибка записи в БД для %s: %v\n", result.URL, err)
		}

		if !result.IsUp || result.SSLError {
			alertMsg := fmt.Sprintf("🚨 <b>АЛЕРТ! Проблема с сайтом:</b>\n🌐 %s\nСтатус: %d\nВремя ответа: %v",
				result.URL, result.StatusCode, result.ResponseTime)
			if result.SSLError {
				alertMsg += "\n⚠️ <i>Ошибка SSL сертификата!</i>"
			}
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

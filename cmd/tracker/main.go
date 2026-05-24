package main

import (
	"fmt"
	"time"

	"github.com/DanChass/uptime-tracker/internal/checker"
)

func main() {
	//fmt.Println("Uptime Tracker запускается...")
	fmt.Println("Uptime Tracker запускается (Стадия 2)...")

	// Список для проверки. Можешь добавить сюда Госуслуги или ФСТЕК для теста
	sites := []string{
		"https://google.com",
		"https://github.com",
		"https://gosuslugi.ru",
		"http://this-site-is-fake-123.com",
		"https://expired.badssl.com/",
	}

	// Создаем наш чекер один раз
	siteChecker := checker.NewChecker()

	// Обходим сайты по очереди
	for _, url := range sites {
		result := siteChecker.CheckSite(url)

		status := "UP"
		if !result.IsUp {
			status = "DOWN"
		}

		// Формируем предупреждение, если есть проблемы с сертификатом
		sslAlert := ""
		if result.SSLError {
			sslAlert = " ⚠️ [SSL ОШИБКА]"
		}

		fmt.Printf("[%s] %s | Статус: %d | Время: %v%s\n",
			status, result.URL, result.StatusCode, result.ResponseTime, sslAlert)

		// Небольшая пауза между запросами (Crawl Delay), чтобы не спамить
		time.Sleep(500 * time.Millisecond)
	}
}

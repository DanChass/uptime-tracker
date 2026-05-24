package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/DanChass/uptime-tracker/internal/checker"
	"github.com/DanChass/uptime-tracker/internal/models"
)

func main() {
	fmt.Println("Uptime Tracker запускается (Стадия 3: Асинхронность)...")

	sites := []string{
		"https://google.com",
		"https://github.com",
		"https://gosuslugi.ru",
		"https://expired.badssl.com", // Сайт с плохим сертификатом
		"http://this-site-is-fake-123.com",
	}

	siteChecker := checker.NewChecker()

	// 1. Создаем канал для сбора результатов.
	// Длина канала равна количеству сайтов (буферизированный канал)
	results := make(chan models.CheckResult, len(sites))

	// 2. Создаем группу ожидания
	var wg sync.WaitGroup

	// Засекаем общее время выполнения программы
	startAll := time.Now()

	// 3. Запускаем горутины в цикле
	for _, url := range sites {
		wg.Add(1) // Увеличиваем счетчик: +1 задача

		// Запускаем анонимную функцию параллельно
		// Передаем u (url) как аргумент, чтобы избежать бага замыкания
		go func(u string) {
			defer wg.Done() // В самом конце скажем: "Задача выполнена (-1)"

			// Проверяем сайт
			result := siteChecker.CheckSite(u)

			// Отправляем результат в канал (в трубу)
			results <- result
		}(url)
	}

	// 4. Ждем, пока все горутины не вызовут wg.Done()
	wg.Wait()

	// Закрываем канал (больше никто в него писать не будет)
	close(results)

	// 5. Вычитываем все результаты из канала
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

	// Выводим общее время работы
	fmt.Printf("\nВсе сайты проверены за: %v\n", time.Since(startAll))
}

package checker

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"time"

	"github.com/DanChass/uptime-tracker/internal/models"
)

// Указываю свой User-Agent
const userAgent = "UptimeTrackerBot/1.0 (+https://github.com/DanChass/uptime-tracker)"

// Checker - структура, которая хранит настроенного клиента
type Checker struct {
	client *http.Client
}

// NewChecker - конструктор, который настраивает HTTP-клиент один раз
func NewChecker() *Checker {
	// 1. Берем системные сертификаты
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	// 2. Пытаемся подгрузить сертификат Минцифры (если он лежит в корне проекта)
	// Если файла нет, ошибка просто проигнорируется (err != nil), и сервис не упадет
	certs, err := os.ReadFile("russian_trusted_root_ca.crt")
	if err == nil {
		rootCAs.AppendCertsFromPEM(certs)
	}

	// 3. Настраиваем транспорт
	tlsConfig := &tls.Config{
		RootCAs:            rootCAs,
		InsecureSkipVerify: false, // Безопасность включена!
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	// 4. Собираем и возвращаем чекер
	return &Checker{
		client: &http.Client{
			Timeout:   10 * time.Second, // Обязательный таймаут
			Transport: transport,
		},
	}
}

// CheckSite - метод для проверки конкретного URL
func (c *Checker) CheckSite(url string) models.CheckResult {
	start := time.Now()

	// Создаем запрос типа HEAD (вместо GET)
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return c.createErrorResult(url, start)
	}

	// Устанавливаем наш кастомный User-Agent
	req.Header.Set("User-Agent", userAgent)

	// Выполняем запрос
	resp, err := c.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return c.createErrorResult(url, start)
	}
	defer resp.Body.Close()

	return models.CheckResult{
		URL:          url,
		StatusCode:   resp.StatusCode,
		ResponseTime: duration,
		IsUp:         resp.StatusCode >= 200 && resp.StatusCode < 400,
		CheckedAt:    time.Now(),
	}
}

// Вспомогательный метод, чтобы не дублировать код при ошибках
func (c *Checker) createErrorResult(url string, start time.Time) models.CheckResult {
	return models.CheckResult{
		URL:          url,
		StatusCode:   0,
		ResponseTime: time.Since(start),
		IsUp:         false,
		CheckedAt:    time.Now(),
	}
}

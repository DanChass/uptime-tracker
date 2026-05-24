package checker

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/DanChass/uptime-tracker/internal/models"
)

const userAgent = "UptimeTrackerBot/1.0 (+https://github.com/DanChass/uptime-tracker)"

// Checker теперь хранит два клиента
type Checker struct {
	strictClient   *http.Client // Проверяет сертификаты
	insecureClient *http.Client // Игнорирует сертификаты
}

func NewChecker(timeout time.Duration) *Checker {
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	certs, err := os.ReadFile("russian_trusted_root_ca.crt")
	if err == nil {
		rootCAs.AppendCertsFromPEM(certs)
	}

	// Настройка для строгого клиента
	strictTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            rootCAs,
			InsecureSkipVerify: false, // Безопасность ВКЛ
		},
	}

	// Настройка для всеядного клиента
	insecureTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Безопасность ВЫКЛ (для второго шанса)
		},
	}

	return &Checker{
		strictClient: &http.Client{
			Timeout:   timeout, // Используем переданный таймаут
			Transport: strictTransport,
		},
		insecureClient: &http.Client{
			Timeout:   timeout, // Используем переданный таймаут
			Transport: insecureTransport,
		},
	}
}

func (c *Checker) CheckSite(url string) models.CheckResult {
	start := time.Now()

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return c.createErrorResult(url, start, false)
	}
	req.Header.Set("User-Agent", userAgent)

	// Попытка №1: Строгая проверка
	resp, err := c.strictClient.Do(req)

	if err != nil {
		// Анализируем ошибку. В Go сетевые ошибки оборачиваются,
		// но текст "certificate" или "x509" надежно выдает проблемы с TLS.
		if strings.Contains(err.Error(), "certificate") || strings.Contains(err.Error(), "x509") {
			// Переходим к Попытке №2
			return c.checkInsecure(req, url, start)
		}

		// Если это таймаут или нет интернета — сайт реально лежит
		return c.createErrorResult(url, start, false)
	}
	defer resp.Body.Close()

	return models.CheckResult{
		URL:          url,
		StatusCode:   resp.StatusCode,
		ResponseTime: time.Since(start),
		IsUp:         resp.StatusCode >= 200 && resp.StatusCode < 400,
		SSLError:     false,
		CheckedAt:    time.Now(),
	}
}

// checkInsecure выполняет скрытый запасной запрос
func (c *Checker) checkInsecure(req *http.Request, url string, start time.Time) models.CheckResult {
	resp, err := c.insecureClient.Do(req)

	if err != nil {
		// Сайт лежит окончательно (даже без проверки SSL не смогли достучаться)
		return c.createErrorResult(url, start, true)
	}
	defer resp.Body.Close()

	return models.CheckResult{
		URL:          url,
		StatusCode:   resp.StatusCode,
		ResponseTime: time.Since(start), // Учитываем время обеих попыток
		IsUp:         resp.StatusCode >= 200 && resp.StatusCode < 400,
		SSLError:     true, // Ставим флаг для преподавателя!
		CheckedAt:    time.Now(),
	}
}

func (c *Checker) createErrorResult(url string, start time.Time, sslError bool) models.CheckResult {
	return models.CheckResult{
		URL:          url,
		StatusCode:   0,
		ResponseTime: time.Since(start),
		IsUp:         false,
		SSLError:     sslError,
		CheckedAt:    time.Now(),
	}
}

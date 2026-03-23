package models

import "time"

// Site описывает веб-ресурс, который мы мониторим
type Site struct {
	ID  int
	URL string
}

// CheckResult описывает результат одной проверки (пинга)
type CheckResult struct {
	SiteID       int
	StatusCode   int           // HTTP статус (например, 200 или 404)
	ResponseTime time.Duration // Время, за которое сайт ответил
	IsUp         bool          // Доступен ли сайт
	CheckedAt    time.Time     // Время проверки
}

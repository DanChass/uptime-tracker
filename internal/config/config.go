package config

import (
	"encoding/json"
	"os"
)

// Config описывает структуру нашего JSON файла.
// Теги `json:"..."` подсказывают парсеру, из каких полей брать данные.
type Config struct {
	TimeoutSeconds int      `json:"timeout_seconds"`
	Sites          []string `json:"sites"`
}

// LoadConfig читает файл по указанному пути и возвращает готовую структуру
func LoadConfig(path string) (*Config, error) {
	// 1. Открываем файл
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close() // Не забываем закрыть файл в конце

	// 2. Создаем переменную нашей структуры
	var cfg Config

	// 3. Декодируем JSON прямо в структуру
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

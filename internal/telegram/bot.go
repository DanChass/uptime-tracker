package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SendAlert отправляет сообщение в указанный чат с ограничением по времени
func SendAlert(token string, chatID int64, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Создаем кастомный клиент. Теперь никакой сбой сети не повесит наше приложение!
	client := &http.Client{
		Timeout: 3 * time.Second, // 5 секунд на ответ, дальше - жесткий сброс
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram api вернул статус: %d", resp.StatusCode)
	}

	return nil
}

package api

import (
	"encoding/json"
	"net/http"

	"github.com/DanChass/uptime-tracker/internal/database"
)

type Server struct {
	db *database.DB
}

func NewServer(db *database.DB) *Server {
	return &Server{db: db}
}

// Start запускает HTTP сервер
func (s *Server) Start(port string) error {
	// Указываем, какая функция сработает при переходе на /logs
	http.HandleFunc("/logs", s.handleGetLogs)
	return http.ListenAndServe(port, nil)
}

// handleGetLogs обрабатывает GET-запросы и отдает JSON
func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Берем последние 15 записей из базы
	logs, err := s.db.GetRecentLogs(15)
	if err != nil {
		http.Error(w, "Ошибка чтения из БД", http.StatusInternalServerError)
		return
	}

	// Говорим браузеру, что возвращаем JSON
	w.Header().Set("Content-Type", "application/json")

	// Конвертируем наш слайс структур в JSON и отправляем пользователю
	json.NewEncoder(w).Encode(logs)
}

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

// SetupServer теперь не блокирует поток, а возвращает настроенный объект сервера
func (s *Server) SetupServer(port string) *http.Server {
	// Создаем роутер, чтобы маршруты не конфликтовали с другими пакетами
	mux := http.NewServeMux()
	mux.HandleFunc("/logs", s.handleGetLogs)

	// Возвращаем сам сервер, чтобы в main.go мы могли им управлять (запускать и останавливать)
	return &http.Server{
		Addr:    port,
		Handler: mux,
	}
}

func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	logs, err := s.db.GetRecentLogs(15)
	if err != nil {
		http.Error(w, "Ошибка чтения из БД", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

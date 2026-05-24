package database

import (
	"database/sql"
	"time"

	"github.com/DanChass/uptime-tracker/internal/models"
	_ "modernc.org/sqlite" // Анонимный импорт драйвера
)

// DB обертка над подключением
type DB struct {
	conn *sql.DB
}

// InitDB открывает файл (или создает его) и накатывает таблицу
func InitDB(filepath string) (*DB, error) {
	dbConn, err := sql.Open("sqlite", filepath)
	if err != nil {
		return nil, err
	}

	if err = dbConn.Ping(); err != nil {
		return nil, err
	}

	// Обычный SQL, как в Postgres
	query := `
	CREATE TABLE IF NOT EXISTS check_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL,
		status_code INTEGER,
		response_time_ms INTEGER,
		is_up BOOLEAN,
		ssl_error BOOLEAN,
		checked_at DATETIME
	);`

	_, err = dbConn.Exec(query)
	if err != nil {
		return nil, err
	}

	return &DB{conn: dbConn}, nil
}

// Close закрывает базу
func (d *DB) Close() error {
	return d.conn.Close()
}

// SaveResult пишет лог в таблицу
func (d *DB) SaveResult(res models.CheckResult) error {
	query := `
	INSERT INTO check_logs (url, status_code, response_time_ms, is_up, ssl_error, checked_at)
	VALUES (?, ?, ?, ?, ?, ?);`

	ms := res.ResponseTime.Milliseconds()
	_, err := d.conn.Exec(query, res.URL, res.StatusCode, ms, res.IsUp, res.SSLError, res.CheckedAt)
	return err
}

// GetRecentLogs возвращает последние N записей из базы
func (d *DB) GetRecentLogs(limit int) ([]models.CheckResult, error) {
	// Делаем SQL-запрос с сортировкой по убыванию ID
	query := `
	SELECT url, status_code, response_time_ms, is_up, ssl_error, checked_at 
	FROM check_logs 
	ORDER BY id DESC LIMIT ?`

	rows, err := d.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.CheckResult

	// Построчно читаем ответ от базы
	for rows.Next() {
		var res models.CheckResult
		var ms int64

		err := rows.Scan(&res.URL, &res.StatusCode, &ms, &res.IsUp, &res.SSLError, &res.CheckedAt)
		if err != nil {
			return nil, err
		}

		// Переводим миллисекунды обратно в тип time.Duration
		res.ResponseTime = time.Duration(ms) * time.Millisecond
		logs = append(logs, res)
	}

	return logs, nil
}

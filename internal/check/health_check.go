package check

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/memcached"
)

// JSONResponse записывает ответ в формате JSON
func JSONResponse(w http.ResponseWriter, statusCode int, status string) {
	response := struct {
		Message string `json:"message"`
	}{
		Message: status,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
	}
}

// HealthCheckHandler — обработка health-check запроса с проверкой memcached и таймингом
func HealthCheckHandler(cache *memcached.Cache, log *logger.Logger, maxResponseTime time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			elapsed := time.Since(start)
			if elapsed > maxResponseTime {
				log.Error("Health check timed out")
				JSONResponse(w, http.StatusInternalServerError, "NOT OK")
				return
			}

			// Проверка доступности memcached
			if err := cache.IsEnabled(); err != nil {
				log.Error("Ошибка проверки активности кэша:", err)
				JSONResponse(w, http.StatusInternalServerError, "NOT OK")
				return
			}

			if err := cache.Ping(); err != nil {
				log.Error("Ошибка проверки доступности кэша:", err)
				JSONResponse(w, http.StatusInternalServerError, "NOT OK")
				return
			}

			JSONResponse(w, http.StatusOK, "OK")
		}()
	}
}

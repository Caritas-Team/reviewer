package handler

import (
	"context"

	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/usecase/user"
	"github.com/google/uuid"
	"github.com/rs/cors"
)

var (
	DefaultCORSMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	DefaultCORSHeaders = []string{"Content-Type", "Authorization"}
)

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAgeSeconds    int
}

func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	if len(cfg.AllowedMethods) == 0 {
		cfg.AllowedMethods = DefaultCORSMethods
	}
	if len(cfg.AllowedHeaders) == 0 {
		cfg.AllowedHeaders = DefaultCORSHeaders
	}
	if len(cfg.AllowedOrigins) == 0 {
		c := cors.AllowAll()
		return func(next http.Handler) http.Handler { return c.Handler(next) }
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   cfg.AllowedMethods,
		AllowedHeaders:   cfg.AllowedHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAgeSeconds,
	})
	return func(next http.Handler) http.Handler { return c.Handler(next) }
}

type RateLimiterMiddleware struct {
	limiter *user.RateLimiter
}

func NewRateLimiterMiddleware(limiter *user.RateLimiter) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		limiter: limiter,
	}
}

func (m *RateLimiterMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		if host, _, err := net.SplitHostPort(ip); err == nil {
			ip = host
		}

		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ips := strings.Split(forwarded, ",")
			if len(ips) > 0 {
				ip = strings.TrimSpace(ips[0])
			}
		}

		err := m.limiter.AllowRequest(r.Context(), ip)
		if err != nil {
			http.Error(w, "Слишком много запросов", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware добавляет идентификаторы запросов и логирование
func LoggingMiddleware(log *logger.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		requestID := uuid.New().String()
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		ctx := r.Context()
		ctx = logger.WithRequestID(ctx, requestID)
		ctx = logger.WithTraceID(ctx, traceID)

		wrappedWriter := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		log.WithContext(ctx).Info("request started",
			"method", r.Method,
			"path", r.URL.Path,
			"user_agent", r.UserAgent(),
			"remote_addr", r.RemoteAddr,
		)

		r = r.WithContext(ctx)
		next.ServeHTTP(wrappedWriter, r)

		duration := time.Since(start)
		log.WithContext(ctx).Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrappedWriter.statusCode,
			"duration_ms", duration.Milliseconds(),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Проверки
// CheckCORS проверяет, что CORS настроен корректно
func CheckCORS(config CORSConfig) error {
	origin := "https://example.com" // Тестовый источник

	// Формируем OPTIONS-запрос с нужным заголовком Origin
	req, err := http.NewRequest(http.MethodOptions, "http://localhost:8080/", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Origin", origin)

	// Выполняем запрос
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	// Закрываем тело ответа с проверкой ошибки
	if err := resp.Body.Close(); err != nil {
		return err
	}

	// Проверяем, содержит ли ответ правильный заголовок Access-Control-Allow-Origin
	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin == "*" || allowOrigin == origin {
		return http.ErrNoCookie
	}
	return nil
}

// Метод проверки работоспособности rate limiting
func (m *RateLimiterMiddleware) IsOperational() error {
	// Проверяем, может ли rate limiter разрешить хотя бы один запрос
	ip := "test_ip"
	err := m.limiter.AllowRequest(context.Background(), ip)
	return err
}

package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Caritas-Team/reviewer/internal/check"
	"github.com/Caritas-Team/reviewer/internal/config"
	"github.com/Caritas-Team/reviewer/internal/handler"
	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/memcached"
	"github.com/Caritas-Team/reviewer/internal/metrics"
	"github.com/Caritas-Team/reviewer/internal/model"
	"github.com/Caritas-Team/reviewer/internal/usecase/assessment"
	"github.com/Caritas-Team/reviewer/internal/usecase/user"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"

	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func main() {
	// Конфиг
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load error", "err", err)
		return
	}

	// Базовый контекст
	ctx := context.Background()

	// Глобальный логер
	log := logger.NewLogger(cfg)

	// Jaeger
	shutdownTracer, err := initTracer(ctx, "reviewer", cfg.Jaeger.Endpoint)
	if err != nil {
		slog.Error("tracer init error", "err", err)
		return
	}
	defer func() {
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracer(shCtx); err != nil {
			slog.Error("tracer shutdown error", "err", err)
		}
	}()

	// Контекст, отменяемый по SIGINT/SIGTERM
	rootCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Кэш
	cache, err := memcached.NewCache(rootCtx, cfg)
	if err != nil {
		log.Error("cache initialization failed", "err", err)
		return
	}

	rateLimiter := user.NewRateLimiter(cache, cfg)
	rateLimiterMiddleware := handler.NewRateLimiterMiddleware(rateLimiter)

	// Экземпляр ReadinessChecker
	checker := check.NewReadinessChecker(cache, rateLimiterMiddleware, log)

	// Хранилище результатов проверки
	resultStorage := assessment.NewResultStorage(cache)

	// Создаем канал для обработки
	inputChan := make(chan []model.StudentPair, cfg.Pipeline.InputBufferSize)

	// Создаем обработчик загрузки
	uploadHandler := handler.NewUploadHandler(cfg, log, cache, resultStorage, inputChan)

	// Worker (пишет processing/completed/failed в ResultStorage)
	processor := assessment.NewProcessor(&assessment.DiffCalculator{})
	w := assessment.NewWorker(log, resultStorage, processor, time.Hour)
	go w.Run(rootCtx, inputChan)

	mux := http.NewServeMux()
	mux.Handle("POST /v1/assessments/upload",
		http.HandlerFunc(uploadHandler.UploadAssessmentsHandler))

	// Эндпоинт для получения результатов обработки (GET /v1/assessments/{request_id})
	mux.Handle("GET /v1/assessments/{request_id}",
		handler.GetAssessmentResultsHandler(resultStorage, log))

	mux.HandleFunc("/v1/assessments/", handler.GetAssessmentResultsHandler(resultStorage, log))

	// Эндпоинт для health check
	mux.HandleFunc("/health", check.HealthCheckHandler(cache, log, 29*time.Second)) // Тайминг можно настроить

	// Эндпоинт для readiness check
	mux.HandleFunc("/ready", check.ReadinessCheckHandler(checker))

	// Метрики
	metrics.InitMetricsOn(mux)

	// CORS
	h := handler.CORS(handler.CORSConfig{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
		AllowCredentials: true,
		MaxAgeSeconds:    3600,
	})(mux)

	h = rateLimiterMiddleware.Handler(h)
	h = handler.LoggingMiddleware(log, h)

	h = otelhttp.NewHandler(h, "http-server")

	// HTTP сервер
	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      h,
		ReadTimeout:  cfg.Server.ReadTimeout(),
		WriteTimeout: cfg.Server.WriteTimeout(),
		IdleTimeout:  5 * time.Minute,
	}

	// Запуск сервера
	errCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", srv.Addr, "pid", os.Getpid())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	// Сигнал или ошибка сервера
	select {
	case <-rootCtx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			log.Error("server failed", "err", err)
			_ = cache.Close()
			return
		}
	}

	// Graceful shutdown
	graceTime := 30 * time.Second
	shCtx, cancel := context.WithTimeout(ctx, graceTime)
	defer cancel()

	if err := srv.Shutdown(shCtx); err != nil {
		log.Error("http shutdown error", "err", err)
	} else {
		log.Info("http server shutdown complete")
	}

	if err := cache.Close(); err != nil {
		log.Error("cache close error", "err", err)
	} else {
		log.Info("cache closed")
	}

	log.Info("graceful shutdown finished")

}

func initTracer(ctx context.Context, serviceName, endpoint string) (func(context.Context) error, error) {
	if endpoint == "" {
		endpoint = "jaeger:4318"
	}

	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)

	// Экспортёр поверх клиента
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, err
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName(serviceName),
			),
		),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return tp.Shutdown, nil
}

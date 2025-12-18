package processing

import (
	"context"
	"sync"
	"time"

	"github.com/Caritas-Team/reviewer/internal/config"
	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/model"
	"github.com/Caritas-Team/reviewer/internal/storage"
)

// PipelineConfig конфигурация пайплайна
type PipelineConfig struct {
	NumWorkers    int // Количество воркеров
	Log           *logger.Logger
	Config        config.Config
	ResultStorage *storage.ResultStorage
}

// Pipeline структура пайплайна
type Pipeline struct {
	inputChan  chan []model.StudentPair
	resultChan chan model.ProcessingResult
	errorChan  chan model.ProcessingError
	workerPool *WorkerPool
	wg         sync.WaitGroup
	ctx        context.Context
	cancelFunc context.CancelFunc
	log        *logger.Logger
	cfg        PipelineConfig
}

// NewPipeline создаёт новый пайплайн
func NewPipeline(inputChan chan []model.StudentPair, resultChan chan model.ProcessingResult, errorChan chan model.ProcessingError, cfg PipelineConfig) *Pipeline {
	return &Pipeline{
		inputChan:  inputChan,
		resultChan: resultChan,
		errorChan:  errorChan,
		log:        cfg.Log,
		cfg:        cfg,
	}
}

// Start запускает пайплайн
func (pl *Pipeline) Start(ctx context.Context) {
	pl.ctx, pl.cancelFunc = context.WithCancel(ctx)

	// Инициализируем воркер-пул
	pl.workerPool = NewWorkerPool(pl.ctx, pl.inputChan, pl.resultChan, pl.errorChan, pl.cfg.NumWorkers, pl.log)
	pl.workerPool.Start()

	// Запускаем агрегатор
	go ResultAggregator(pl.resultChan, pl.errorChan, pl.cfg.ResultStorage, pl.log)

	// Запускаем очистку устаревших записей
	go pl.cfg.ResultStorage.CleanupTicker(5 * time.Minute)

	pl.log.Info("Pipeline started")
}

// Stop останавливает пайплайн
func (pl *Pipeline) Stop() {
	pl.cancelFunc()
	pl.workerPool.Stop()
	pl.wg.Wait()
	pl.log.Info("Pipeline stopped")
}

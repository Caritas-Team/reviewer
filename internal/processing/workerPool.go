package processing

import (
	"context"
	"fmt"
	"sync"

	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/model"
)

// WorkerPool управляет группой воркеров
type WorkerPool struct {
	inputChan  chan []model.StudentPair
	resultChan chan model.ProcessingResult
	errorChan  chan model.ProcessingError
	numWorkers int
	wg         sync.WaitGroup
	ctx        context.Context
	cancelFunc context.CancelFunc
	logger     *logger.Logger // Экземпляр логгера
}

// Новый воркер-пул
func NewWorkerPool(
	ctx context.Context,
	inputChan chan []model.StudentPair,
	resultChan chan model.ProcessingResult,
	errorChan chan model.ProcessingError,
	numWorkers int,
	lgr *logger.Logger,
) *WorkerPool {
	poolCtx, cancel := context.WithCancel(ctx)
	return &WorkerPool{
		inputChan:  inputChan,
		resultChan: resultChan,
		errorChan:  errorChan,
		numWorkers: numWorkers,
		wg:         sync.WaitGroup{},
		ctx:        poolCtx,
		cancelFunc: cancel,
		logger:     lgr,
	}
}

// Запуск воркер-пула
func (pool *WorkerPool) Start() {
	for i := 0; i < pool.numWorkers; i++ {
		pool.wg.Add(1)
		go pool.worker(i + 1)
	}
	pool.logger.Info("Worker pool started with %d workers.", pool.numWorkers)
}

// Остановка воркер-пула
func (pool *WorkerPool) Stop() {
	pool.cancelFunc()
	pool.wg.Wait()
	pool.logger.Info("Worker pool stopped.")
}

// worker — внутренний рабочий цикл воркера
func (pool *WorkerPool) worker(id int) {
	defer pool.wg.Done()

	for {
		select {
		case pairs, ok := <-pool.inputChan:
			if !ok {
				pool.logger.Debug("Worker %d: input channel closed.", id)
				return // Канал закрыт
			}

			// Обработка пакета пар документов
			for _, pair := range pairs {
				// Вызвать обработку пары документов
				result, err := pool.processPair(pair)
				if err != nil {
					// Создаем ошибку и отправляем в канал ошибок
					errorMsg := model.ProcessingError{
						RequestID: pair.RequestID,
						StudentID: pair.StudentID, // Добавляем StudentID
						Message:   err.Error(),
					}
					pool.errorChan <- errorMsg
					pool.logger.Error("Worker %d encountered an error during processing: %v", id, err)
				} else {
					pool.resultChan <- result
					pool.logger.Debug("Worker %d successfully processed pair %s.", id, pair.RequestID)
				}
			}

		case <-pool.ctx.Done():
			pool.logger.Debug("Worker %d stopping.", id)
			return
		}
	}
}

// Внутренняя функция обработки пары документов
func (pool *WorkerPool) processPair(pair model.StudentPair) (model.ProcessingResult, error) {
	// Этапы обработки:
	// 1. Проверка наличия документов
	if pair.Before == nil || pair.After == nil {
		return model.ProcessingResult{}, fmt.Errorf("one of documents is missing")
	}

	// 2. Расчет изменений
	diffCalc := DiffCalculator{} // Создаем экземпляр калькулятора
	diff, err := diffCalc.Calculate(pair.Before, pair.After)
	if err != nil {
		return model.ProcessingResult{}, fmt.Errorf("diff calculation error: %w", err)
	}

	// Формирование результата
	result := model.ProcessingResult{
		RequestID:     pair.RequestID,
		Status:        "processing",
		ResultDetails: map[string]interface{}{"diff": diff},
	}

	return result, nil
}

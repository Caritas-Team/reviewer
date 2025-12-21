package processing

import (
	"fmt"
	"time"

	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/model"
	"github.com/Caritas-Team/reviewer/internal/storage"
)

// ResultAggregator собирает результаты и ошибки
func ResultAggregator(
	resultChan <-chan model.ProcessingResult,
	errorChan <-chan model.ProcessingError,
	resultStorage *storage.ResultStorage,
	log *logger.Logger,
) {
	for {
		select {
		case result := <-resultChan:
			currentResult, exists := resultStorage.Get(result.RequestID)

			if exists {
				updateAggregatedResult(&currentResult, result)
				resultStorage.Set(result.RequestID, currentResult)
			} else {
				if result.ResultDetails == nil {
					result.ResultDetails = map[string]interface{}{}
				}
				normalizeDiffKeys(&result)
				result.CreatedAt = time.Now()
				resultStorage.Set(result.RequestID, result)
			}

			checkCompletion(resultStorage, result.RequestID, log)

		case err := <-errorChan:
			processError(resultStorage, err, log)
		}
	}
}

// updateAggregatedResult обновляет совокупный результат
func updateAggregatedResult(current *model.ProcessingResult, newResult model.ProcessingResult) {
	current.ProcessedStudents += newResult.ProcessedStudents

	if current.ResultDetails == nil {
		current.ResultDetails = map[string]interface{}{}
	}
	normalizeDiffKeys(current)

	payload := interface{}(newResult.ResultDetails)
	if newResult.ResultDetails != nil {
		if v, ok := newResult.ResultDetails["diff"]; ok {
			payload = v
		}
	}

	next := nextDiffIndex(current.ResultDetails)
	key := fmt.Sprintf("diff%d", next)
	current.ResultDetails[key] = payload

	// Если вдруг TotalStudents не был установлен в первой записи
	if current.TotalStudents == 0 {
		current.TotalStudents = newResult.TotalStudents
	}
}

// checkCompletion проверяет, завершил ли запрос обработку
func checkCompletion(resultStorage *storage.ResultStorage, requestID string, log *logger.Logger) {
	result, exists := resultStorage.Get(requestID)
	if exists {
		if result.ProcessedStudents == result.TotalStudents {
			// Все пары обработаны, проверяем наличие ошибок
			if len(result.Errors) > 0 {
				// Если есть ошибки, меняем статус на "failed"
				resultStorage.UpdateStatus(requestID, "failed")
				log.Error("Request %s failed due to previous errors", requestID)
			} else {
				// Если ошибок нет, меняем статус на "completed"
				resultStorage.UpdateStatus(requestID, "completed")
				log.Info("Request %s completed successfully", requestID)
			}
		}
	}
}

// processError обрабатывает ошибку
func processError(resultStorage *storage.ResultStorage, err model.ProcessingError, log *logger.Logger) {
	// Логируем ошибку
	log.Error("Error during processing: %s", err.Message)

	// Получаем текущий результат по request_id
	currentResult, exists := resultStorage.Get(err.RequestID)

	if exists {
		// Добавляем ошибку в список ошибок
		currentResult.Errors[err.StudentID] = err.Message // Правильно используем карту
		resultStorage.Set(err.RequestID, currentResult)
		log.Error("Adding error to list for request %s", err.RequestID)
	} else {
		// Если результат не найден, создаём новый с ошибкой
		failedResult := model.ProcessingResult{
			RequestID: err.RequestID,
			Status:    "processing",
			Errors:    map[string]string{err.StudentID: err.Message}, // Правильно инициализируем карту
			CreatedAt: time.Now(),
		}
		resultStorage.Set(err.RequestID, failedResult)
		log.Error("Creating new result with error for request %s", err.RequestID)
	}
}

func normalizeDiffKeys(r *model.ProcessingResult) {
	if r.ResultDetails == nil {
		r.ResultDetails = map[string]interface{}{}
		return
	}
	if v, ok := r.ResultDetails["diff"]; ok {
		if _, exists := r.ResultDetails["diff1"]; !exists {
			r.ResultDetails["diff1"] = v
		}
		delete(r.ResultDetails, "diff")
	}
}

// Считает сколько уже есть diffN
// При наличии diff1 следующий - diff2
func nextDiffIndex(details map[string]interface{}) int {
	max := 0
	for k := range details {
		var n int
		if _, err := fmt.Sscanf(k, "diff%d", &n); err == nil && n > max {
			max = n
		}
	}
	return max + 1
}

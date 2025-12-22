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
				if err := resultStorage.Set(result.RequestID, currentResult); err != nil {
					log.Error("Failed to persist aggregated result for request %s: %v", result.RequestID, err)
					continue
				}
			} else {
				if result.ResultDetails == nil {
					result.ResultDetails = map[string]interface{}{}
				}
				normalizeDiffKeys(&result)
				result.CreatedAt = time.Now()
				if err := resultStorage.Set(result.RequestID, result); err != nil {
					log.Error("Failed to persist initial result for request %s: %v", result.RequestID, err)
					continue
				}
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

	if current.TotalStudents == 0 {
		current.TotalStudents = newResult.TotalStudents
	}
}

// checkCompletion проверяет, завершил ли запрос обработку
func checkCompletion(resultStorage *storage.ResultStorage, requestID string, log *logger.Logger) {
	result, exists := resultStorage.Get(requestID)
	if !exists {
		return
	}

	if result.ProcessedStudents != result.TotalStudents {
		return
	}

	if len(result.Errors) > 0 {
		if err := resultStorage.UpdateStatus(requestID, "failed"); err != nil {
			log.Error("Failed to update status to failed for request %s: %v", requestID, err)
			return
		}
		log.Error("Request %s failed due to previous errors", requestID)
		return
	}

	if err := resultStorage.UpdateStatus(requestID, "completed"); err != nil {
		log.Error("Failed to update status to completed for request %s: %v", requestID, err)
		return
	}
	log.Info("Request %s completed successfully", requestID)
}

// processError обрабатывает ошибку
func processError(resultStorage *storage.ResultStorage, err model.ProcessingError, log *logger.Logger) {
	log.Error("Error during processing: %s", err.Message)

	currentResult, exists := resultStorage.Get(err.RequestID)

	if exists {
		if currentResult.Errors == nil {
			currentResult.Errors = map[string]string{}
		}
		currentResult.Errors[err.StudentID] = err.Message

		if e := resultStorage.Set(err.RequestID, currentResult); e != nil {
			log.Error("Failed to persist error for request %s: %v", err.RequestID, e)
			return
		}
		log.Error("Adding error to list for request %s", err.RequestID)
		return
	}

	failedResult := model.ProcessingResult{
		RequestID: err.RequestID,
		Status:    "processing",
		Errors:    map[string]string{err.StudentID: err.Message},
		CreatedAt: time.Now(),
	}

	if e := resultStorage.Set(err.RequestID, failedResult); e != nil {
		log.Error("Failed to create new result with error for request %s: %v", err.RequestID, e)
		return
	}
	log.Error("Creating new result with error for request %s", err.RequestID)
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

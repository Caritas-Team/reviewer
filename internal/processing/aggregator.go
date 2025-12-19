package processing

import (
	"fmt"
	"time"

	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/model"
	"github.com/Caritas-Team/reviewer/internal/storage"
)

// ResultAggregator собирает результаты и ошибки
func ResultAggregator(resultChan <-chan model.ProcessingResult, errorChan <-chan model.ProcessingError, resultStorage *storage.ResultStorage, log *logger.Logger) {
	for {
		select {
		case result := <-resultChan:
			// Получаем результат и ищем запись по request_id
			currentResult, exists := resultStorage.Get(result.RequestID)

			if exists {
				// Если запись уже существует, обновляем её
				updateAggregatedResult(currentResult, result)
				resultStorage.Set(result.RequestID, currentResult)
			} else {
				// Иначе создаём новую запись
				result.CreatedAt = time.Now() // Добавляем время создания
				resultStorage.Set(result.RequestID, result)
			}

			// Проверяем, завершился ли запрос
			checkCompletion(resultStorage, result.RequestID, log)

		case err := <-errorChan:
			// Обработка ошибок
			processError(resultStorage, err, log)
		}
	}
}

// updateAggregatedResult обновляет совокупный результат
func updateAggregatedResult(currentResult, newResult model.ProcessingResult) {
	// Объединяем результаты
	currentResult.ProcessedStudents += newResult.ProcessedStudents

	// Генерация уникального ключа для нового результата
	key := fmt.Sprintf("diff%d", len(currentResult.ResultDetails)+1)

	// Добавляем новые данные под уникальным ключом
	currentResult.ResultDetails[key] = newResult.ResultDetails
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

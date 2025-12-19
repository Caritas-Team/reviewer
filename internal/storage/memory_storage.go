package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Caritas-Team/reviewer/internal/config"
	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/memcached"
	"github.com/Caritas-Team/reviewer/internal/model"
)

// ResultStorage - хранилище результатов с использованием мем-кеша
type ResultStorage struct {
	memcachedClient *memcached.Cache
	lock            sync.RWMutex
	results         map[string]model.ProcessingResult
	config          config.Config
	log             *logger.Logger
}

// NewResultStorage создаёт новое хранилище результатов
func NewResultStorage(config config.Config, log *logger.Logger) (*ResultStorage, error) {
	cache, err := memcached.NewCache(context.TODO(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize memcached: %w", err)
	}

	return &ResultStorage{
		memcachedClient: cache,
		results:         make(map[string]model.ProcessingResult),
		config:          config,
		log:             log,
	}, nil
}

// Set сохраняет результат по request_id
func (rs *ResultStorage) Set(requestID string, result model.ProcessingResult) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	// Сохраняем результат в локальном кеше
	rs.results[requestID] = result

	// Сохраняем результат в мем-кеше
	value, err := serialize(result)
	if err != nil {
		return fmt.Errorf("serialization error: %w", err)
	}

	err = rs.memcachedClient.Set(context.TODO(), requestID, value, 0)
	if err != nil {
		return fmt.Errorf("memcached set error: %w", err)
	}

	return nil
}

// Get получает результат по request_id
func (rs *ResultStorage) Get(requestID string) (model.ProcessingResult, bool) {
	rs.lock.RLock()
	defer rs.lock.RUnlock()

	// Сначала проверяем локальный кеш
	result, exists := rs.results[requestID]
	if exists {
		return result, true
	}

	// Если нет в локальном кеше, запрашиваем из мем-кеша
	value, err := rs.memcachedClient.Get(context.TODO(), requestID)
	if err != nil {
		if err == memcached.ErrCacheMiss {
			return model.ProcessingResult{}, false
		}
		return model.ProcessingResult{}, false
	}

	// Десериализуем результат
	result, err = deserialize(value)
	if err != nil {
		return model.ProcessingResult{}, false
	}

	// Обновляем локальный кеш
	rs.results[requestID] = result

	return result, true
}

// UpdateStatus обновляет статус результата
func (rs *ResultStorage) UpdateStatus(requestID, status string) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	if result, exists := rs.results[requestID]; exists {
		// Создаем копию результата с обновленным статусом
		updatedResult := result
		updatedResult.Status = status

		// Обновляем запись в карте
		rs.results[requestID] = updatedResult
		return nil
	}

	return fmt.Errorf("result not found")
}

// UpdateProgress обновляет прогресс обработки
func (rs *ResultStorage) UpdateProgress(requestID string, progress int) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	if result, exists := rs.results[requestID]; exists {
		result.ProcessedStudents += progress
		return nil
	}

	return fmt.Errorf("result not found")
}

// Delete удаляет результат по requestID
func (rs *ResultStorage) Delete(requestID string) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	delete(rs.results, requestID) // Удаляем без предварительной проверки

	err := rs.memcachedClient.Delete(context.TODO(), requestID)
	if err != nil {
		return fmt.Errorf("memcached delete error: %w", err)
	}

	return nil
}

// GetAll возвращает все результаты
func (rs *ResultStorage) GetAll() map[string]model.ProcessingResult {
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	return rs.results
}

// CleanupTicker запускает удаление устаревших записей
func (rs *ResultStorage) CleanupTicker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		rs.removeOldResults()
	}
}

// removeOldResults удаляет записи старше заданного срока
func (rs *ResultStorage) removeOldResults() {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	now := time.Now()
	for reqID, result := range rs.results {
		if result.CreatedAt.Add(1 * time.Hour).Before(now) {
			delete(rs.results, reqID)
			err := rs.memcachedClient.Delete(context.TODO(), reqID)
			if err != nil {
				rs.log.Error("Error deleting from memcached: %v", err)
			}
		}
	}
}

// serialize конвертирует результат в байтовый массив
func serialize(result model.ProcessingResult) ([]byte, error) {
	bytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("serialization error: %w", err)
	}
	return bytes, nil
}

// deserialize восстанавливает результат из байтового массива
func deserialize(data []byte) (model.ProcessingResult, error) {
	var result model.ProcessingResult
	err := json.Unmarshal(data, &result)
	if err != nil {
		return model.ProcessingResult{}, fmt.Errorf("deserialization error: %w", err)
	}
	return result, nil
}

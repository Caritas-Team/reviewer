//go:build integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Caritas-Team/reviewer/internal/config"
	"github.com/Caritas-Team/reviewer/internal/logger"
	memcachedclient "github.com/Caritas-Team/reviewer/internal/memcached"
	"github.com/Caritas-Team/reviewer/internal/model"
	"github.com/Caritas-Team/reviewer/internal/usecase/assessment"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/memcached"
)

// createTestJSON создает JSON для тестового файла
func createTestJSON(studentID, date string, control, preintentional float64, activeWords int) string {
	// Создаем словарь с нужным количеством активных слов
	dictItems := "["
	for i := 0; i < activeWords; i++ {
		if i > 0 {
			dictItems += ","
		}
		dictItems += fmt.Sprintf(`{"itemOffStyle": "", "content": "word%d"}`, i+1)
	}
	// Добавляем одно неактивное слово для полноты
	if activeWords > 0 {
		dictItems += `,{"itemOffStyle": "res-dict__item-off", "content": "inactive"}`
	}
	dictItems += "]"

	return fmt.Sprintf(`{
		"por01": "<div>%s</div>",
		"por02": "<div>%s</div>",
		"newAct01": {"procNumElem": "%.0f%%"},
		"newAct02": {"procNumElem": "5%%"},
		"newAct03": {"procNumElem": "3%%"},
		"newAct04": {"procNumElem": "2%%"},
		"diagramBlock": {
			"predActProcNumElem": "%.0f%%",
			"protActProcNumElem": "10%%",
			"protInitProcNumElem": "5%%",
			"golActProcNumElem": "15%%",
			"golInitProcNumElem": "20%%",
			"fraActProcNumElem": "20%%",
			"fraInitProcNumElem": "25%%"
		},
		"basicDictionary": %s,
		"dictBasicMore": []
	}`, date, studentID, control, preintentional, dictItems)
}

func TestIntegration_GroupAverages_ThreeStudents(t *testing.T) {
	ctx := context.Background()

	// Запускаем memcached через testcontainers
	memcachedContainer, err := memcached.Run(ctx, "memcached:1.6-alpine")
	require.NoError(t, err)
	defer func() {
		err := memcachedContainer.Terminate(ctx)
		if err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Получаем адрес memcached
	host, err := memcachedContainer.Host(ctx)
	require.NoError(t, err)
	port, err := memcachedContainer.MappedPort(ctx, "11211")
	require.NoError(t, err)

	memcachedAddr := fmt.Sprintf("%s:%s", host, port.Port())

	// Настраиваем конфиг
	cfg := config.Config{
		Files: config.Files{
			MaxFilesPerRequest: 20,
			MaxFileSize:        10 * 1024 * 1024,
		},
		Memcached: config.Memcached{
			Enable:     true,
			Servers:    []string{memcachedAddr},
			DefaultTTL: 3600,
			KeyPrefix:  "test",
		},
		Pipeline: config.PipelineConfig{
			InputBufferSize: 10,
		},
	}

	log := logger.NewLogger(cfg)

	// Подключаемся к memcached
	cache, err := memcachedclient.NewCache(ctx, cfg)
	require.NoError(t, err)
	defer func() {
		if err := cache.Close(); err != nil {
			t.Logf("Failed to close cache: %v", err)
		}
	}()

	// Создаем storage и worker
	resultStorage := assessment.NewResultStorage(cache)
	inputChan := make(chan []model.StudentPair, 10)
	uploadHandler := NewUploadHandler(cfg, log, cache, resultStorage, inputChan)

	// Запускаем worker
	processor := assessment.NewProcessor(&assessment.DiffCalculator{})
	worker := assessment.NewWorker(log, resultStorage, processor, time.Hour)
	go worker.Run(ctx, inputChan)

	requestID := uuid.New().String()

	// Создаем 6 файлов: 3 студента (A, B, C), по 2 файла на каждого (12 и 15 число)
	testFiles := []struct {
		filename    string
		studentID   string
		date        string
		control     float64
		preintent   float64
		activeWords int
	}{
		// Студент A
		{"studentA_2025-12-12.json", "A", "2025-12-12", 10.0, 50.0, 100},
		{"studentA_2025-12-15.json", "A", "2025-12-15", 15.0, 55.0, 110},
		// Студент B
		{"studentB_2025-12-12.json", "B", "2025-12-12", 20.0, 60.0, 120},
		{"studentB_2025-12-15.json", "B", "2025-12-15", 25.0, 65.0, 130},
		// Студент C
		{"studentC_2025-12-12.json", "C", "2025-12-12", 5.0, 40.0, 80},
		{"studentC_2025-12-15.json", "C", "2025-12-15", 10.0, 45.0, 90},
	}

	// Создаем multipart запрос
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, tf := range testFiles {
		jsonData := createTestJSON(tf.studentID, tf.date, tf.control, tf.preintent, tf.activeWords)
		part, err := writer.CreateFormFile("files", tf.filename)
		require.NoError(t, err)
		_, err = part.Write([]byte(jsonData))
		require.NoError(t, err)
	}

	require.NoError(t, writer.Close())

	// Загружаем файлы
	req := httptest.NewRequest("POST", "/v1/assessments/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Request-Id", requestID)

	rr := httptest.NewRecorder()
	uploadHandler.UploadAssessmentsHandler(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var uploadResponse map[string]any
	err = json.Unmarshal(rr.Body.Bytes(), &uploadResponse)
	require.NoError(t, err)
	assert.Equal(t, "processing", uploadResponse["status"])

	// Ждем обработки (максимум 30 секунд)
	deadline := time.Now().Add(30 * time.Second)
	var result *assessment.ProcessingResult

	for time.Now().Before(deadline) {
		res, err := resultStorage.Get(ctx, requestID)
		if err == nil && res.Status == "completed" {
			result = res
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	require.NotNil(t, result, "Result should be available")
	assert.Equal(t, "completed", result.Status)
	assert.Len(t, result.Results, 3, "Should have 3 student results")

	// Проверяем групповые средние
	require.NotNil(t, result.GroupAverages, "Group averages should be calculated")
	assert.Len(t, result.GroupAverages, 2, "Should have group averages for 2 dates")

	// Находим средние за 12 и 15 число
	var avgDec12, avgDec15 *assessment.GroupAverage
	for i := range result.GroupAverages {
		if result.GroupAverages[i].Date == "2025-12-12" {
			avgDec12 = &result.GroupAverages[i]
		}
		if result.GroupAverages[i].Date == "2025-12-15" {
			avgDec15 = &result.GroupAverages[i]
		}
	}

	require.NotNil(t, avgDec12, "Should have average for 2025-12-12")
	require.NotNil(t, avgDec15, "Should have average for 2025-12-15")

	// Проверяем средние за 12 число
	assert.Equal(t, 3, avgDec12.StudentsCount)
	// Control: (10+20+5)/3 = 11.67
	assert.InDelta(t, 11.67, avgDec12.CommunicativeFuncs.Control, 0.1)
	// Preintentional: (50+60+40)/3 = 50.0
	assert.InDelta(t, 50.0, avgDec12.LanguageLevels.Preintentional.Activity, 0.1)
	// ActiveWordsCount: (100+120+80)/3 = 100
	assert.Equal(t, 100, avgDec12.Vocabulary.ActiveWordsCount)

	// Проверяем средние за 15 число
	assert.Equal(t, 3, avgDec15.StudentsCount)
	// Control: (15+25+10)/3 = 16.67
	assert.InDelta(t, 16.67, avgDec15.CommunicativeFuncs.Control, 0.1)
	// Preintentional: (55+65+45)/3 = 55.0
	assert.InDelta(t, 55.0, avgDec15.LanguageLevels.Preintentional.Activity, 0.1)
	// ActiveWordsCount: (110+130+90)/3 = 110
	assert.Equal(t, 110, avgDec15.Vocabulary.ActiveWordsCount)

	// Проверяем, что даты отсортированы
	assert.Equal(t, "2025-12-12", result.GroupAverages[0].Date)
	assert.Equal(t, "2025-12-15", result.GroupAverages[1].Date)
}

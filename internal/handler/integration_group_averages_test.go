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
func createTestJSON(studentID, date string, control, preintentional float64, activeWords int, protolanguage, holophrase, phrase float64) string {
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
            "protActProcNumElem": "%.0f%%",
            "protInitProcNumElem": "5%%",
            "golActProcNumElem": "%.0f%%",
            "golInitProcNumElem": "20%%",
            "fraActProcNumElem": "%.0f%%",
            "fraInitProcNumElem": "25%%"
        },
        "basicDictionary": %s,
        "dictBasicMore": []
    }`, date, studentID, control, preintentional, protolanguage, holophrase, phrase, dictItems)
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
		protolang   float64
		holophrase  float64
		phrase      float64
		activeWords int
	}{
		// Студент A - 12 декабря
		{"studentA_2025-12-12.json", "A", "2025-12-12", 10.0, 50.0, 30.0, 40.0, 60.0, 100},
		// Студент A - 15 декабря
		{"studentA_2025-12-15.json", "A", "2025-12-15", 15.0, 55.0, 35.0, 45.0, 65.0, 110},
		// Студент B - 12 декабря
		{"studentB_2025-12-12.json", "B", "2025-12-12", 20.0, 60.0, 40.0, 50.0, 70.0, 120},
		// Студент B - 15 декабря
		{"studentB_2025-12-15.json", "B", "2025-12-15", 25.0, 65.0, 45.0, 55.0, 75.0, 130},
		// Студент C - 12 декабря
		{"studentC_2025-12-12.json", "C", "2025-12-12", 5.0, 40.0, 20.0, 30.0, 50.0, 80},
		// Студент C - 15 декабря
		{"studentC_2025-12-15.json", "C", "2025-12-15", 10.0, 45.0, 25.0, 35.0, 55.0, 90},
	}

	// Создаем multipart запрос
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, tf := range testFiles {
		jsonData := createTestJSON(tf.studentID, tf.date, tf.control, tf.preintent,
			tf.activeWords, tf.protolang, tf.holophrase, tf.phrase)
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

	// Проверяем прогресс группы
	require.NotNil(t, result.GroupProgress, "Group progress should be calculated")
	assert.Len(t, result.GroupProgress, 1, "Should have progress between 2 dates")

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
	// Protolanguage: (30+40+20)/3 = 30.0
	assert.InDelta(t, 30.0, avgDec12.LanguageLevels.Protolanguage.Activity, 0.1)
	// Holophrase: (40+50+30)/3 = 40.0
	assert.InDelta(t, 40.0, avgDec12.LanguageLevels.Holophrase.Activity, 0.1)
	// Phrase: (60+70+50)/3 = 60.0
	assert.InDelta(t, 60.0, avgDec12.LanguageLevels.Phrase.Activity, 0.1)

	// Проверяем средние за 15 число
	assert.Equal(t, 3, avgDec15.StudentsCount)
	// Control: (15+25+10)/3 = 16.67
	assert.InDelta(t, 16.67, avgDec15.CommunicativeFuncs.Control, 0.1)
	// Preintentional: (55+65+45)/3 = 55.0
	assert.InDelta(t, 55.0, avgDec15.LanguageLevels.Preintentional.Activity, 0.1)
	// Protolanguage: (35+45+25)/3 = 35.0
	assert.InDelta(t, 35.0, avgDec15.LanguageLevels.Protolanguage.Activity, 0.1)
	// Holophrase: (45+55+35)/3 = 45.0
	assert.InDelta(t, 45.0, avgDec15.LanguageLevels.Holophrase.Activity, 0.1)
	// Phrase: (65+75+55)/3 = 65.0
	assert.InDelta(t, 65.0, avgDec15.LanguageLevels.Phrase.Activity, 0.1)

	// Проверяем, что даты отсортированы
	assert.Equal(t, "2025-12-12", result.GroupAverages[0].Date)
	assert.Equal(t, "2025-12-15", result.GroupAverages[1].Date)

	// Проверяем прогресс группы
	require.NotNil(t, result.GroupProgress, "Group progress should be calculated")
	assert.Len(t, result.GroupProgress, 1, "Should have progress between 2 dates")

	if len(result.GroupProgress) > 0 {
		progress := result.GroupProgress[0]
		assert.Equal(t, "2025-12-12", progress.PeriodStart)
		assert.Equal(t, "2025-12-15", progress.PeriodEnd)

		// Проверяем расчет процентов:
		// Preintentional: ((55-50)/50)*100 = 10%
		assert.InDelta(t, 10.0, progress.LanguageLevels.Preintentional.ActivityPercent, 0.1)
		// Protolanguage: ((35-30)/30)*100 = 16.67%
		assert.InDelta(t, 16.67, progress.LanguageLevels.Protolanguage.ActivityPercent, 0.1)
		// Holophrase: ((45-40)/40)*100 = 12.5%
		assert.InDelta(t, 12.5, progress.LanguageLevels.Holophrase.ActivityPercent, 0.1)
		// Phrase: ((65-60)/60)*100 = 8.33%
		assert.InDelta(t, 8.33, progress.LanguageLevels.Phrase.ActivityPercent, 0.1)
	}
}
func TestIntegration_APIResponseContainsGroupProgress(t *testing.T) {
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
		protolang   float64
		holophrase  float64
		phrase      float64
		activeWords int
	}{
		// Студент A
		{"studentA_2025-12-12.json", "A", "2025-12-12", 10.0, 50.0, 30.0, 40.0, 60.0, 100},
		{"studentA_2025-12-15.json", "A", "2025-12-15", 15.0, 55.0, 35.0, 45.0, 65.0, 110},
		// Студент B
		{"studentB_2025-12-12.json", "B", "2025-12-12", 20.0, 60.0, 40.0, 50.0, 70.0, 120},
		{"studentB_2025-12-15.json", "B", "2025-12-15", 25.0, 65.0, 45.0, 55.0, 75.0, 130},
		// Студент C
		{"studentC_2025-12-12.json", "C", "2025-12-12", 5.0, 40.0, 20.0, 30.0, 50.0, 80},
		{"studentC_2025-12-15.json", "C", "2025-12-15", 10.0, 45.0, 25.0, 35.0, 55.0, 90},
	}

	// Создаем multipart запрос
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, tf := range testFiles {
		jsonData := createTestJSON(tf.studentID, tf.date, tf.control, tf.preintent,
			tf.activeWords, tf.protolang, tf.holophrase, tf.phrase)
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

	// Теперь проверяем API ответ через GET endpoint
	getReq := httptest.NewRequest("GET", "/v1/assessments/"+requestID+"?keep_in_cache=true", nil)
	getReq.Header.Set("Content-Type", "application/json")

	getRR := httptest.NewRecorder()

	// Используем GetAssessmentResultsHandler
	handler := GetAssessmentResultsHandler(resultStorage, log)
	handler.ServeHTTP(getRR, getReq)

	assert.Equal(t, http.StatusOK, getRR.Code, "GET request should return 200 OK")

	var apiResponse map[string]interface{}
	err = json.Unmarshal(getRR.Body.Bytes(), &apiResponse)
	require.NoError(t, err, "Should parse JSON response")

	// Проверяем структуру ответа
	assert.Equal(t, "completed", apiResponse["status"], "Status should be 'completed'")

	// Проверяем наличие group_progress
	groupProgress, exists := apiResponse["group_progress"]
	require.True(t, exists, "API response should contain group_progress field")

	groupProgressArray, ok := groupProgress.([]interface{})
	require.True(t, ok, "group_progress should be an array")
	assert.Len(t, groupProgressArray, 1, "Should have one group progress entry")

	// Проверяем структуру прогресса
	progressObj := groupProgressArray[0].(map[string]interface{})
	assert.Equal(t, "2025-12-12", progressObj["period_start"])
	assert.Equal(t, "2025-12-15", progressObj["period_end"])

	// Проверяем language_levels
	languageLevels, exists := progressObj["language_levels"].(map[string]interface{})
	require.True(t, exists, "Should have language_levels")

	// Вспомогательная функция для проверки уровня
	checkLevel := func(levelName string, expectedPercent float64) {
		level, exists := languageLevels[levelName].(map[string]interface{})
		require.True(t, exists, "Should have %s level", levelName)

		activityPercent, exists := level["activity_percent"].(float64)
		require.True(t, exists, "%s should have activity_percent", levelName)

		assert.InDelta(t, expectedPercent, activityPercent, 0.1,
			"%s activity_percent should be %.2f%%", levelName, expectedPercent)
	}

	// Проверяем все четыре параметра
	checkLevel("preintentional", 10.0)
	checkLevel("protolanguage", 16.67)
	checkLevel("holophrase", 12.5)
	checkLevel("phrase", 8.33)

	// Также проверяем что group_averages присутствует
	groupAverages, exists := apiResponse["group_averages"]
	require.True(t, exists, "API response should contain group_averages field")

	groupAveragesArray, ok := groupAverages.([]interface{})
	require.True(t, ok, "group_averages should be an array")
	assert.Len(t, groupAveragesArray, 2, "Should have two group averages")

	// Проверяем что results присутствует
	results, exists := apiResponse["results"]
	require.True(t, exists, "API response should contain results field")

	resultsArray, ok := results.([]interface{})
	require.True(t, ok, "results should be an array")
	assert.Len(t, resultsArray, 3, "Should have three student results")
}
func TestIntegration_GroupProgress_NegativeValues(t *testing.T) {
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

	// Создаем тестовые данные где показатели ухудшаются со временем
	testFiles := []struct {
		filename    string
		studentID   string
		date        string
		control     float64
		preintent   float64
		protolang   float64
		holophrase  float64
		phrase      float64
		activeWords int
	}{
		// 12 декабря - хорошие показатели
		{"studentA_2025-12-12.json", "A", "2025-12-12", 10.0, 50.0, 30.0, 40.0, 60.0, 100},
		{"studentB_2025-12-12.json", "B", "2025-12-12", 20.0, 60.0, 40.0, 50.0, 70.0, 120},
		{"studentC_2025-12-12.json", "C", "2025-12-12", 5.0, 40.0, 20.0, 30.0, 50.0, 80},

		// 15 декабря - показатели ухудшились у всех студентов
		{"studentA_2025-12-15.json", "A", "2025-12-15", 8.0, 45.0, 25.0, 35.0, 55.0, 90},
		{"studentB_2025-12-15.json", "B", "2025-12-15", 15.0, 55.0, 35.0, 45.0, 65.0, 110},
		{"studentC_2025-12-15.json", "C", "2025-12-15", 3.0, 35.0, 15.0, 25.0, 45.0, 70},
	}

	// Создаем multipart запрос
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, tf := range testFiles {
		jsonData := createTestJSON(tf.studentID, tf.date, tf.control, tf.preintent,
			tf.activeWords, tf.protolang, tf.holophrase, tf.phrase)
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
	// Preintentional: (50+60+40)/3 = 50.0
	assert.InDelta(t, 50.0, avgDec12.LanguageLevels.Preintentional.Activity, 0.1)
	// Protolanguage: (30+40+20)/3 = 30.0
	assert.InDelta(t, 30.0, avgDec12.LanguageLevels.Protolanguage.Activity, 0.1)
	// Holophrase: (40+50+30)/3 = 40.0
	assert.InDelta(t, 40.0, avgDec12.LanguageLevels.Holophrase.Activity, 0.1)
	// Phrase: (60+70+50)/3 = 60.0
	assert.InDelta(t, 60.0, avgDec12.LanguageLevels.Phrase.Activity, 0.1)

	// Проверяем средние за 15 число (ухудшение)
	assert.Equal(t, 3, avgDec15.StudentsCount)
	// Preintentional: (45+55+35)/3 = 45.0
	assert.InDelta(t, 45.0, avgDec15.LanguageLevels.Preintentional.Activity, 0.1)
	// Protolanguage: (25+35+15)/3 = 25.0
	assert.InDelta(t, 25.0, avgDec15.LanguageLevels.Protolanguage.Activity, 0.1)
	// Holophrase: (35+45+25)/3 = 35.0
	assert.InDelta(t, 35.0, avgDec15.LanguageLevels.Holophrase.Activity, 0.1)
	// Phrase: (55+65+45)/3 = 55.0
	assert.InDelta(t, 55.0, avgDec15.LanguageLevels.Phrase.Activity, 0.1)

	// Проверяем прогресс группы (должен быть отрицательным)
	require.NotNil(t, result.GroupProgress, "Group progress should be calculated")
	assert.Len(t, result.GroupProgress, 1, "Should have progress between 2 dates")

	if len(result.GroupProgress) > 0 {
		progress := result.GroupProgress[0]
		assert.Equal(t, "2025-12-12", progress.PeriodStart)
		assert.Equal(t, "2025-12-15", progress.PeriodEnd)

		// Должны быть отрицательные значения
		assert.True(t, progress.LanguageLevels.Preintentional.ActivityPercent < 0,
			"Preintentional progress should be negative")
		assert.True(t, progress.LanguageLevels.Protolanguage.ActivityPercent < 0,
			"Protolanguage progress should be negative")
		assert.True(t, progress.LanguageLevels.Holophrase.ActivityPercent < 0,
			"Holophrase progress should be negative")
		assert.True(t, progress.LanguageLevels.Phrase.ActivityPercent < 0,
			"Phrase progress should be negative")

		// Конкретные значения
		// Preintentional: ((45-50)/50)*100 = -10%
		assert.InDelta(t, -10.0, progress.LanguageLevels.Preintentional.ActivityPercent, 0.1)
		// Protolanguage: ((25-30)/30)*100 = -16.67%
		assert.InDelta(t, -16.67, progress.LanguageLevels.Protolanguage.ActivityPercent, 0.1)
		// Holophrase: ((35-40)/40)*100 = -12.5%
		assert.InDelta(t, -12.5, progress.LanguageLevels.Holophrase.ActivityPercent, 0.1)
		// Phrase: ((55-60)/60)*100 = -8.33%
		assert.InDelta(t, -8.33, progress.LanguageLevels.Phrase.ActivityPercent, 0.1)
	}

	// Дополнительно проверяем через API GET endpoint
	getReq := httptest.NewRequest("GET", "/v1/assessments/"+requestID+"?keep_in_cache=true", nil)
	getReq.Header.Set("Content-Type", "application/json")

	getRR := httptest.NewRecorder()

	handler := GetAssessmentResultsHandler(resultStorage, log)
	handler.ServeHTTP(getRR, getReq)

	assert.Equal(t, http.StatusOK, getRR.Code)

	var apiResponse map[string]interface{}
	err = json.Unmarshal(getRR.Body.Bytes(), &apiResponse)
	require.NoError(t, err)

	// Проверяем что в API ответе тоже отрицательные значения
	groupProgressArray, ok := apiResponse["group_progress"].([]interface{})
	require.True(t, ok)

	if len(groupProgressArray) > 0 {
		progressObj := groupProgressArray[0].(map[string]interface{})
		languageLevels := progressObj["language_levels"].(map[string]interface{})

		// Проверяем что все значения отрицательные
		checkNegativeLevel := func(levelName string) {
			level := languageLevels[levelName].(map[string]interface{})
			activityPercent := level["activity_percent"].(float64)
			assert.True(t, activityPercent < 0, "%s should be negative", levelName)
		}

		checkNegativeLevel("preintentional")
		checkNegativeLevel("protolanguage")
		checkNegativeLevel("holophrase")
		checkNegativeLevel("phrase")
	}
}

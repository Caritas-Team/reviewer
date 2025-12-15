package handler

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Caritas-Team/reviewer/internal/config"
	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/memcached"
	"github.com/Caritas-Team/reviewer/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

//go:embed test_file.json
var jsJSONData []byte

//go:embed test_file1.json
var js1JSONData []byte

func TestIntegration_RealFileParsing(t *testing.T) {

	cfg := config.Config{
		Files: config.Files{MaxFilesPerRequest: 20, MaxFileSize: maxFileSize},
	}
	log := logger.NewLogger(cfg)
	mockCache := NewMockCache()
	inputChan := make(chan []model.StudentPair, 10)

	handler := NewUploadHandler(cfg, log, mockCache, inputChan)

	receivedData := make(chan []model.StudentPair, 1)
	go func() {
		select {
		case data := <-inputChan:
			receivedData <- data
		case <-time.After(5 * time.Second):
			receivedData <- nil
		}
	}()

	requestID := uuid.New().String()

	mockCache.On("Get", mock.Anything, mock.MatchedBy(func(key string) bool {
		return key == fmt.Sprintf("task:%s", requestID)
	})).Return(nil, memcached.ErrCacheMiss).Once()

	mockCache.On("Set", mock.Anything, mock.MatchedBy(func(key string) bool {
		return key == fmt.Sprintf("task:%s", requestID)
	}), mock.Anything, mock.Anything).Return(nil).Once()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part1, err := writer.CreateFormFile("files", "js.json")
	require.NoError(t, err)
	_, err = part1.Write(jsJSONData)
	require.NoError(t, err)

	part2, err := writer.CreateFormFile("files", "js1.json")
	require.NoError(t, err)
	_, err = part2.Write(js1JSONData)
	require.NoError(t, err)

	require.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/v1/assessments/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Request-Id", requestID)

	rr := httptest.NewRecorder()

	handler.UploadAssessmentsHandler(rr, req)

	assert.Equal(t, 201, rr.Code, "Expected 201 Created status")

	var httpResponse map[string]any
	err = json.Unmarshal(rr.Body.Bytes(), &httpResponse)
	require.NoError(t, err)

	assert.Equal(t, "processing", httpResponse["status"])
	assert.Equal(t, float64(2), httpResponse["accepted_files"])
	assert.Equal(t, float64(1), httpResponse["students_count"])

	select {
	case pairs := <-receivedData:
		if pairs == nil {
			t.Fatal("No data received in channel")
		}

		require.Len(t, pairs, 1, "Should have 1 student pair")

		pair := pairs[0]

		// Проверяем конкретные значения из JSON данных
		assert.Equal(t, "123", pair.StudentID)
		assert.Equal(t, "before", pair.Before.Metadata.AssessmentType)
		assert.Equal(t, "after", pair.After.Metadata.AssessmentType)
		assert.True(t, pair.Before.Metadata.Date.Before(pair.After.Metadata.Date))

		// Проверяем парсинг процентов из первого файла
		assert.Equal(t, 16.0, pair.Before.CommunicativeFuncs.Control)
		assert.Equal(t, 34.0, pair.Before.LanguageLevels.Preintentional.Activity)

		// Проверяем парсинг процентов из второго файла
		assert.Equal(t, 17.0, pair.After.CommunicativeFuncs.Control)
		assert.Equal(t, 40.0, pair.After.LanguageLevels.Preintentional.Activity)

		// Проверяем словарный запас
		assert.Equal(t, 1, pair.Before.Vocabulary.ActiveWordsCount)
		assert.Equal(t, 3, pair.After.Vocabulary.ActiveWordsCount)

	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for channel data")
	}

	mockCache.AssertExpectations(t)
}

func TestDebugParsing(t *testing.T) {

	var jsonData map[string]any
	err := json.Unmarshal(jsJSONData, &jsonData)
	require.NoError(t, err, "jsJSONData should be valid JSON")
	if _, ok := jsonData["basicDictionary"].([]any); !ok {
		t.Error("basicDictionary not found in jsJSONData")
	}
}

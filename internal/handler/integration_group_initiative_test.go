//go:build integration

package handler

import (
	"bytes"
	"context"
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

func createTestJSONWithInitiative(
	studentID, date string,
	control, preintentional float64,
	activeWords int,
	protolanguage, holophrase, phrase float64,
	protInit, golInit, fraInit float64,
) string {
	dictItems := "["
	for i := 0; i < activeWords; i++ {
		if i > 0 {
			dictItems += ","
		}
		dictItems += fmt.Sprintf(`{"itemOffStyle": "", "content": "word%d"}`, i+1)
	}
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
            "protInitProcNumElem": "%.0f%%",
            "golActProcNumElem": "%.0f%%",
            "golInitProcNumElem": "%.0f%%",
            "fraActProcNumElem": "%.0f%%",
            "fraInitProcNumElem": "%.0f%%"
        },
        "basicDictionary": %s,
        "dictBasicMore": []
    }`, date, studentID,
		control,
		preintentional, protolanguage, protInit, holophrase, golInit, phrase, fraInit,
		dictItems)
}

func TestIntegration_GroupProgress_InitiativeDiff(t *testing.T) {
	ctx := context.Background()

	memcachedContainer, err := memcached.Run(ctx, "memcached:1.6-alpine")
	require.NoError(t, err)
	defer func() {
		_ = memcachedContainer.Terminate(ctx)
	}()

	host, err := memcachedContainer.Host(ctx)
	require.NoError(t, err)
	port, err := memcachedContainer.MappedPort(ctx, "11211")
	require.NoError(t, err)

	memcachedAddr := fmt.Sprintf("%s:%s", host, port.Port())

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

	cache, err := memcachedclient.NewCache(ctx, cfg)
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	resultStorage := assessment.NewResultStorage(cache)
	inputChan := make(chan []model.StudentPair, 10)
	uploadHandler := NewUploadHandler(cfg, log, cache, resultStorage, inputChan)

	processor := assessment.NewProcessor(&assessment.DiffCalculator{})
	worker := assessment.NewWorker(log, resultStorage, processor, time.Hour)
	go worker.Run(ctx, inputChan)

	requestID := uuid.New().String()

	// 3 студента × 2 даты, но инициатива меняется между датами
	// 12-е: protInit=10, golInit=30, fraInit=40
	// 15-е: protInit=25, golInit=20, fraInit=40
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
		protInit    float64
		golInit     float64
		fraInit     float64
	}{
		{"A_2025-12-12.json", "A", "2025-12-12", 10, 50, 30, 40, 60, 10, 10, 30, 40},
		{"A_2025-12-15.json", "A", "2025-12-15", 15, 55, 35, 45, 65, 10, 25, 20, 40},

		{"B_2025-12-12.json", "B", "2025-12-12", 20, 60, 40, 50, 70, 10, 10, 30, 40},
		{"B_2025-12-15.json", "B", "2025-12-15", 25, 65, 45, 55, 75, 10, 25, 20, 40},

		{"C_2025-12-12.json", "C", "2025-12-12", 5, 40, 20, 30, 50, 10, 10, 30, 40},
		{"C_2025-12-15.json", "C", "2025-12-15", 10, 45, 25, 35, 55, 10, 25, 20, 40},
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, tf := range testFiles {
		jsonData := createTestJSONWithInitiative(
			tf.studentID, tf.date,
			tf.control, tf.preintent, tf.activeWords,
			tf.protolang, tf.holophrase, tf.phrase,
			tf.protInit, tf.golInit, tf.fraInit,
		)
		part, err := writer.CreateFormFile("files", tf.filename)
		require.NoError(t, err)
		_, err = part.Write([]byte(jsonData))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/v1/assessments/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Request-Id", requestID)

	rr := httptest.NewRecorder()
	uploadHandler.UploadAssessmentsHandler(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	deadline := time.Now().Add(30 * time.Second)
	var result *assessment.ProcessingResult
	for time.Now().Before(deadline) {
		res, err := resultStorage.Get(ctx, requestID)
		if err == nil && res.Status == "completed" {
			result = res
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	require.NotNil(t, result)
	require.NotNil(t, result.GroupProgress)
	require.Len(t, result.GroupProgress, 1)

	progress := result.GroupProgress[0]
	assert.Equal(t, "2025-12-12", progress.PeriodStart)
	assert.Equal(t, "2025-12-15", progress.PeriodEnd)

	// ожидаемые diff по инициативе (групповые средние одинаковые у всех студентов, diff тот же)
	// prot: 25 - 10 = +15
	// hol: 20 - 30 = -10
	// fra: 40 - 40 = 0
	assert.InDelta(t, 15.0, progress.LanguageLevels.Protolanguage.InitiativeDiff, 0.01)
	assert.InDelta(t, -10.0, progress.LanguageLevels.Holophrase.InitiativeDiff, 0.01)
	assert.InDelta(t, 0.0, progress.LanguageLevels.Phrase.InitiativeDiff, 0.01)
}

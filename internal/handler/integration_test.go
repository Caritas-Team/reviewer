package handler

import (
	"bytes"
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

func TestIntegration_RealFileParsing(t *testing.T) {

	jsJSONData := `{
  "por01": "<div>2025-11-11</div>",
  "por02": "<div>123</div>",
  "por03": "<div>2025-12-24</div>",
  "newAct01": {
    "procNumElem": "16%"
  },
  "newAct02": {
    "procNumElem": "0%"
  },
  "newAct03": {
    "procNumElem": "0%"
  },
  "newAct04": {
    "procNumElem": "0%"
  },
  "levelZoneZbr": "Фраза",
  "diagramBlock": {
    "predActProcNumElem": "34%",
    "protActProcNumElem": "4%",
    "protInitProcNumElem": "0%",
    "golActProcNumElem": "2%",
    "golInitProcNumElem": "33%",
    "fraActProcNumElem": "1%",
    "fraInitProcNumElem": "25%"
  },
  "basicDictionary": [
    {
      "colorStyle": "res-dict__item-viol",
      "itemOffStyle": "",
      "content": "что?"
    },
    {
      "colorStyle": "res-dict__item-yellow",
      "itemOffStyle": "res-dict__item-off",
      "content": "я, мой"
    },
    {
      "colorStyle": "res-dict__item-green",
      "itemOffStyle": "res-dict__item-off",
      "content": "хотеть"
    }
  ],
  "dictBasicMore": [
    "123"
  ]
}`

	js1JSONData := `{
  "por01": "<div>2025-12-11</div>",
  "por02": "<div>123</div>",
  "por03": "<div>2025-12-24</div>",
  "newAct01": {
    "procNumElem": "17%"
  },
  "newAct02": {
    "procNumElem": "1%"
  },
  "newAct03": {
    "procNumElem": "1%"
  },
  "newAct04": {
    "procNumElem": "1%"
  },
  "levelZoneZbr": "Фраза",
  "diagramBlock": {
    "predActProcNumElem": "40%",
    "protActProcNumElem": "10%",
    "protInitProcNumElem": "2%",
    "golActProcNumElem": "12%",
    "golInitProcNumElem": "40%",
    "fraActProcNumElem": "10%",
    "fraInitProcNumElem": "29%"
  },
  "basicDictionary": [
    {
      "colorStyle": "res-dict__item-viol",
      "itemOffStyle": "",
      "content": "что?"
    },
    {
      "colorStyle": "res-dict__item-yellow",
      "itemOffStyle": "",
      "content": "я, мой"
    },
    {
      "colorStyle": "res-dict__item-green",
      "itemOffStyle": "",
      "content": "хотеть"
    }
  ],
  "dictBasicMore": [
    "123", "124"
  ]
}`

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
	_, err = part1.Write([]byte(jsJSONData))
	require.NoError(t, err)

	part2, err := writer.CreateFormFile("files", "js1.json")
	require.NoError(t, err)
	_, err = part2.Write([]byte(js1JSONData))
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

		t.Log("=== PARSING RESULTS ===")
		t.Logf("Student: %s", pair.StudentID)
		t.Logf("Request ID: %s", pair.RequestID)

		t.Log("\nBEFORE assessment:")
		printAssessmentInfo(t, pair.Before)

		t.Log("\nAFTER assessment:")
		printAssessmentInfo(t, pair.After)

		t.Log("\n=== PROGRESS ===")
		progress := calculateProgress(pair.Before, pair.After)
		for key, value := range progress {
			t.Logf("%s: %v", key, value)
		}

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

func printAssessmentInfo(t *testing.T, doc *model.AssessmentDocument) {
	t.Logf("  Date: %s", doc.Metadata.Date.Format("2006-01-02"))
	t.Logf("  Type: %s", doc.Metadata.AssessmentType)
	t.Logf("  Control: %.1f%%", doc.CommunicativeFuncs.Control)
	t.Logf("  Preintentional: %.1f%%", doc.LanguageLevels.Preintentional.Activity)
	t.Logf("  Vocabulary: %d words (%d active + %d additional)",
		doc.Vocabulary.TotalWordsCount,
		doc.Vocabulary.ActiveWordsCount,
		len(doc.Vocabulary.AdditionalWords))
}

func calculateProgress(before, after *model.AssessmentDocument) map[string]any {
	return map[string]any{
		"control_delta":           after.CommunicativeFuncs.Control - before.CommunicativeFuncs.Control,
		"preintentional_delta":    after.LanguageLevels.Preintentional.Activity - before.LanguageLevels.Preintentional.Activity,
		"vocabulary_growth":       after.Vocabulary.TotalWordsCount - before.Vocabulary.TotalWordsCount,
		"active_words_growth":     after.Vocabulary.ActiveWordsCount - before.Vocabulary.ActiveWordsCount,
		"additional_words_growth": len(after.Vocabulary.AdditionalWords) - len(before.Vocabulary.AdditionalWords),
	}
}

func TestDebugParsing(t *testing.T) {

	jsJSONData := `{
  "por01": "<div>2025-11-11</div>",
  "por02": "<div>123</div>",
  "por03": "<div>2025-12-24</div>",
  "newAct01": {
    "procNumElem": "16%"
  },
  "newAct02": {
    "procNumElem": "0%"
  },
  "newAct03": {
    "procNumElem": "0%"
  },
  "newAct04": {
    "procNumElem": "0%"
  },
  "levelZoneZbr": "Фраза",
  "diagramBlock": {
    "predActProcNumElem": "34%",
    "protActProcNumElem": "4%",
    "protInitProcNumElem": "0%",
    "golActProcNumElem": "2%",
    "golInitProcNumElem": "33%",
    "fraActProcNumElem": "1%",
    "fraInitProcNumElem": "25%"
  },
  "basicDictionary": [
    {
      "colorStyle": "res-dict__item-viol",
      "itemOffStyle": "",
      "content": "что?"
    },
    {
      "colorStyle": "res-dict__item-yellow",
      "itemOffStyle": "res-dict__item-off",
      "content": "я, мой"
    },
    {
      "colorStyle": "res-dict__item-green",
      "itemOffStyle": "res-dict__item-off",
      "content": "хотеть"
    }
  ],
  "dictBasicMore": [
    "123"
  ]
}`

	var jsonData map[string]any
	err := json.Unmarshal([]byte(jsJSONData), &jsonData)
	require.NoError(t, err, "js.json should be valid JSON")

	t.Log("js.json is valid JSON")
	t.Logf("por02: %v", jsonData["por02"])
	t.Logf("newAct01: %v", jsonData["newAct01"])

	if dict, ok := jsonData["basicDictionary"].([]any); ok {
		t.Logf("basicDictionary has %d items", len(dict))
		for i, item := range dict {
			if itemMap, ok := item.(map[string]interface{}); ok {
				t.Logf("  Item %d: itemOffStyle=%v", i+1, itemMap["itemOffStyle"])
			}
		}
	}
}

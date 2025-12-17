package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Caritas-Team/reviewer/internal/config"
	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/memcached"
	"github.com/Caritas-Team/reviewer/internal/model"
	"github.com/Caritas-Team/reviewer/internal/usecase/assessment"
	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockCache struct {
	mock.Mock
	data map[string][]byte
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string][]byte),
	}
}

func (m *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if data, ok := m.data[key]; ok {
		return data, nil
	}
	return nil, args.Error(1)
}

func (m *MockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	m.data[key] = value
	return args.Error(0)
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	delete(m.data, key)
	return args.Error(0)
}

func (m *MockCache) Ping() error {
	return m.Called().Error(0)
}

func (m *MockCache) IsEnabled() error {
	return m.Called().Error(0)
}

func (m *MockCache) Close() error {
	return m.Called().Error(0)
}

func (m *MockCache) Increment(ctx context.Context, key string, value uint64) (uint64, error) {
	args := m.Called(ctx, key, value)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockCache) Decrement(ctx context.Context, key string, value uint64) (uint64, error) {
	args := m.Called(ctx, key, value)
	return args.Get(0).(uint64), args.Error(1)
}

func createFullJSON(studentID, date string) string {
	return fmt.Sprintf(`{
		"por01": "<div>%s</div>",
		"por02": "<div>%s</div>",
		"por03": "<div>2025-12-24</div>",
		"newAct01": {"procNumElem": "16%%"},
		"newAct02": {"procNumElem": "0%%"},
		"newAct03": {"procNumElem": "0%%"},
		"newAct04": {"procNumElem": "0%%"},
		"levelZoneZbr": "Фраза",
		"diagramBlock": {
			"predActProcNumElem": "34%%",
			"protActProcNumElem": "4%%",
			"protInitProcNumElem": "0%%",
			"golActProcNumElem": "2%%",
			"golInitProcNumElem": "33%%",
			"fraActProcNumElem": "1%%",
			"fraInitProcNumElem": "25%%"
		},
		"basicDictionary": [
			{"colorStyle": "res-dict__item-viol", "itemOffStyle": "res-dict__item-off", "content": "что?"},
			{"colorStyle": "res-dict__item-yellow", "itemOffStyle": "", "content": "я, мой"},
			{"colorStyle": "res-dict__item-green", "itemOffStyle": "", "content": "хотеть"}
		],
		"dictBasicMore": ["123"]
	}`, date, studentID)
}

const maxFileSize = 10 * 1024 * 1024

func TestUploadHandler_ValidationScenarios(t *testing.T) {
	cfg := config.Config{
		Files: config.Files{MaxFilesPerRequest: 20, MaxFileSize: 1024},
	}
	log := logger.NewLogger(cfg)
	mockCache := NewMockCache()

	inputChan := make(chan []model.StudentPair, 10)

	resultStorage := assessment.NewResultStorage(mockCache)
	handler := NewUploadHandler(cfg, log, mockCache, resultStorage, inputChan)

	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedStatus int
		expectedError  string
		checkDetails   func(t *testing.T, response map[string]any)
	}{
		{
			name: "Missing X-Request-Id header",
			setupRequest: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, err := writer.CreateFormFile("files", "test1.json")
				if err != nil {
					panic(err)
				}
				_, err = io.WriteString(part, createFullJSON("student1", "2023-01-01"))
				if err != nil {
					panic(err)
				}
				part2, err := writer.CreateFormFile("files", "test2.json")
				if err != nil {
					panic(err)
				}
				_, err = io.WriteString(part2, createFullJSON("student1", "2023-06-01"))
				if err != nil {
					panic(err)
				}

				if err := writer.Close(); err != nil {
					panic(err)
				}

				req := httptest.NewRequest("POST", "/v1/assessments/upload", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing X-Request-Id header",
			checkDetails: func(t *testing.T, response map[string]any) {
				details, ok := response["details"].(map[string]any)
				assert.True(t, ok, "Response should contain details")
				assert.Equal(t, "X-Request-Id", details["field"])
				assert.Equal(t, "required", details["constraint"])
			},
		},
		{
			name: "Invalid UUID format",
			setupRequest: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, _ := writer.CreateFormFile("files", "test1.json")
				_, err := io.WriteString(part, createFullJSON("student1", "2023-01-01"))
				if err != nil {
					panic(err)
				}
				part2, _ := writer.CreateFormFile("files", "test2.json")
				_, err = io.WriteString(part2, createFullJSON("student1", "2023-06-01"))
				if err != nil {
					panic(err)
				}
				if err := writer.Close(); err != nil {
					panic(err)
				}

				req := httptest.NewRequest("POST", "/v1/assessments/upload", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				req.Header.Set("X-Request-Id", "not-a-valid-uuid")
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid X-Request-Id format",
			checkDetails: func(t *testing.T, response map[string]any) {
				details, ok := response["details"].(map[string]any)
				assert.True(t, ok, "Response should contain details")
				assert.Equal(t, "X-Request-Id", details["field"])
				assert.Equal(t, "uuid", details["format"])
			},
		},
		{
			name: "Odd number of files",
			setupRequest: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				for i := 1; i <= 3; i++ {
					part, err := writer.CreateFormFile("files", fmt.Sprintf("test%d.json", i))
					if err != nil {
						panic(err)
					}
					_, err = io.WriteString(part, createFullJSON("student1", fmt.Sprintf("2023-0%d-01", i)))
					if err != nil {
						panic(err)
					}
				}
				if err := writer.Close(); err != nil {
					panic(err)
				}

				req := httptest.NewRequest("POST", "/v1/assessments/upload", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				req.Header.Set("X-Request-Id", uuid.New().String())
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Number of files must be even",
			checkDetails: func(t *testing.T, response map[string]any) {
				details, ok := response["details"].(map[string]any)
				assert.True(t, ok, "Response should contain details")
				assert.Equal(t, "files", details["field"])
				assert.Equal(t, "even_count", details["constraint"])
				assert.Equal(t, float64(3), details["got_items"])
			},
		},
		{
			name: "File size exceeds limit",
			setupRequest: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)

				part, _ := writer.CreateFormFile("files", "large.json")

				largeContent := strings.Repeat(createFullJSON("student1", "2023-01-01"), 1024)
				_, err := io.WriteString(part, largeContent)
				if err != nil {
					panic(err)
				}

				part2, _ := writer.CreateFormFile("files", "large2.json")
				_, err = io.WriteString(part2, largeContent)
				if err != nil {
					panic(err)
				}

				if err := writer.Close(); err != nil {
					panic(err)
				}

				req := httptest.NewRequest("POST", "/v1/assessments/upload", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				req.Header.Set("X-Request-Id", uuid.New().String())
				return req
			},
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectedError:  "Total file size exceeds 50MB",
			checkDetails: func(t *testing.T, response map[string]any) {
				details, ok := response["details"].(map[string]any)
				assert.True(t, ok, "Response should contain details")
				assert.Equal(t, float64(50), details["max_size_mb"])
				assert.Equal(t, "max_total_size", details["constraint"])
			},
		},
		{
			name: "Wrong number of documents per student",
			setupRequest: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				files := []string{
					createFullJSON("student1", "2023-01-01"),
					createFullJSON("student1", "2023-06-01"),
					createFullJSON("student2", "2023-01-01"),
					createFullJSON("student3", "2023-06-01"),
				}
				for i, content := range files {
					part, _ := writer.CreateFormFile("files", fmt.Sprintf("test%d.json", i+1))
					_, err := io.WriteString(part, content)
					if err != nil {
						panic(err)
					}
				}
				if err := writer.Close(); err != nil {
					panic(err)
				}

				req := httptest.NewRequest("POST", "/v1/assessments/upload", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				req.Header.Set("X-Request-Id", uuid.New().String())
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "has 1 documents, expected exactly 2",
			checkDetails: func(t *testing.T, response map[string]any) {
				details, ok := response["details"].(map[string]any)
				assert.True(t, ok, "Response should contain details")
				assert.Contains(t, []string{"student2", "student3"}, details["student_id"])
				assert.Equal(t, "exactly_two_per_student", details["constraint"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			rr := httptest.NewRecorder()

			mockCache.On("Get", mock.Anything, mock.Anything).Return(nil, memcached.ErrCacheMiss).Maybe()
			mockCache.On("Get", mock.Anything, mock.MatchedBy(func(key string) bool {
				return strings.Contains(key, "task:")
			})).Return(func(ctx context.Context, key string) ([]byte, error) {
				if data, ok := mockCache.data[key]; ok {
					return data, nil
				}
				return nil, memcached.ErrCacheMiss
			}, nil).Maybe()

			if tt.expectedStatus == http.StatusCreated {
				mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			}

			handler.UploadAssessmentsHandler(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				message, ok := response["message"].(string)
				assert.True(t, ok, "Response should contain message field")

				if ok {
					assert.Contains(t, strings.ToLower(message), strings.ToLower(tt.expectedError),
						fmt.Sprintf("Expected error to contain '%s', got: '%s'", tt.expectedError, message))
				}

				if tt.checkDetails != nil {
					tt.checkDetails(t, response)
				}
			}
		})
	}
}

func TestUploadHandler_ErrorDetails(t *testing.T) {
	cfg := config.Config{
		Files: config.Files{MaxFilesPerRequest: 20, MaxFileSize: maxFileSize},
	}
	log := logger.NewLogger(cfg)
	mockCache := NewMockCache()

	inputChan := make(chan []model.StudentPair, 10)

	resultStorage := assessment.NewResultStorage(mockCache)
	handler := NewUploadHandler(cfg, log, mockCache, resultStorage, inputChan)

	tests := []struct {
		name         string
		jsonContent  string
		expectedKey  string
		checkDetails func(t *testing.T, details map[string]any)
	}{
		{
			name: "Missing por02 field",
			jsonContent: `{
				"por01": "<div>2023-01-01</div>",
				"por03": "<div>2025-12-24</div>"
			}`,
			expectedKey: "parse_error",
			checkDetails: func(t *testing.T, details map[string]any) {
				assert.Equal(t, "por02", details["field"])
				assert.Equal(t, "required", details["constraint"])
			},
		},
		{
			name: "Missing por01 field",
			jsonContent: `{
				"por02": "<div>student1</div>",
				"por03": "<div>2025-12-24</div>"
			}`,
			expectedKey: "parse_error",
			checkDetails: func(t *testing.T, details map[string]any) {
				assert.Equal(t, "por01", details["field"])
				assert.Equal(t, "required", details["constraint"])
			},
		},
		{
			name: "Invalid date format",
			jsonContent: `{
				"por01": "<div>not-a-date</div>",
				"por02": "<div>student1</div>"
			}`,
			expectedKey: "parse_error",
			checkDetails: func(t *testing.T, details map[string]any) {
				assert.Equal(t, "por01", details["field"])
				assert.Equal(t, "YYYY-MM-DD", details["format"])
				assert.Equal(t, "date_format", details["constraint"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			createTestFile(t, writer, "test1.json", tt.jsonContent)
			createTestFile(t, writer, "test2.json", createFullJSON("student1", "2023-06-01"))
			if err := writer.Close(); err != nil {
				panic(err)
			}

			req := httptest.NewRequest("POST", "/v1/assessments/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("X-Request-Id", uuid.New().String())

			rr := httptest.NewRecorder()

			mockCache.On("Get", mock.Anything, mock.Anything).Return(nil, memcached.ErrCacheMiss)

			handler.UploadAssessmentsHandler(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)

			var response map[string]any
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedKey, response["error"])

			details, ok := response["details"].(map[string]any)
			assert.True(t, ok, "Response should contain details")

			if tt.checkDetails != nil {
				tt.checkDetails(t, details)
			}
		})
	}
}

func TestUploadHandler_SuccessfulUpload(t *testing.T) {
	cfg := config.Config{
		Files: config.Files{MaxFilesPerRequest: 20, MaxFileSize: maxFileSize},
	}
	log := logger.NewLogger(cfg)
	mockCache := NewMockCache()
	inputChan := make(chan []model.StudentPair, 10)

	resultStorage := assessment.NewResultStorage(mockCache)
	handler := NewUploadHandler(cfg, log, mockCache, resultStorage, inputChan)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	files := []string{
		createFullJSON("student1", "2023-01-01"),
		createFullJSON("student1", "2023-06-01"),
		createFullJSON("student2", "2023-02-01"),
		createFullJSON("student2", "2023-07-01"),
	}

	for i, content := range files {
		part, _ := writer.CreateFormFile("files", fmt.Sprintf("test%d.json", i+1))
		_, err := io.WriteString(part, content)
		require.NoError(t, err)
	}

	meta := `{"organization": "Test Org", "specialist": "Test Specialist"}`
	err := writer.WriteField("meta", meta)
	require.NoError(t, err)

	require.NoError(t, writer.Close())

	requestID := uuid.New().String()
	req := httptest.NewRequest("POST", "/v1/assessments/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Request-Id", requestID)

	rr := httptest.NewRecorder()

	mockCache.On("Get", mock.Anything, mock.Anything).Return(nil, memcached.ErrCacheMiss)
	mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	done := make(chan bool)
	go func() {
		handler.UploadAssessmentsHandler(rr, req)
		done <- true
	}()

	select {
	case pairs := <-inputChan:
		assert.Equal(t, 2, len(pairs), "Should have 2 student pairs")

		// Проверяем структуру данных
		for _, pair := range pairs {
			assert.Equal(t, requestID, pair.RequestID)
			assert.Contains(t, []string{"student1", "student2"}, pair.StudentID)
			assert.NotNil(t, pair.Before)
			assert.NotNil(t, pair.After)
			assert.True(t, pair.Before.Metadata.Date.Before(pair.After.Metadata.Date))
			assert.Equal(t, "before", pair.Before.Metadata.AssessmentType)
			assert.Equal(t, "after", pair.After.Metadata.AssessmentType)

			// Проверяем распарсенные данные
			assert.Equal(t, 16.0, pair.Before.CommunicativeFuncs.Control)
			assert.Equal(t, 34.0, pair.Before.LanguageLevels.Preintentional.Activity)
			assert.Equal(t, 2, pair.Before.Vocabulary.ActiveWordsCount) // 2 активных слова в basicDictionary
			assert.Equal(t, 1, len(pair.Before.Vocabulary.AdditionalWords))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for input channel")
	}

	<-done

	// Проверяем ответ
	assert.Equal(t, http.StatusCreated, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, requestID, response["request_id"])
	assert.Equal(t, "processing", response["status"])
	assert.Equal(t, float64(4), response["accepted_files"])
	assert.Equal(t, float64(2), response["students_count"])
	assert.Equal(t, float64(15), response["estimated_completion_sec"])
}

func TestUploadHandler_Idempotency(t *testing.T) {
	cfg := config.Config{
		Files: config.Files{MaxFilesPerRequest: 20, MaxFileSize: maxFileSize},
	}
	log := logger.NewLogger(cfg)
	mockCache := NewMockCache()
	inputChan := make(chan []model.StudentPair, 10)

	resultStorage := assessment.NewResultStorage(mockCache)
	handler := NewUploadHandler(cfg, log, mockCache, resultStorage, inputChan)

	requestID := uuid.New().String()

	// Создаем существующую задачу в кэше
	existingTask := UploadTask{
		RequestID:     requestID,
		Status:        "processing",
		TotalFiles:    2,
		TotalStudents: 1,
		CreatedAt:     time.Now().UTC().Add(-5 * time.Minute),
		UpdatedAt:     time.Now().UTC(),
		EstimatedSec:  10,
	}

	taskData, err := json.Marshal(existingTask)
	require.NoError(t, err)
	mockCache.data[fmt.Sprintf("task:%s", requestID)] = taskData

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("files", "test1.json")
	_, err = io.WriteString(part, createFullJSON("student1", "2023-01-01"))
	require.NoError(t, err)

	part2, _ := writer.CreateFormFile("files", "test2.json")
	_, err = io.WriteString(part2, createFullJSON("student1", "2023-06-01"))
	require.NoError(t, err)

	require.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/v1/assessments/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Request-Id", requestID)

	rr := httptest.NewRecorder()

	mockCache.On("Get", mock.Anything, mock.MatchedBy(func(key string) bool {
		return key == fmt.Sprintf("task:%s", requestID)
	})).Return(taskData, nil)

	handler.UploadAssessmentsHandler(rr, req)

	// Должен вернуть 409 Conflict
	assert.Equal(t, http.StatusConflict, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "conflict", response["error"])
	assert.Contains(t, response["message"], "already being processed")
}

func createTestFile(t *testing.T, writer *multipart.Writer, filename, content string) {
	t.Helper()
	part, err := writer.CreateFormFile("files", filename)
	require.NoError(t, err)
	_, err = io.WriteString(part, content)
	require.NoError(t, err)
}

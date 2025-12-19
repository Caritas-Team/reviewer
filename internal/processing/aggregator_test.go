package processing

import (
	"context"
	"testing"
	"time"

	"github.com/Caritas-Team/reviewer/internal/config"
	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/model"
	"github.com/Caritas-Team/reviewer/internal/storage"
)

// Тест на успешную обработку результатов
func TestResultAggregator_Success(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultChan := make(chan model.ProcessingResult, 10)
	errorChan := make(chan model.ProcessingError, 10)

	// Создаем фиктивную конфигурацию с настоящими настройками мем-кеша
	mockCfg := config.Config{
		Logging: config.Logging{
			Level:  "debug",
			Format: "json",
		},
		Memcached: config.Memcached{
			Enable:     true,
			Servers:    []string{"memcached:11211"}, // Адрес мем-кеша
			DefaultTTL: 3600,                        // Срок хранения по умолчанию
			KeyPrefix:  "pdf_api",                   // Префикс ключей
		},
	}

	// Создаем логгер
	testLogger := logger.NewLogger(mockCfg)

	// Создаем настоящее хранилище
	realStorage, err := storage.NewResultStorage(mockCfg, testLogger)
	if err != nil {
		t.Fatalf("Failed to create result storage: %v", err)
	}

	// Создаем результаты для тестирования
	result1 := model.ProcessingResult{
		RequestID:         "test-request",
		Status:            "processing",
		ProcessedStudents: 1,
		TotalStudents:     2,
		ResultDetails: map[string]interface{}{
			"diff": AssessmentDiff{
				StudentID:   "S12345",
				PeriodStart: "2025-12-18",
				PeriodEnd:   "2025-12-19",
				LangDevDiff: LangDevDiff{
					PreintentionalDelta: 10.5,
					ProtolanguageDelta:  10,
					HolophraseDelta:     10,
					PhraseDelta:         10,
				},
				CommFuncsDiff: CommFuncsDiff{
					ControlDelta:             10,
					ObtainingDesiredDelta:    10,
					SocialInteractionDelta:   10,
					InformationExchangeDelta: 10,
				},
				VocabularyDiff: VocabularyDiff{
					ActiveWordsDelta: 20,
					TotalWordsDelta:  40,
				},
				GeneralProgress: GeneralProgress{
					AverageProgress: 10.0625,
				},
			},
		},
		CreatedAt: time.Now(),
	}

	result2 := model.ProcessingResult{
		RequestID:         "test-request",
		Status:            "processing",
		ProcessedStudents: 1,
		TotalStudents:     2,
		ResultDetails: map[string]interface{}{
			"diff": AssessmentDiff{
				StudentID:   "S12345",
				PeriodStart: "2025-12-18",
				PeriodEnd:   "2025-12-19",
				LangDevDiff: LangDevDiff{
					PreintentionalDelta: 10.5,
					ProtolanguageDelta:  10,
					HolophraseDelta:     10,
					PhraseDelta:         10,
				},
				CommFuncsDiff: CommFuncsDiff{
					ControlDelta:             10,
					ObtainingDesiredDelta:    10,
					SocialInteractionDelta:   10,
					InformationExchangeDelta: 10,
				},
				VocabularyDiff: VocabularyDiff{
					ActiveWordsDelta: 20,
					TotalWordsDelta:  40,
				},
				GeneralProgress: GeneralProgress{
					AverageProgress: 10.0625,
				},
			},
		},
		CreatedAt: time.Now(),
	}

	// Запускаем аггрегатор
	go ResultAggregator(resultChan, errorChan, realStorage, testLogger)

	// Отправляем результаты
	resultChan <- result1
	resultChan <- result2

	// Ждём завершения
	time.Sleep(500 * time.Millisecond)

	// Проверяем результат
	finalResult, exists := realStorage.Get("test-request")
	if !exists {
		t.Fatalf("Result not found")
	}

	expectedResult := model.ProcessingResult{
		RequestID:         "test-request",
		Status:            "completed",
		ProcessedStudents: 2,
		TotalStudents:     2,
		ResultDetails: map[string]interface{}{
			"diff1": AssessmentDiff{
				StudentID:   "S12345",
				PeriodStart: "2025-12-18",
				PeriodEnd:   "2025-12-19",
				LangDevDiff: LangDevDiff{
					PreintentionalDelta: 10.5,
					ProtolanguageDelta:  10,
					HolophraseDelta:     10,
					PhraseDelta:         10,
				},
				CommFuncsDiff: CommFuncsDiff{
					ControlDelta:             10,
					ObtainingDesiredDelta:    10,
					SocialInteractionDelta:   10,
					InformationExchangeDelta: 10,
				},
				VocabularyDiff: VocabularyDiff{
					ActiveWordsDelta: 20,
					TotalWordsDelta:  40,
				},
				GeneralProgress: GeneralProgress{
					AverageProgress: 10.0625,
				},
			},
			"diff2": AssessmentDiff{
				StudentID:   "S12345",
				PeriodStart: "2025-12-18",
				PeriodEnd:   "2025-12-19",
				LangDevDiff: LangDevDiff{
					PreintentionalDelta: 10.5,
					ProtolanguageDelta:  10,
					HolophraseDelta:     10,
					PhraseDelta:         10,
				},
				CommFuncsDiff: CommFuncsDiff{
					ControlDelta:             10,
					ObtainingDesiredDelta:    10,
					SocialInteractionDelta:   10,
					InformationExchangeDelta: 10,
				},
				VocabularyDiff: VocabularyDiff{
					ActiveWordsDelta: 20,
					TotalWordsDelta:  40,
				},
				GeneralProgress: GeneralProgress{
					AverageProgress: 10.0625,
				},
			},
		},
		CreatedAt: time.Now(),
	}

	// Проверяем поля результата
	if finalResult.RequestID != expectedResult.RequestID {
		t.Fatalf("RequestID mismatch: got %s, want %s", finalResult.RequestID, expectedResult.RequestID)
	}
	if finalResult.Status != expectedResult.Status {
		t.Fatalf("Status mismatch: got %s, want %s", finalResult.Status, expectedResult.Status)
	}
	if finalResult.ProcessedStudents != expectedResult.ProcessedStudents {
		t.Fatalf("ProcessedStudents mismatch: got %d, want %d", finalResult.ProcessedStudents, expectedResult.ProcessedStudents)
	}
	if finalResult.TotalStudents != expectedResult.TotalStudents {
		t.Fatalf("TotalStudents mismatch: got %d, want %d", finalResult.TotalStudents, expectedResult.TotalStudents)
	}

	// Проверяем ResultDetails вручную
	expectedDetails := expectedResult.ResultDetails["diff"].(AssessmentDiff)
	finalDetails := finalResult.ResultDetails["diff"].(AssessmentDiff)

	if expectedDetails.StudentID != finalDetails.StudentID {
		t.Fatalf("StudentID mismatch: got %s, want %s", finalDetails.StudentID, expectedDetails.StudentID)
	}
	if expectedDetails.PeriodStart != finalDetails.PeriodStart {
		t.Fatalf("PeriodStart mismatch: got %s, want %s", finalDetails.PeriodStart, expectedDetails.PeriodStart)
	}
	if expectedDetails.PeriodEnd != finalDetails.PeriodEnd {
		t.Fatalf("PeriodEnd mismatch: got %s, want %s", finalDetails.PeriodEnd, expectedDetails.PeriodEnd)
	}

	// Проверяем LangDevDiff
	if expectedDetails.LangDevDiff.PreintentionalDelta != finalDetails.LangDevDiff.PreintentionalDelta ||
		expectedDetails.LangDevDiff.ProtolanguageDelta != finalDetails.LangDevDiff.ProtolanguageDelta ||
		expectedDetails.LangDevDiff.HolophraseDelta != finalDetails.LangDevDiff.HolophraseDelta ||
		expectedDetails.LangDevDiff.PhraseDelta != finalDetails.LangDevDiff.PhraseDelta {
		t.Fatalf("LangDevDiff mismatch: got %+v, want %+v", finalDetails.LangDevDiff, expectedDetails.LangDevDiff)
	}

	// Проверяем CommFuncsDiff
	if expectedDetails.CommFuncsDiff.ControlDelta != finalDetails.CommFuncsDiff.ControlDelta ||
		expectedDetails.CommFuncsDiff.ObtainingDesiredDelta != finalDetails.CommFuncsDiff.ObtainingDesiredDelta ||
		expectedDetails.CommFuncsDiff.SocialInteractionDelta != finalDetails.CommFuncsDiff.SocialInteractionDelta ||
		expectedDetails.CommFuncsDiff.InformationExchangeDelta != finalDetails.CommFuncsDiff.InformationExchangeDelta {
		t.Fatalf("CommFuncsDiff mismatch: got %+v, want %+v", finalDetails.CommFuncsDiff, expectedDetails.CommFuncsDiff)
	}

	// Проверяем VocabularyDiff
	if expectedDetails.VocabularyDiff.ActiveWordsDelta != finalDetails.VocabularyDiff.ActiveWordsDelta ||
		expectedDetails.VocabularyDiff.TotalWordsDelta != finalDetails.VocabularyDiff.TotalWordsDelta {
		t.Fatalf("VocabularyDiff mismatch: got %+v, want %+v", finalDetails.VocabularyDiff, expectedDetails.VocabularyDiff)
	}

	// Проверяем GeneralProgress
	if expectedDetails.GeneralProgress.AverageProgress != finalDetails.GeneralProgress.AverageProgress {
		t.Fatalf("GeneralProgress mismatch: got %+v, want %+v", finalDetails.GeneralProgress, expectedDetails.GeneralProgress)
	}

	// Проверяем CreatedAt
	if !finalResult.CreatedAt.Equal(expectedResult.CreatedAt) {
		t.Fatalf("CreatedAt mismatch: got %v, want %v", finalResult.CreatedAt, expectedResult.CreatedAt)
	}

	cancel()
}

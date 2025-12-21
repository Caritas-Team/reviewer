package processing

import (
	"context"
	"fmt"
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
			Servers:    []string{"127.0.0.1:11211"}, // Адрес мем-кеша
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

	reqID := fmt.Sprintf("test-request-%d", time.Now().UnixNano())

	// Создаем результаты для тестирования
	result1 := model.ProcessingResult{
		RequestID:         reqID,
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
		RequestID:         reqID,
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

	expectedDiff := result1.ResultDetails["diff"].(AssessmentDiff)
	// Запускаем аггрегатор
	go ResultAggregator(resultChan, errorChan, realStorage, testLogger)

	// Отправляем результаты
	resultChan <- result1
	resultChan <- result2

	// Ждём, пока статус станет completed
	var finalResult model.ProcessingResult
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		r, ok := realStorage.Get(reqID)
		if ok && r.Status == "completed" {
			finalResult = r
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if finalResult.RequestID == "" {
		t.Fatalf("Result not completed in time")
	}

	if finalResult.Status != "completed" {
		t.Fatalf("Status mismatch: got %s, want completed", finalResult.Status)
	}
	if finalResult.ProcessedStudents != 2 {
		t.Fatalf("ProcessedStudents mismatch: got %d, want 2", finalResult.ProcessedStudents)
	}
	if finalResult.TotalStudents != 2 {
		t.Fatalf("TotalStudents mismatch: got %d, want 2", finalResult.TotalStudents)
	}
	if finalResult.CreatedAt.IsZero() {
		t.Fatalf("CreatedAt should be set")
	}

	// Проверяем ResultDetails diff1 diff2
	v1, ok := finalResult.ResultDetails["diff1"]
	if !ok || v1 == nil {
		t.Fatalf("diff1 missing: %#v", finalResult.ResultDetails)
	}
	d1, ok := v1.(AssessmentDiff)
	if !ok {
		t.Fatalf("diff1 wrong type %T: %#v", v1, v1)
	}

	v2, ok := finalResult.ResultDetails["diff2"]
	if !ok || v2 == nil {
		t.Fatalf("diff2 missing: %#v", finalResult.ResultDetails)
	}
	d2, ok := v2.(AssessmentDiff)
	if !ok {
		t.Fatalf("diff2 wrong type %T: %#v", v2, v2)
	}

	// Проверяем основные поля для обоих diff
	assertDiff := func(name string, got AssessmentDiff) {
		if got.StudentID != expectedDiff.StudentID {
			t.Fatalf("%s.StudentID mismatch: got %s, want %s", name, got.StudentID, expectedDiff.StudentID)
		}
		if got.PeriodStart != expectedDiff.PeriodStart {
			t.Fatalf("%s.PeriodStart mismatch: got %s, want %s", name, got.PeriodStart, expectedDiff.PeriodStart)
		}
		if got.PeriodEnd != expectedDiff.PeriodEnd {
			t.Fatalf("%s.PeriodEnd mismatch: got %s, want %s", name, got.PeriodEnd, expectedDiff.PeriodEnd)
		}

		if got.LangDevDiff != expectedDiff.LangDevDiff {
			t.Fatalf("%s.LangDevDiff mismatch: got %+v, want %+v", name, got.LangDevDiff, expectedDiff.LangDevDiff)
		}
		if got.CommFuncsDiff != expectedDiff.CommFuncsDiff {
			t.Fatalf("%s.CommFuncsDiff mismatch: got %+v, want %+v", name, got.CommFuncsDiff, expectedDiff.CommFuncsDiff)
		}
		if got.VocabularyDiff != expectedDiff.VocabularyDiff {
			t.Fatalf("%s.VocabularyDiff mismatch: got %+v, want %+v", name, got.VocabularyDiff, expectedDiff.VocabularyDiff)
		}
		if got.GeneralProgress != expectedDiff.GeneralProgress {
			t.Fatalf("%s.GeneralProgress mismatch: got %+v, want %+v", name, got.GeneralProgress, expectedDiff.GeneralProgress)
		}
	}

	assertDiff("diff1", d1)
	assertDiff("diff2", d2)

	cancel()
}

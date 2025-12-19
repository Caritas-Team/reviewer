package processing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Caritas-Team/reviewer/internal/config"
	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/model"
)

// CheckProcessingResultFields проверяет все поля структуры ProcessingResult
func CheckProcessingResultFields(t *testing.T, want, got model.ProcessingResult) error {
	if want.RequestID != got.RequestID {
		return errors.New("RequestID mismatch")
	}
	if want.Status != got.Status {
		return errors.New("Status mismatch")
	}
	if want.ErrorMessage != got.ErrorMessage {
		return errors.New("ErrorMessage mismatch")
	}
	if want.ProcessedStudents != got.ProcessedStudents {
		return errors.New("ProcessedStudents mismatch")
	}
	if want.TotalStudents != got.TotalStudents {
		return errors.New("TotalStudents mismatch")
	}

	// Проверка ResultDetails
	if err := ManualCompareResultDetails(t, want.ResultDetails, got.ResultDetails); err != nil {
		return err
	}

	// Проверка Errors
	if len(want.Errors) != len(got.Errors) {
		return errors.New("Errors length mismatch")
	}
	for k, v := range want.Errors {
		if gv, ok := got.Errors[k]; !ok || v != gv {
			return errors.New("Errors content mismatch")
		}
	}

	// Проверка CreatedAt
	if !want.CreatedAt.Equal(got.CreatedAt) {
		return errors.New("CreatedAt mismatch")
	}

	return nil
}

// ManualCompareResultDetails сравнивает поля ResultDetails вручную
func ManualCompareResultDetails(t *testing.T, want, got interface{}) error {
	wantDetails := want.(map[string]interface{})
	gotDetails := got.(map[string]interface{})

	// Проверка поля "diff"
	wantDiff := wantDetails["diff"].(AssessmentDiff)
	gotDiff := gotDetails["diff"].(AssessmentDiff)

	// Сравниваем поля структуры AssessmentDiff
	if wantDiff.StudentID != gotDiff.StudentID {
		return errors.New("StudentID mismatch")
	}
	if wantDiff.PeriodStart != gotDiff.PeriodStart {
		return errors.New("PeriodStart mismatch")
	}
	if wantDiff.PeriodEnd != gotDiff.PeriodEnd {
		return errors.New("PeriodEnd mismatch")
	}

	// Сравниваем LangDevDiff
	if wantDiff.LangDevDiff.PreintentionalDelta != gotDiff.LangDevDiff.PreintentionalDelta ||
		wantDiff.LangDevDiff.ProtolanguageDelta != gotDiff.LangDevDiff.ProtolanguageDelta ||
		wantDiff.LangDevDiff.HolophraseDelta != gotDiff.LangDevDiff.HolophraseDelta ||
		wantDiff.LangDevDiff.PhraseDelta != gotDiff.LangDevDiff.PhraseDelta {
		return errors.New("LangDevDiff mismatch")
	}

	// Сравниваем CommFuncsDiff
	if wantDiff.CommFuncsDiff.ControlDelta != gotDiff.CommFuncsDiff.ControlDelta ||
		wantDiff.CommFuncsDiff.ObtainingDesiredDelta != gotDiff.CommFuncsDiff.ObtainingDesiredDelta ||
		wantDiff.CommFuncsDiff.SocialInteractionDelta != gotDiff.CommFuncsDiff.SocialInteractionDelta ||
		wantDiff.CommFuncsDiff.InformationExchangeDelta != gotDiff.CommFuncsDiff.InformationExchangeDelta {
		return errors.New("CommFuncsDiff mismatch")
	}

	// Сравниваем VocabularyDiff
	if wantDiff.VocabularyDiff.ActiveWordsDelta != gotDiff.VocabularyDiff.ActiveWordsDelta ||
		wantDiff.VocabularyDiff.TotalWordsDelta != gotDiff.VocabularyDiff.TotalWordsDelta {
		return errors.New("VocabularyDiff mismatch")
	}

	// Сравниваем GeneralProgress
	if wantDiff.GeneralProgress.AverageProgress != gotDiff.GeneralProgress.AverageProgress {
		return errors.New("GeneralProgress mismatch")
	}

	return nil
}

// Тест на успешную обработку пары документов
func TestWorkerPool_ProcessPair_Success(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputChan := make(chan []model.StudentPair, 10)
	resultChan := make(chan model.ProcessingResult, 10)
	errorChan := make(chan model.ProcessingError, 10)

	// Создаем фиктивную конфигурацию для логгера
	mockCfg := config.Config{
		Logging: config.Logging{
			Level:  "debug",
			Format: "json",
		},
	}

	// Создаем логгер
	testLogger := logger.NewLogger(mockCfg)

	wp := NewWorkerPool(ctx, inputChan, resultChan, errorChan, 2, testLogger)
	wp.Start()

	// Установим фиксированное время
	fixedTimeBefore := time.Date(2025, 12, 18, 13, 59, 22, 0, time.UTC)
	fixedTimeAfter := time.Date(2025, 12, 19, 13, 59, 22, 0, time.UTC)

	beforeDoc := &model.AssessmentDocument{
		Metadata: model.AssessmentMetadata{
			StudentID:      "S12345",
			Date:           fixedTimeBefore,
			AssessmentType: "before",
			FileName:       "document_before.pdf",
		},
		LanguageLevels: model.LanguageDevelopment{
			Preintentional: model.Preintentional{
				Activity: 80.0,
			},
			Protolanguage: model.LanguageActivity{
				Activity:   60.0,
				Initiative: 50.0,
			},
			Holophrase: model.LanguageActivity{
				Activity:   40.0,
				Initiative: 30.0,
			},
			Phrase: model.LanguageActivity{
				Activity:   20.0,
				Initiative: 10.0,
			},
		},
		CommunicativeFuncs: model.CommunicativeFunctions{
			Control:             70.0,
			ObtainingDesired:    50.0,
			SocialInteraction:   30.0,
			InformationExchange: 10.0,
		},
		Vocabulary: model.VocabularyData{
			ActiveWordsCount: 100,
			AdditionalWords:  []string{"apple", "banana"},
			TotalWordsCount:  200,
		},
	}

	afterDoc := &model.AssessmentDocument{
		Metadata: model.AssessmentMetadata{
			StudentID:      "S12345",
			Date:           fixedTimeAfter,
			AssessmentType: "after",
			FileName:       "document_after.pdf",
		},
		LanguageLevels: model.LanguageDevelopment{
			Preintentional: model.Preintentional{
				Activity: 90.5,
			},
			Protolanguage: model.LanguageActivity{
				Activity:   70.0,
				Initiative: 60.0,
			},
			Holophrase: model.LanguageActivity{
				Activity:   50.0,
				Initiative: 40.0,
			},
			Phrase: model.LanguageActivity{
				Activity:   30.0,
				Initiative: 20.0,
			},
		},
		CommunicativeFuncs: model.CommunicativeFunctions{
			Control:             80.0,
			ObtainingDesired:    60.0,
			SocialInteraction:   40.0,
			InformationExchange: 20.0,
		},
		Vocabulary: model.VocabularyData{
			ActiveWordsCount: 120,
			AdditionalWords:  []string{"orange", "grapefruit"},
			TotalWordsCount:  240,
		},
	}

	// Готовим пару документов
	pair := model.StudentPair{
		RequestID: "test-request",
		StudentID: "test-student",
		Before:    beforeDoc,
		After:     afterDoc,
	}

	// Отправляем пару на обработку
	inputChan <- []model.StudentPair{pair}

	// Ждём результата
	result := <-resultChan

	// Ожидаемый результат
	expectedResult := model.ProcessingResult{
		RequestID:         "test-request",
		Status:            "processing",
		ErrorMessage:      "",
		ProcessedStudents: 0,
		TotalStudents:     0,
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
		Errors:    map[string]string{},
		CreatedAt: time.Time{}, // Пустое время, если не требуется проверка
	}

	// Комплексная проверка всех полей
	if err := CheckProcessingResultFields(t, expectedResult, result); err != nil {
		t.Fatalf("Comparison failed: %v", err)
	}

	wp.Stop()
}

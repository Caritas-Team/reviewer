package processing

import (
	"fmt"

	"github.com/Caritas-Team/reviewer/internal/model"
)

// DiffCalculator реализует вычисление различий между документами
type DiffCalculator struct{}

// AssessmentDiff - результат сравнения "до/после"
type AssessmentDiff struct {
	StudentID       string          `json:"student_id"`       // ID студента
	PeriodStart     string          `json:"period_start"`     // Начальная дата периода
	PeriodEnd       string          `json:"period_end"`       // Конечная дата периода
	LangDevDiff     LangDevDiff     `json:"lang_dev_diff"`    // Различия в развитии языка
	CommFuncsDiff   CommFuncsDiff   `json:"comm_funcs_diff"`  // Различия в коммуникативных функциях
	VocabularyDiff  VocabularyDiff  `json:"vocab_diff"`       // Различия в словаре
	GeneralProgress GeneralProgress `json:"general_progress"` // Общий прогресс
}

// LangDevDiff различия в развитии языка
type LangDevDiff struct {
	PreintentionalDelta float64
	ProtolanguageDelta  float64
	HolophraseDelta     float64
	PhraseDelta         float64
}

// CommFuncsDiff различия в коммуникативных функциях
type CommFuncsDiff struct {
	ControlDelta             float64
	ObtainingDesiredDelta    float64
	SocialInteractionDelta   float64
	InformationExchangeDelta float64
}

// VocabularyDiff различия в словаре
type VocabularyDiff struct {
	ActiveWordsDelta int
	TotalWordsDelta  int
}

// GeneralProgress общий прогресс
type GeneralProgress struct {
	AverageProgress float64
}

// Calculate вычисляет различия между двумя документами
func (dc *DiffCalculator) Calculate(before, after *model.AssessmentDocument) (*AssessmentDiff, error) {
	if before == nil || after == nil {
		return nil, fmt.Errorf("both documents are required")
	}

	// Создаем структуру для различий и заполняем метаданные
	diff := &AssessmentDiff{
		StudentID:   before.Metadata.StudentID,
		PeriodStart: before.Metadata.Date.Format("2006-01-02"),
		PeriodEnd:   after.Metadata.Date.Format("2006-01-02"),
	}

	// 1. Сравнение уровней языкового развития
	diff.LangDevDiff = dc.compareLanguageDevelopment(before.LanguageLevels, after.LanguageLevels)

	// 2. Сравнение коммуникативных функций
	diff.CommFuncsDiff = dc.compareCommunicativeFunctions(before.CommunicativeFuncs, after.CommunicativeFuncs)

	// 3. Сравнение словарного запаса
	diff.VocabularyDiff = dc.compareVocabulary(before.Vocabulary, after.Vocabulary)

	// 4. Общий прогресс
	diff.GeneralProgress.AverageProgress = dc.calculateAverageProgress(diff.LangDevDiff, diff.CommFuncsDiff)

	return diff, nil
}

// compareLanguageDevelopment сравнивает уровни языкового развития
func (dc *DiffCalculator) compareLanguageDevelopment(before, after model.LanguageDevelopment) LangDevDiff {
	return LangDevDiff{
		PreintentionalDelta: after.Preintentional.Activity - before.Preintentional.Activity,
		ProtolanguageDelta:  after.Protolanguage.Activity - before.Protolanguage.Activity,
		HolophraseDelta:     after.Holophrase.Activity - before.Holophrase.Activity,
		PhraseDelta:         after.Phrase.Activity - before.Phrase.Activity,
	}
}

// compareCommunicativeFunctions сравнивает коммуникативные функции
func (dc *DiffCalculator) compareCommunicativeFunctions(before, after model.CommunicativeFunctions) CommFuncsDiff {
	return CommFuncsDiff{
		ControlDelta:             after.Control - before.Control,
		ObtainingDesiredDelta:    after.ObtainingDesired - before.ObtainingDesired,
		SocialInteractionDelta:   after.SocialInteraction - before.SocialInteraction,
		InformationExchangeDelta: after.InformationExchange - before.InformationExchange,
	}
}

// compareVocabulary сравнивает словарный запас
func (dc *DiffCalculator) compareVocabulary(before, after model.VocabularyData) VocabularyDiff {
	return VocabularyDiff{
		ActiveWordsDelta: after.ActiveWordsCount - before.ActiveWordsCount,
		TotalWordsDelta:  after.TotalWordsCount - before.TotalWordsCount,
	}
}

// calculateAverageProgress вычисляет средний общий прогресс
func (dc *DiffCalculator) calculateAverageProgress(langDevDiff LangDevDiff, commFuncsDiff CommFuncsDiff) float64 {
	values := []float64{
		langDevDiff.PreintentionalDelta,
		langDevDiff.ProtolanguageDelta,
		langDevDiff.HolophraseDelta,
		langDevDiff.PhraseDelta,
		commFuncsDiff.ControlDelta,
		commFuncsDiff.ObtainingDesiredDelta,
		commFuncsDiff.SocialInteractionDelta,
		commFuncsDiff.InformationExchangeDelta,
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))

}

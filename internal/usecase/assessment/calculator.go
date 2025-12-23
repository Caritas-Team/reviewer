package assessment

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
	PreintentionalDelta float64 `json:"preintentional_delta"`
	ProtolanguageDelta  float64 `json:"protolanguage_delta"`
	HolophraseDelta     float64 `json:"holophrase_delta"`
	PhraseDelta         float64 `json:"phrase_delta"`
}

// CommFuncsDiff различия в коммуникативных функциях
type CommFuncsDiff struct {
	ControlDelta             float64 `json:"control_delta"`
	ObtainingDesiredDelta    float64 `json:"obtaining_desired_delta"`
	SocialInteractionDelta   float64 `json:"social_interaction_delta"`
	InformationExchangeDelta float64 `json:"information_exchange_delta"`
}

// VocabularyDiff различия в словаре
type VocabularyDiff struct {
	ActiveWordsDelta int `json:"active_words_delta"`
	TotalWordsDelta  int `json:"total_words_delta"`
}

// GeneralProgress общий прогресс
type GeneralProgress struct {
	AverageProgress float64 `json:"average_progress"`
}

// Calculate вычисляет различия между двумя документами
func (dc *DiffCalculator) Calculate(before, after *model.AssessmentDocument) (AssessmentDiff, error) {
	if before == nil || after == nil {
		return AssessmentDiff{}, fmt.Errorf("both documents are required")
	}

	// Создаем структуру для различий и заполняем метаданные
	diff := AssessmentDiff{
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
	diff.GeneralProgress.AverageProgress = dc.avg(diff.LangDevDiff, diff.CommFuncsDiff)

	return diff, nil
}

// compareLanguageDevelopment сравнивает уровни языкового развития
func (dc *DiffCalculator) compareLanguageDevelopment(before, after model.LanguageDevelopment) LangDevDiff {
	return LangDevDiff{
		PreintentionalDelta: float64(after.Preintentional.Activity - before.Preintentional.Activity),
		ProtolanguageDelta:  float64(after.Protolanguage.Activity - before.Protolanguage.Activity),
		HolophraseDelta:     float64(after.Holophrase.Activity - before.Holophrase.Activity),
		PhraseDelta:         float64(after.Phrase.Activity - before.Phrase.Activity),
	}
}

// compareCommunicativeFunctions сравнивает коммуникативные функции
func (dc *DiffCalculator) compareCommunicativeFunctions(before, after model.CommunicativeFunctions) CommFuncsDiff {
	return CommFuncsDiff{
		ControlDelta:             float64(after.Control - before.Control),
		ObtainingDesiredDelta:    float64(after.ObtainingDesired - before.ObtainingDesired),
		SocialInteractionDelta:   float64(after.SocialInteraction - before.SocialInteraction),
		InformationExchangeDelta: float64(after.InformationExchange - before.InformationExchange),
	}
}

// compareVocabulary сравнивает словарный запас
func (dc *DiffCalculator) compareVocabulary(before, after model.VocabularyData) VocabularyDiff {
	return VocabularyDiff{
		ActiveWordsDelta: after.ActiveWordsCount - before.ActiveWordsCount,
		TotalWordsDelta:  after.TotalWordsCount - before.TotalWordsCount,
	}
}

// avg вычисляет средний общий прогресс
func (dc *DiffCalculator) avg(lang LangDevDiff, comm CommFuncsDiff) float64 {
	values := []float64{
		lang.PreintentionalDelta, lang.ProtolanguageDelta, lang.HolophraseDelta, lang.PhraseDelta,
		comm.ControlDelta, comm.ObtainingDesiredDelta, comm.SocialInteractionDelta, comm.InformationExchangeDelta,
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

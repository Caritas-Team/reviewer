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

// GroupAverage - средние значения по группе за конкретную дату
type GroupAverage struct {
	Date               string                       `json:"date"`                // Дата в формате "2006-01-02"
	LanguageLevels     model.LanguageDevelopment    `json:"language_levels"`     // Средние значения уровней развития
	CommunicativeFuncs model.CommunicativeFunctions `json:"communicative_funcs"` // Средние значения коммуникативных функций
	Vocabulary         model.VocabularyData         `json:"vocabulary"`          // Средние значения словарного запаса
	StudentsCount      int                          `json:"students_count"`      // Количество студентов в группе
}

// GroupProgress - прогресс группы по датам
type GroupProgress struct {
	PeriodStart    string              `json:"period_start"`    // Начальная дата
	PeriodEnd      string              `json:"period_end"`      // Конечная дата
	LanguageLevels GroupLanguageLevels `json:"language_levels"` // Прогресс по уровням
}

type GroupDiff struct {
	LangDevDiff   LangDevDiffWithInitiative `json:"lang_dev_diff"`
	CommFuncsDiff CommFuncsDiff             `json:"comm_funcs_diff"`
}

// GroupLanguageLevels прогресс по четырем параметрам
type GroupLanguageLevels struct {
	Preintentional GroupLevelProgress `json:"preintentional"` // Доинтенциональная коммуникация
	Protolanguage  GroupLevelProgress `json:"protolanguage"`  // Протоязык
	Holophrase     GroupLevelProgress `json:"holophrase"`     // Голофраза
	Phrase         GroupLevelProgress `json:"phrase"`         // Фраза
}

// GroupLevelProgress прогресс отдельного уровня
type GroupLevelProgress struct {
	ActivityPercent float64 `json:"activity_percent"` // Прогресс в процентах
}

// Calculate вычисляет различия между двумя документами
func (dc *DiffCalculator) Calculate(before, after *model.AssessmentDocument) (AssessmentDiff, error) {
	if before == nil || after == nil {
		return AssessmentDiff{}, fmt.Errorf("оба документа обязательны для сравнения")
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

// CalculateGroupAverage вычисляет средние значения по группе документов
func (dc *DiffCalculator) CalculateGroupAverage(documents []*model.AssessmentDocument) (GroupAverage, error) {
	if len(documents) == 0 {
		return GroupAverage{}, fmt.Errorf("empty documents list")
	}

	date := documents[0].Metadata.Date.Format("2006-01-02")

	var sumLangDev model.LanguageDevelopment
	var sumCommFuncs model.CommunicativeFunctions
	var sumVocab model.VocabularyData
	count := len(documents)

	for _, doc := range documents {
		if doc.Metadata.Date.Format("2006-01-02") != date {
			return GroupAverage{}, fmt.Errorf("documents have different dates")
		}

		sumLangDev.Preintentional.Activity += doc.LanguageLevels.Preintentional.Activity
		sumLangDev.Protolanguage.Activity += doc.LanguageLevels.Protolanguage.Activity
		sumLangDev.Protolanguage.Initiative += doc.LanguageLevels.Protolanguage.Initiative
		sumLangDev.Holophrase.Activity += doc.LanguageLevels.Holophrase.Activity
		sumLangDev.Holophrase.Initiative += doc.LanguageLevels.Holophrase.Initiative
		sumLangDev.Phrase.Activity += doc.LanguageLevels.Phrase.Activity
		sumLangDev.Phrase.Initiative += doc.LanguageLevels.Phrase.Initiative

		sumCommFuncs.Control += doc.CommunicativeFuncs.Control
		sumCommFuncs.ObtainingDesired += doc.CommunicativeFuncs.ObtainingDesired
		sumCommFuncs.SocialInteraction += doc.CommunicativeFuncs.SocialInteraction
		sumCommFuncs.InformationExchange += doc.CommunicativeFuncs.InformationExchange

		sumVocab.ActiveWordsCount += doc.Vocabulary.ActiveWordsCount
		sumVocab.TotalWordsCount += doc.Vocabulary.TotalWordsCount
	}

	avgLangDev := model.LanguageDevelopment{
		Preintentional: model.Preintentional{
			Activity: sumLangDev.Preintentional.Activity / float64(count),
		},
		Protolanguage: model.LanguageActivity{
			Activity:   sumLangDev.Protolanguage.Activity / float64(count),
			Initiative: sumLangDev.Protolanguage.Initiative / float64(count),
		},
		Holophrase: model.LanguageActivity{
			Activity:   sumLangDev.Holophrase.Activity / float64(count),
			Initiative: sumLangDev.Holophrase.Initiative / float64(count),
		},
		Phrase: model.LanguageActivity{
			Activity:   sumLangDev.Phrase.Activity / float64(count),
			Initiative: sumLangDev.Phrase.Initiative / float64(count),
		},
	}

	avgCommFuncs := model.CommunicativeFunctions{
		Control:             sumCommFuncs.Control / float64(count),
		ObtainingDesired:    sumCommFuncs.ObtainingDesired / float64(count),
		SocialInteraction:   sumCommFuncs.SocialInteraction / float64(count),
		InformationExchange: sumCommFuncs.InformationExchange / float64(count),
	}

	avgVocab := model.VocabularyData{
		ActiveWordsCount: sumVocab.ActiveWordsCount / count,
		TotalWordsCount:  sumVocab.TotalWordsCount / count,
		AdditionalWords:  nil, // Не суммируем, это список строк
	}

	return GroupAverage{
		Date:               date,
		LanguageLevels:     avgLangDev,
		CommunicativeFuncs: avgCommFuncs,
		Vocabulary:         avgVocab,
		StudentsCount:      count,
	}, nil
}

// CalculateGroupProgress рассчитывает прогресс группы между двумя датами
func (dc *DiffCalculator) CalculateGroupProgress(earlierAvg, laterAvg GroupAverage) (GroupProgress, error) {
	if earlierAvg.Date == "" || laterAvg.Date == "" {
		return GroupProgress{}, fmt.Errorf("both group averages must have valid dates")
	}

	// Рассчитываем процентный прогресс для каждого уровня
	preintentionalPercent := calculatePercentChange(
		earlierAvg.LanguageLevels.Preintentional.Activity,
		laterAvg.LanguageLevels.Preintentional.Activity,
	)

	protolanguagePercent := calculatePercentChange(
		earlierAvg.LanguageLevels.Protolanguage.Activity,
		laterAvg.LanguageLevels.Protolanguage.Activity,
	)

	holophrasePercent := calculatePercentChange(
		earlierAvg.LanguageLevels.Holophrase.Activity,
		laterAvg.LanguageLevels.Holophrase.Activity,
	)

	phrasePercent := calculatePercentChange(
		earlierAvg.LanguageLevels.Phrase.Activity,
		laterAvg.LanguageLevels.Phrase.Activity,
	)

	return GroupProgress{
		PeriodStart: earlierAvg.Date,
		PeriodEnd:   laterAvg.Date,
		LanguageLevels: GroupLanguageLevels{
			Preintentional: GroupLevelProgress{ActivityPercent: preintentionalPercent},
			Protolanguage:  GroupLevelProgress{ActivityPercent: protolanguagePercent},
			Holophrase:     GroupLevelProgress{ActivityPercent: holophrasePercent},
			Phrase:         GroupLevelProgress{ActivityPercent: phrasePercent},
		},
	}, nil
}

// calculatePercentChange рассчитывает процентное изменение
func calculatePercentChange(before, after float64) float64 {
	if before == 0 {

		if after == 0 {
			return 0
		}
		return 100.0
	}
	return ((after - before) / before) * 100.0
}

type LangActivityDelta struct {
	Activity   float64 `json:"activity"`
	Initiative float64 `json:"initiative"`
}

type LangDevDiffWithInitiative struct {
	PreintentionalDelta float64           `json:"preintentional_delta"`
	ProtolanguageDelta  LangActivityDelta `json:"protolanguage_delta"`
	HolophraseDelta     LangActivityDelta `json:"holophrase_delta"`
	PhraseDelta         LangActivityDelta `json:"phrase_delta"`
}

// вычисляет абсолютные разницы между двумя групповыми средними
func (dc *DiffCalculator) CalculateGroupDiff(earlier, later GroupAverage) GroupDiff {
	return GroupDiff{
		CommFuncsDiff: CommFuncsDiff{
			ControlDelta:             later.CommunicativeFuncs.Control - earlier.CommunicativeFuncs.Control,
			ObtainingDesiredDelta:    later.CommunicativeFuncs.ObtainingDesired - earlier.CommunicativeFuncs.ObtainingDesired,
			SocialInteractionDelta:   later.CommunicativeFuncs.SocialInteraction - earlier.CommunicativeFuncs.SocialInteraction,
			InformationExchangeDelta: later.CommunicativeFuncs.InformationExchange - earlier.CommunicativeFuncs.InformationExchange,
		},
		LangDevDiff: LangDevDiffWithInitiative{
			PreintentionalDelta: later.LanguageLevels.Preintentional.Activity - earlier.LanguageLevels.Preintentional.Activity,
			ProtolanguageDelta: LangActivityDelta{
				Activity:   later.LanguageLevels.Protolanguage.Activity - earlier.LanguageLevels.Protolanguage.Activity,
				Initiative: later.LanguageLevels.Protolanguage.Initiative - earlier.LanguageLevels.Protolanguage.Initiative,
			},
			HolophraseDelta: LangActivityDelta{
				Activity:   later.LanguageLevels.Holophrase.Activity - earlier.LanguageLevels.Holophrase.Activity,
				Initiative: later.LanguageLevels.Holophrase.Initiative - earlier.LanguageLevels.Holophrase.Initiative,
			},
			PhraseDelta: LangActivityDelta{
				Activity:   later.LanguageLevels.Phrase.Activity - earlier.LanguageLevels.Phrase.Activity,
				Initiative: later.LanguageLevels.Phrase.Initiative - earlier.LanguageLevels.Phrase.Initiative,
			},
		},
	}
}

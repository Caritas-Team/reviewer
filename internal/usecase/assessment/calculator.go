package assessment

import (
	"fmt"
	"math"

	"github.com/Caritas-Team/reviewer/internal/model"
)

// DiffCalculator реализует вычисление различий между документами
type DiffCalculator struct{}

// AssessmentDiff - результат сравнения "до/после"
type AssessmentDiff struct {
	StudentID       string          `json:"student_id"`   // ID студента
	PeriodStart     string          `json:"period_start"` // Начальная дата периода
	PeriodEnd       string          `json:"period_end"`   // Конечная дата периода
	BirthDate       string          `json:"birth_date"`
	LangDevDiff     LangDevDiff     `json:"lang_dev_diff"`    // Различия в развитии языка
	CommFuncsDiff   CommFuncsDiff   `json:"comm_funcs_diff"`  // Различия в коммуникативных функциях
	VocabularyDiff  VocabularyDiff  `json:"vocab_diff"`       // Различия в словаре
	GeneralProgress GeneralProgress `json:"general_progress"` // Общий прогресс

	Diagnosis         string `json:"diagnosis,omitempty"`
	LivingSituation   string `json:"living_situation,omitempty"`
	FamilyDescription string `json:"family_description,omitempty"`

	BeforeData IndividualData `json:"before_data,omitempty"`
	AfterData  IndividualData `json:"after_data,omitempty"`

	FastMessages        []string       `json:"fast_messages,omitempty"`
	CommunicationCounts map[string]int `json:"communication_counts,omitempty"`
}

// IndividualData Данные из DiagramBlock и NewAct
type IndividualData struct {
	DiagramRaw     model.DiagramRaw                  `json:"diagram_raw"`
	NewAct01       string                            `json:"newAct01"`
	NewAct02       string                            `json:"newAct02"`
	NewAct03       string                            `json:"newAct03"`
	NewAct04       string                            `json:"newAct04"`
	OtherActBlocks map[string]model.ActBlockOtherRaw `json:"other_act_blocks,omitempty"`
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
	ActiveWordsDelta     int      `json:"active_words_delta"`
	TotalWordsDelta      int      `json:"total_words_delta"`
	VerbalWordsDelta     int      `json:"verbal_words_delta"`
	AdditionalWordsDelta int      `json:"additional_words_delta"`
	VerbalWordsCount     int      `json:"verbal_words_count"`
	AdditionalWordsCount int      `json:"additional_words_count"`
	CommunicationWays    []string `json:"communication_ways,omitempty"`
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
	ActBlock           model.ActBlockData           `json:"act_block,omitempty"` // Среднее значение контроля
	StudentsCount      int                          `json:"students_count"`      // Количество студентов в группе

}

// GroupProgress - прогресс группы по датам
type GroupProgress struct {
	PeriodStart    string              `json:"period_start"`             // Начальная дата
	PeriodEnd      string              `json:"period_end"`               // Конечная дата
	LanguageLevels GroupLanguageLevels `json:"language_levels"`          // Прогресс по уровням
	ActBlockDiff   GroupActBlockDiff   `json:"act_block_diff,omitempty"` // Прогресс функции контроля
}

// GroupVocabularyProgress - прогресс группы по словарному запасу
type GroupVocabularyProgress struct {
	NewWordsCount           int      `json:"new_words_count"`
	NewWordsDiff            int      `json:"new_words_diff"`
	VerbalWordsCount        int      `json:"verbal_words_count"`
	VerbalWordsDiff         int      `json:"verbal_words_diff"`
	AdditionalWordsCount    int      `json:"additional_words_count"`
	NonDictionaryWordsDiff  int      `json:"non_dictionary_words_diff"`
	CommonCommunicationWays []string `json:"common_communication_ways,omitempty"`
}

// GroupActBlockDiff - разница в ActBlock данных между двумя группами
type GroupActBlockDiff struct {
	Prot GroupActivityDiff `json:"prot,omitempty"`
	Gol  GroupActivityDiff `json:"gol,omitempty"`
	Fra  GroupActivityDiff `json:"fra,omitempty"`
}

// GroupActivityDiff - разница в активности между группами
type GroupActivityDiff struct {
	ActivityDelta   float64 `json:"activity_delta,omitempty"`
	InitiativeDelta float64 `json:"initiative_delta,omitempty"`
	FrequencyDelta  float64 `json:"frequency_delta,omitempty"`
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
		StudentID:         before.Metadata.StudentID,
		PeriodStart:       before.Metadata.Date.Format("2006-01-02"),
		PeriodEnd:         after.Metadata.Date.Format("2006-01-02"),
		BirthDate:         after.BirthDate,
		Diagnosis:         after.Diagnosis,
		LivingSituation:   after.LivingSituation,
		FamilyDescription: after.FamilyDescription,
		BeforeData: IndividualData{
			DiagramRaw:     before.DiagramRaw,
			NewAct01:       before.NewAct01Raw,
			NewAct02:       before.NewAct02Raw,
			NewAct03:       before.NewAct03Raw,
			NewAct04:       before.NewAct04Raw,
			OtherActBlocks: before.OtherActBlocks,
		},
		AfterData: IndividualData{
			DiagramRaw:     after.DiagramRaw,
			NewAct01:       after.NewAct01Raw,
			NewAct02:       after.NewAct02Raw,
			NewAct03:       after.NewAct03Raw,
			NewAct04:       after.NewAct04Raw,
			OtherActBlocks: after.OtherActBlocks,
		},
		FastMessages:        after.FastMessages,
		CommunicationCounts: after.CommunicationCounts,
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
		ActiveWordsDelta:     after.ActiveWordsCount - before.ActiveWordsCount,
		TotalWordsDelta:      after.TotalWordsCount - before.TotalWordsCount,
		VerbalWordsDelta:     after.VerbalWordsCount - before.VerbalWordsCount,
		AdditionalWordsDelta: len(after.AdditionalWords) - len(before.AdditionalWords),
		VerbalWordsCount:     after.VerbalWordsCount,
		AdditionalWordsCount: len(after.AdditionalWords),
		CommunicationWays:    after.CommunicationWays,
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
		sumLangDev.Preintentional.Initiative += doc.LanguageLevels.Preintentional.Initiative
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
			Activity:   roundToOneDecimal(sumLangDev.Preintentional.Activity / float64(count)),
			Initiative: roundToOneDecimal(sumLangDev.Preintentional.Initiative / float64(count)),
		},
		Protolanguage: model.LanguageActivity{
			Activity:   roundToOneDecimal(sumLangDev.Protolanguage.Activity / float64(count)),
			Initiative: roundToOneDecimal(sumLangDev.Protolanguage.Initiative / float64(count)),
		},
		Holophrase: model.LanguageActivity{
			Activity:   roundToOneDecimal(sumLangDev.Holophrase.Activity / float64(count)),
			Initiative: roundToOneDecimal(sumLangDev.Holophrase.Initiative / float64(count)),
		},
		Phrase: model.LanguageActivity{
			Activity:   roundToOneDecimal(sumLangDev.Phrase.Activity / float64(count)),
			Initiative: roundToOneDecimal(sumLangDev.Phrase.Initiative / float64(count)),
		},
	}

	avgCommFuncs := model.CommunicativeFunctions{
		Control:             roundToOneDecimal(sumCommFuncs.Control / float64(count)),
		ObtainingDesired:    roundToOneDecimal(sumCommFuncs.ObtainingDesired / float64(count)),
		SocialInteraction:   roundToOneDecimal(sumCommFuncs.SocialInteraction / float64(count)),
		InformationExchange: roundToOneDecimal(sumCommFuncs.InformationExchange / float64(count)),
	}

	avgVocab := model.VocabularyData{
		ActiveWordsCount: sumVocab.ActiveWordsCount / count,
		TotalWordsCount:  sumVocab.TotalWordsCount / count,
		AdditionalWords:  nil, // Не суммируем, это список строк
	}

	actBlockAvg := dc.calculateGroupActBlockAverage(documents)

	return GroupAverage{
		Date:               date,
		LanguageLevels:     avgLangDev,
		CommunicativeFuncs: avgCommFuncs,
		Vocabulary:         avgVocab,
		ActBlock:           actBlockAvg,
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

	// Рассчитываем разницу в ActBlock данных между группами
	actBlockDiff := GroupActBlockDiff{
		Prot: GroupActivityDiff{
			ActivityDelta:   laterAvg.ActBlock.Prot.ActivityPercent - earlierAvg.ActBlock.Prot.ActivityPercent,
			InitiativeDelta: laterAvg.ActBlock.Prot.InitiativePercent - earlierAvg.ActBlock.Prot.InitiativePercent,
			FrequencyDelta:  laterAvg.ActBlock.Prot.FrequencyPercent - earlierAvg.ActBlock.Prot.FrequencyPercent,
		},
		Gol: GroupActivityDiff{
			ActivityDelta:   laterAvg.ActBlock.Gol.ActivityPercent - earlierAvg.ActBlock.Gol.ActivityPercent,
			InitiativeDelta: laterAvg.ActBlock.Gol.InitiativePercent - earlierAvg.ActBlock.Gol.InitiativePercent,
			FrequencyDelta:  laterAvg.ActBlock.Gol.FrequencyPercent - earlierAvg.ActBlock.Gol.FrequencyPercent,
		},
		Fra: GroupActivityDiff{
			ActivityDelta:   laterAvg.ActBlock.Fra.ActivityPercent - earlierAvg.ActBlock.Fra.ActivityPercent,
			InitiativeDelta: laterAvg.ActBlock.Fra.InitiativePercent - earlierAvg.ActBlock.Fra.InitiativePercent,
			FrequencyDelta:  laterAvg.ActBlock.Fra.FrequencyPercent - earlierAvg.ActBlock.Fra.FrequencyPercent,
		},
	}

	return GroupProgress{
		PeriodStart: earlierAvg.Date,
		PeriodEnd:   laterAvg.Date,
		LanguageLevels: GroupLanguageLevels{
			Preintentional: GroupLevelProgress{ActivityPercent: preintentionalPercent},
			Protolanguage:  GroupLevelProgress{ActivityPercent: protolanguagePercent},
			Holophrase:     GroupLevelProgress{ActivityPercent: holophrasePercent},
			Phrase:         GroupLevelProgress{ActivityPercent: phrasePercent},
		},
		ActBlockDiff: actBlockDiff,
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

// CalculateGroupActBlockDiff вычисляет разницу в ActBlock данных между двумя группами
func (dc *DiffCalculator) CalculateGroupActBlockDiff(beforeDocs, afterDocs []*model.AssessmentDocument) GroupActBlockDiff {
	// Вычисляем средние для каждой группы
	beforeAvg := dc.calculateGroupActBlockAverage(beforeDocs)
	afterAvg := dc.calculateGroupActBlockAverage(afterDocs)

	// Сравниваем средние значения
	return GroupActBlockDiff{
		Prot: GroupActivityDiff{
			ActivityDelta:   afterAvg.Prot.ActivityPercent - beforeAvg.Prot.ActivityPercent,
			InitiativeDelta: afterAvg.Prot.InitiativePercent - beforeAvg.Prot.InitiativePercent,
			FrequencyDelta:  afterAvg.Prot.FrequencyPercent - beforeAvg.Prot.FrequencyPercent,
		},
		Gol: GroupActivityDiff{
			ActivityDelta:   afterAvg.Gol.ActivityPercent - beforeAvg.Gol.ActivityPercent,
			InitiativeDelta: afterAvg.Gol.InitiativePercent - beforeAvg.Gol.InitiativePercent,
			FrequencyDelta:  afterAvg.Gol.FrequencyPercent - beforeAvg.Gol.FrequencyPercent,
		},
		Fra: GroupActivityDiff{
			ActivityDelta:   afterAvg.Fra.ActivityPercent - beforeAvg.Fra.ActivityPercent,
			InitiativeDelta: afterAvg.Fra.InitiativePercent - beforeAvg.Fra.InitiativePercent,
			FrequencyDelta:  afterAvg.Fra.FrequencyPercent - beforeAvg.Fra.FrequencyPercent,
		},
	}

}

// calculateGroupActBlockAverage вычисляет средние ActBlock данные по группе
func (dc *DiffCalculator) calculateGroupActBlockAverage(documents []*model.AssessmentDocument) model.ActBlockData {
	if len(documents) == 0 {
		return model.ActBlockData{}
	}
	// Инициализируем суммы
	var protActivitySum, protInitiativeSum, protFrequencySum float64
	var golActivitySum, golInitiativeSum, golFrequencySum float64
	var fraActivitySum, fraInitiativeSum, fraFrequencySum float64

	// Суммируем данные всех документов
	for _, doc := range documents {
		// Протоязык
		protActivitySum += doc.ActBlock.Prot.ActivityPercent
		protInitiativeSum += doc.ActBlock.Prot.InitiativePercent
		protFrequencySum += doc.ActBlock.Prot.FrequencyPercent

		// Голофраза
		golActivitySum += doc.ActBlock.Gol.ActivityPercent
		golInitiativeSum += doc.ActBlock.Gol.InitiativePercent
		golFrequencySum += doc.ActBlock.Gol.FrequencyPercent

		// Фраза
		fraActivitySum += doc.ActBlock.Fra.ActivityPercent
		fraInitiativeSum += doc.ActBlock.Fra.InitiativePercent
		fraFrequencySum += doc.ActBlock.Fra.FrequencyPercent
	}

	// Вычисляем средние
	count := float64(len(documents))
	return model.ActBlockData{
		Prot: model.ActivityData{
			ActivityPercent:   roundToOneDecimal(protActivitySum / count),
			InitiativePercent: roundToOneDecimal(protInitiativeSum / count),
			FrequencyPercent:  roundToOneDecimal(protFrequencySum / count),
		},
		Gol: model.ActivityData{
			ActivityPercent:   roundToOneDecimal(golActivitySum / count),
			InitiativePercent: roundToOneDecimal(golInitiativeSum / count),
			FrequencyPercent:  roundToOneDecimal(golFrequencySum / count),
		},
		Fra: model.ActivityData{
			ActivityPercent:   roundToOneDecimal(fraActivitySum / count),
			InitiativePercent: roundToOneDecimal(fraInitiativeSum / count),
			FrequencyPercent:  roundToOneDecimal(fraFrequencySum / count),
		},
	}
}

// CalculateGroupVocabularyProgress рассчитывает прогресс по словарю для группы
func (dc *DiffCalculator) CalculateGroupVocabularyProgress(earlierDocs, laterDocs []*model.AssessmentDocument) (GroupVocabularyProgress, error) {
	if len(earlierDocs) == 0 || len(laterDocs) == 0 {
		return GroupVocabularyProgress{}, fmt.Errorf("оба набора документов обязательны")
	}

	var (
		beforeTotalSum      int
		beforeActiveSum     int
		beforeVerbalSum     int
		beforeAdditionalSum int
	)

	var (
		afterTotalSum      int
		afterActiveSum     int
		afterVerbalSum     int
		afterAdditionalSum int
	)

	var allLaterCommunicationWays [][]string

	for _, doc := range earlierDocs {
		beforeTotalSum += doc.Vocabulary.TotalWordsCount
		beforeActiveSum += doc.Vocabulary.ActiveWordsCount
		beforeVerbalSum += doc.Vocabulary.VerbalWordsCount
		beforeAdditionalSum += len(doc.Vocabulary.AdditionalWords)
	}

	for _, doc := range laterDocs {
		afterTotalSum += doc.Vocabulary.TotalWordsCount
		afterActiveSum += doc.Vocabulary.ActiveWordsCount
		afterVerbalSum += doc.Vocabulary.VerbalWordsCount
		afterAdditionalSum += len(doc.Vocabulary.AdditionalWords)
		allLaterCommunicationWays = append(allLaterCommunicationWays, doc.Vocabulary.CommunicationWays)
	}

	newWordsAfter := afterActiveSum + afterVerbalSum + afterAdditionalSum
	newWordsBefore := beforeActiveSum + beforeVerbalSum + beforeAdditionalSum
	newWordsDiff := newWordsAfter - newWordsBefore

	verbalWordsCount := afterVerbalSum

	verbalWordsDiff := afterVerbalSum - beforeVerbalSum

	additionalWordsCount := afterAdditionalSum

	additionalWordsDiff := afterAdditionalSum - beforeAdditionalSum

	allCommunicationWays := dc.findCommonCommunicationWays(allLaterCommunicationWays)

	return GroupVocabularyProgress{
		NewWordsCount:           newWordsAfter,
		NewWordsDiff:            newWordsDiff,         // Разница новых слов
		VerbalWordsCount:        verbalWordsCount,     // Вербальные слова (последняя дата)
		VerbalWordsDiff:         verbalWordsDiff,      // Разница вербальных слов
		AdditionalWordsCount:    additionalWordsCount, // Слова не из словаря (последняя дата)
		NonDictionaryWordsDiff:  additionalWordsDiff,  // Разница слов не из словаря
		CommonCommunicationWays: allCommunicationWays, // Все уникальные способы общения
	}, nil
}

// findCommonCommunicationWays находит общие способы общения для всех студентов
func (dc *DiffCalculator) findCommonCommunicationWays(allWays [][]string) []string {
	if len(allWays) == 0 {
		return nil
	}

	common := make([]string, len(allWays[0]))
	copy(common, allWays[0])

	uniqueWays := make(map[string]bool)

	for _, studentWays := range allWays {
		for _, way := range studentWays {
			if way != "" {
				uniqueWays[way] = true
			}
		}
	}

	result := make([]string, 0, len(uniqueWays))
	for way := range uniqueWays {
		result = append(result, way)
	}

	return result
}

func roundToOneDecimal(v float64) float64 {
	return math.Round(v*10) / 10
}

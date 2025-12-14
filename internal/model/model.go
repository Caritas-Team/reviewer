package model

import (
	"time"
)

// AssessmentDocument основной документ обследования
type AssessmentDocument struct {
	Metadata           AssessmentMetadata     `json:"metadata"`
	LanguageLevels     LanguageDevelopment    `json:"language_levels"`
	CommunicativeFuncs CommunicativeFunctions `json:"communicative_functions"`
	Vocabulary         VocabularyData         `json:"vocabulary"`
}

// AssessmentMetadata метаданные оценки
type AssessmentMetadata struct {
	StudentID      string    `json:"student_id"`
	Date           time.Time `json:"date"`
	AssessmentType string    `json:"assessment_type"` // "before" или "after"
	FileName       string    `json:"file_name,omitempty"`
}

// LanguageDevelopment уровни языкового развития
type LanguageDevelopment struct {
	Preintentional Preintentional   `json:"preintentional"` //доинтенциональная коммуникация
	Protolanguage  LanguageActivity `json:"protolanguage"`  //протоязык
	Holophrase     LanguageActivity `json:"holophrase"`     //голофраза
	Phrase         LanguageActivity `json:"phrase"`         //фраза
}

type Preintentional struct {
	Activity float64 `json:"activity"`
}

type LanguageActivity struct {
	Activity   float64 `json:"activity"`
	Initiative float64 `json:"initiative"` //инициатива
}

// CommunicativeFunctions коммуникативные функции
type CommunicativeFunctions struct {
	Control             float64 `json:"control"`              //контроль
	ObtainingDesired    float64 `json:"obtaining_desired"`    //получение желаемого
	SocialInteraction   float64 `json:"social_interaction"`   //cоциальное взаимодействие
	InformationExchange float64 `json:"information_exchange"` //обмен информацией
}

// VocabularyData словарный запас
type VocabularyData struct {
	ActiveWordsCount int      `json:"active_words_count"`
	AdditionalWords  []string `json:"additional_words"`
	TotalWordsCount  int      `json:"total_words_count"`
}

// StudentPair пара документов для одного студента (до/после)
type StudentPair struct {
	RequestID string              `json:"request_id"`
	StudentID string              `json:"student_id"`
	Before    *AssessmentDocument `json:"before"`
	After     *AssessmentDocument `json:"after"`
}

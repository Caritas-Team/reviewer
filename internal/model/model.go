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
	ActBlock           ActBlockData           `json:"act_block,omitempty"`
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
	ActiveWordsCount  int      `json:"active_words_count"`
	AdditionalWords   []string `json:"additional_words"`
	TotalWordsCount   int      `json:"total_words_count"`
	CommunicationWays []string `json:"communication_ways,omitempty"`
	VerbalWordsCount  int      `json:"verbal_words_count,omitempty"`
}

// StudentPair пара документов для одного студента (до/после)
type StudentPair struct {
	RequestID string              `json:"request_id"`
	StudentID string              `json:"student_id"`
	Before    *AssessmentDocument `json:"before"`
	After     *AssessmentDocument `json:"after"`
}

// ProcessingResult - результат обработки пары документов
type ProcessingResult struct {
	RequestID         string                 `json:"request_id"`              // ID запроса
	Status            string                 `json:"status"`                  // Текущий статус обработки (processing, completed, failed)
	ErrorMessage      string                 `json:"error_message,omitempty"` // Сообщение об ошибке, если возникли проблемы
	ProcessedStudents int                    `json:"processed_students"`      // Количество обработанных учеников
	TotalStudents     int                    `json:"total_students"`          // Общее число учеников
	ResultDetails     map[string]interface{} `json:"result_details"`          // Подробности расчёта
	Errors            map[string]string      `json:"errors,omitempty"`        // Карта ошибок (ключ - StudentID, значение - сообщение об ошибке)
	CreatedAt         time.Time              `json:"created_at"`              // Время создания результата
}

// ProcessingError - структура ошибки обработки
type ProcessingError struct {
	RequestID string `json:"request_id"` // ID запроса
	StudentID string `json:"student_id"` // ID студента
	Message   string `json:"message"`    // Сообщение об ошибке
}

// ActivityData данные по активности уровня
type ActivityData struct {
	ActivityPercent   float64 `json:"activity_percent,omitempty"`
	InitiativePercent float64 `json:"initiative_percent,omitempty"`
	FrequencyPercent  float64 `json:"frequency_percent,omitempty"`
}

// ActBlockData структура для хранения данных из actBlock01
type ActBlockData struct {
	Prot ActivityData `json:"prot,omitempty"`
	Gol  ActivityData `json:"gol,omitempty"`
	Fra  ActivityData `json:"fra,omitempty"`
}

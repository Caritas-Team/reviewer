package file

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Caritas-Team/reviewer/internal/model"
)

type DocumentParser struct{}

// Parse парсит JSON и создает структурированный AssessmentDocument
func (p *DocumentParser) Parse(r io.Reader, filename string) (*model.AssessmentDocument, error) {

	var fullRawData FullRawJSON
	if err := json.NewDecoder(r).Decode(&fullRawData); err != nil {
		return nil, fmt.Errorf("не удалось разобрать структуру JSON: %w", err)
	}

	studentID := extractFromDivTag(fullRawData.Por02)
	if studentID == "" {
		return nil, fmt.Errorf("идентификатор студента не найден в поле por02")
	}

	dateStr := extractFromDivTag(fullRawData.Por01)
	if dateStr == "" {
		return nil, fmt.Errorf("дата не найдена в поле por01")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("неверный формат даты в por01: %w", err)
	}

	// Создаем структурированный документ
	doc := &model.AssessmentDocument{
		Metadata: model.AssessmentMetadata{
			StudentID:      studentID,
			Date:           date,
			AssessmentType: "unknown", // Будет установлено позже
			FileName:       filename,
		},
	}

	// Парсим уровни языкового развития
	if err := p.parseLanguageLevels(&fullRawData, &doc.LanguageLevels); err != nil {
		return nil, fmt.Errorf("не удалось разобрать уровни языка: %w", err)
	}

	// Парсим коммуникативные функции
	if err := p.parseCommunicativeFunctions(&fullRawData, &doc.CommunicativeFuncs); err != nil {
		return nil, fmt.Errorf("не удалось разобрать коммуникативные функции: %w", err)
	}

	// Парсим словарный запас
	if err := p.parseVocabulary(&fullRawData, &doc.Vocabulary); err != nil {
		return nil, fmt.Errorf("не удалось разобрать словарный запас: %w", err)
	}
	// Парсим actBlock01 данные
	p.parseActBlockData(&fullRawData, doc)

	return doc, nil
}

type ProcNumElem struct {
	ProcNumElem string `json:"procNumElem"`
}

type DiagramBlock struct {
	PredActProcNumElem  string `json:"predActProcNumElem"`
	ProtActProcNumElem  string `json:"protActProcNumElem"`
	ProtInitProcNumElem string `json:"protInitProcNumElem"`
	GolActProcNumElem   string `json:"golActProcNumElem"`
	GolInitProcNumElem  string `json:"golInitProcNumElem"`
	FraActProcNumElem   string `json:"fraActProcNumElem"`
	FraInitProcNumElem  string `json:"fraInitProcNumElem"`
}

type ActBlock01 struct {
	ProtSforProcElem  string `json:"protSforProcElem"`
	ProtInitProcElem  string `json:"protInitProcElem"`
	ProtChastProcElem string `json:"protChastProcElem"`
	GolSforProcElem   string `json:"golSforProcElem"`
	GolInitProcElem   string `json:"golInitProcElem"`
	GolChastProcElem  string `json:"golChastProcElem"`
	FraSforProcElem   string `json:"fraSforProcElem"`
	FraInitProcElem   string `json:"fraInitProcElem"`
	FraChastProcElem  string `json:"fraChastProcElem"`
}

type FullRawJSON struct {
	Por01    string      `json:"por01"` // Дата в формате <div>2025-11-11</div>
	Por02    string      `json:"por02"` // Student ID в формате <div>123</div>
	NewAct01 ProcNumElem `json:"newAct01"`
	NewAct02 ProcNumElem `json:"newAct02"`
	NewAct03 ProcNumElem `json:"newAct03"`
	NewAct04 ProcNumElem `json:"newAct04"`

	DiagramBlock DiagramBlock `json:"diagramBlock"`

	BasicDictionary []BasicDictionaryItem `json:"basicDictionary"`
	DictBasicMore   []string              `json:"dictBasicMore"`
	DictSposObsh    []string              `json:"dictSposObsh"`
	DictWerbSlov    []string              `json:"dictWerbSlov"`

	ActBlock01 ActBlock01 `json:"actBlock01"`
}

type BasicDictionaryItem struct {
	ItemOffStyle string `json:"itemOffStyle"`
}

// parseLanguageLevels парсит уровни языкового развития
func (p *DocumentParser) parseLanguageLevels(raw *FullRawJSON, levels *model.LanguageDevelopment) error {
	parsePercent := func(s string) (float64, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0.0, nil
		}
		s = strings.TrimSuffix(s, "%")
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0.0, fmt.Errorf("неверный формат процента '%s': %w", s, err)
		}
		return val, nil
	}

	var err error

	if levels.Preintentional.Activity, err = parsePercent(raw.DiagramBlock.PredActProcNumElem); err != nil {
		return fmt.Errorf("доинтенциональная активность: %w", err)
	}

	if levels.Protolanguage.Activity, err = parsePercent(raw.DiagramBlock.ProtActProcNumElem); err != nil {
		return fmt.Errorf("активность протоязыка: %w", err)
	}

	if levels.Protolanguage.Initiative, err = parsePercent(raw.DiagramBlock.ProtInitProcNumElem); err != nil {
		return fmt.Errorf("инициатива протоязыка: %w", err)
	}

	if levels.Holophrase.Activity, err = parsePercent(raw.DiagramBlock.GolActProcNumElem); err != nil {
		return fmt.Errorf("активность голофразы: %w", err)
	}

	if levels.Holophrase.Initiative, err = parsePercent(raw.DiagramBlock.GolInitProcNumElem); err != nil {
		return fmt.Errorf("инициатива голофразы: %w", err)
	}

	if levels.Phrase.Activity, err = parsePercent(raw.DiagramBlock.FraActProcNumElem); err != nil {
		return fmt.Errorf("активность фразы: %w", err)
	}

	if levels.Phrase.Initiative, err = parsePercent(raw.DiagramBlock.FraInitProcNumElem); err != nil {
		return fmt.Errorf("инициатива фразы: %w", err)
	}

	return nil
}

// parseCommunicativeFunctions парсит коммуникативные функции
func (p *DocumentParser) parseCommunicativeFunctions(raw *FullRawJSON, funcs *model.CommunicativeFunctions) error {
	parsePercent := func(s string) (float64, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0.0, nil
		}
		s = strings.TrimSuffix(s, "%")
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0.0, fmt.Errorf("неверный формат процента '%s': %w", s, err)
		}
		return val, nil
	}

	var err error

	if funcs.Control, err = parsePercent(raw.NewAct01.ProcNumElem); err != nil {
		return fmt.Errorf("control: %w", err)
	}

	if funcs.ObtainingDesired, err = parsePercent(raw.NewAct02.ProcNumElem); err != nil {
		return fmt.Errorf("получение желаемого: %w", err)
	}

	if funcs.SocialInteraction, err = parsePercent(raw.NewAct03.ProcNumElem); err != nil {
		return fmt.Errorf("социальное взаимодействие: %w", err)
	}

	if funcs.InformationExchange, err = parsePercent(raw.NewAct04.ProcNumElem); err != nil {
		return fmt.Errorf("обмен информацией: %w", err)
	}

	return nil
}

// parseVocabulary парсит словарный запас
func (p *DocumentParser) parseVocabulary(raw *FullRawJSON, vocab *model.VocabularyData) error {
	// Подсчитываем активные слова (itemOffStyle == "")
	activeWordsCount := 0
	for _, item := range raw.BasicDictionary {
		if item.ItemOffStyle == "" {
			activeWordsCount++
		}
	}

	// Подсчет дополнительных слов из dictBasicMore
	additionalWords := make([]string, 0)
	for _, word := range raw.DictBasicMore {
		trimmedWord := strings.TrimSpace(word)
		if trimmedWord != "" {
			additionalWords = append(additionalWords, trimmedWord)
		}
	}

	// Получаем способы общения dictSposObsh
	communicationWays := make([]string, 0)
	for _, way := range raw.DictSposObsh {
		trimmedWay := strings.TrimSpace(way)
		if trimmedWay != "" {
			communicationWays = append(communicationWays, trimmedWay)
		}
	}

	// Считаем количество вербальных слов
	verbalWordsCount := 0
	for _, word := range raw.DictWerbSlov {
		trimmedWord := strings.TrimSpace(word)
		if trimmedWord != "" {
			verbalWordsCount++
		}
	}

	vocab.ActiveWordsCount = activeWordsCount
	vocab.AdditionalWords = additionalWords
	vocab.TotalWordsCount = activeWordsCount + len(additionalWords) + verbalWordsCount
	vocab.CommunicationWays = communicationWays
	vocab.VerbalWordsCount = verbalWordsCount

	return nil
}

// extractFromDivTag извлекает содержимое из HTML тега <div>
func extractFromDivTag(divContent string) string {
	if divContent == "" {
		return ""
	}

	content := strings.TrimPrefix(divContent, "<div>")
	content = strings.TrimSuffix(content, "</div>")
	content = strings.TrimSpace(content)

	return content
}

// parseActBlockData парсит данные из actBlock01 полей
func (p *DocumentParser) parseActBlockData(raw *FullRawJSON, doc *model.AssessmentDocument) {
	// Функция для парсинга ProcNumElem в float64
	parsePercentStr := func(el string) float64 {
		if el == "" {
			return 0
		}

		s := strings.TrimSpace(el)
		s = strings.TrimSuffix(s, "%")
		if s == "" {
			return 0
		}

		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}

		return val
	}

	// Парсим все поля
	doc.ActBlock.Prot.ActivityPercent = parsePercentStr(raw.ActBlock01.ProtSforProcElem)
	doc.ActBlock.Prot.InitiativePercent = parsePercentStr(raw.ActBlock01.ProtInitProcElem)
	doc.ActBlock.Prot.FrequencyPercent = parsePercentStr(raw.ActBlock01.ProtChastProcElem)

	doc.ActBlock.Gol.ActivityPercent = parsePercentStr(raw.ActBlock01.GolSforProcElem)
	doc.ActBlock.Gol.InitiativePercent = parsePercentStr(raw.ActBlock01.GolInitProcElem)
	doc.ActBlock.Gol.FrequencyPercent = parsePercentStr(raw.ActBlock01.GolChastProcElem)

	doc.ActBlock.Fra.ActivityPercent = parsePercentStr(raw.ActBlock01.FraSforProcElem)
	doc.ActBlock.Fra.InitiativePercent = parsePercentStr(raw.ActBlock01.FraInitProcElem)
	doc.ActBlock.Fra.FrequencyPercent = parsePercentStr(raw.ActBlock01.FraChastProcElem)
}

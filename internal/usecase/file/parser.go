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
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать данные: %w", err)
	}

	var fullRawData FullRawJSON
	if err := json.Unmarshal(data, &fullRawData); err != nil {
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

	diagnosis := extractFromDivTag(fullRawData.Por04)
	livingSituation := extractFromDivTag(fullRawData.Por05)
	familyDescription := extractFromDivTag(fullRawData.Por06)

	// Создаем структурированный документ
	doc := &model.AssessmentDocument{
		Metadata: model.AssessmentMetadata{
			StudentID:      studentID,
			Date:           date,
			AssessmentType: "unknown", // Будет установлено позже
			FileName:       filename,
		},
		Diagnosis:         diagnosis,
		LivingSituation:   livingSituation,
		FamilyDescription: familyDescription,
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
	if err := p.parseVocabulary(&fullRawData, &doc.Vocabulary, doc); err != nil {
		return nil, fmt.Errorf("не удалось разобрать словарный запас: %w", err)
	}
	// Парсим actBlock01 данные
	p.parseActBlockData(&fullRawData, doc)

	otherBlocks := []struct {
		block ActBlockOther
		id    string
	}{
		{fullRawData.ActBlock02, "actBlock02"},
		{fullRawData.ActBlock03, "actBlock03"},
		{fullRawData.ActBlock04, "actBlock04"},
		{fullRawData.ActBlock05, "actBlock05"},
		{fullRawData.ActBlock08, "actBlock08"},
		{fullRawData.ActBlock09, "actBlock09"},
		{fullRawData.ActBlock10, "actBlock10"},
		{fullRawData.ActBlock11, "actBlock11"},
		{fullRawData.ActBlock14, "actBlock14"},
		{fullRawData.ActBlock15, "actBlock15"},
		{fullRawData.ActBlock16, "actBlock16"},
		{fullRawData.ActBlock18, "actBlock18"},
	}
	doc.OtherActBlocks = make(map[string]model.ActBlockOtherRaw)
	doc.OtherActBlocks["actBlock01"] = convertActBlock01(fullRawData.ActBlock01)
	for _, b := range otherBlocks {
		filtered := convertActBlockOther(b.block)
		doc.OtherActBlocks[b.id] = filtered
	}

	doc.FastMessages = fullRawData.DictBystrSoobsh

	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return nil, fmt.Errorf("не удалось разобрать структуру JSON: %w", err)
	} else {
		counts := map[string]int{
			"first":  0,
			"second": 0,
			"third":  0,
			"fourth": 0,
		}
		for key := range rawMap {
			switch {
			case strings.HasPrefix(key, "roundTargetFirst"):
				counts["first"]++
			case strings.HasPrefix(key, "roundTargetSecond"):
				counts["second"]++
			case strings.HasPrefix(key, "roundTargetThird"):
				counts["third"]++
			case strings.HasPrefix(key, "roundTargetFourth"):
				counts["fourth"]++
			}
		}

		if counts["first"] > 0 || counts["second"] > 0 || counts["third"] > 0 || counts["fourth"] > 0 {
			doc.CommunicationCounts = counts
		}
	}

	doc.DiagramRaw = model.DiagramRaw{
		PredActProcNumElem:  fullRawData.DiagramBlock.PredActProcNumElem,
		PredInitProcNumElem: fullRawData.DiagramBlock.PredInitProcNumElem,
		ProtActProcNumElem:  fullRawData.DiagramBlock.ProtActProcNumElem,
		ProtInitProcNumElem: fullRawData.DiagramBlock.ProtInitProcNumElem,
		GolActProcNumElem:   fullRawData.DiagramBlock.GolActProcNumElem,
		GolInitProcNumElem:  fullRawData.DiagramBlock.GolInitProcNumElem,
		FraActProcNumElem:   fullRawData.DiagramBlock.FraActProcNumElem,
		FraInitProcNumElem:  fullRawData.DiagramBlock.FraInitProcNumElem,
	}

	doc.NewAct01Raw = fullRawData.NewAct01.ProcNumElem
	doc.NewAct02Raw = fullRawData.NewAct02.ProcNumElem
	doc.NewAct03Raw = fullRawData.NewAct03.ProcNumElem
	doc.NewAct04Raw = fullRawData.NewAct04.ProcNumElem

	birthDate := extractFromDivTag(fullRawData.Por03)
	doc.BirthDate = birthDate

	return doc, nil
}

type ProcNumElem struct {
	ProcNumElem string `json:"procNumElem"`
}

type DiagramBlock struct {
	PredActProcNumElem  string `json:"predActProcNumElem"`
	PredInitProcNumElem string `json:"predInitProcNumElem"`
	ProtActProcNumElem  string `json:"protActProcNumElem"`
	ProtInitProcNumElem string `json:"protInitProcNumElem"`
	GolActProcNumElem   string `json:"golActProcNumElem"`
	GolInitProcNumElem  string `json:"golInitProcNumElem"`
	FraActProcNumElem   string `json:"fraActProcNumElem"`
	FraInitProcNumElem  string `json:"fraInitProcNumElem"`
}

type ActBlock01 struct {
	ProtSforProcElem   string `json:"protSforProcElem"`
	ProtInitProcElem   string `json:"protInitProcElem"`
	ProtChastProcElem  string `json:"protChastProcElem"`
	GolSforProcElem    string `json:"golSforProcElem"`
	GolInitProcElem    string `json:"golInitProcElem"`
	GolChastProcElem   string `json:"golChastProcElem"`
	FraSforProcElem    string `json:"fraSforProcElem"`
	FraInitProcElem    string `json:"fraInitProcElem"`
	FraChastProcElem   string `json:"fraChastProcElem"`
	BodyBlockElem      string `json:"bodyBlockElem"`
	UnavailableTagElem string `json:"unavailableTagElem"`
	ProtUnElem         string `json:"protUnElem"`
	ProtOverElem       string `json:"protOverElem"`
	GolUnElem          string `json:"golUnElem"`
	GolOverElem        string `json:"golOverElem"`
	FraUnElem          string `json:"fraUnElem"`
}

type ActBlockOther struct {
	BodyBlockElem      string `json:"bodyBlockElem"`
	UnavailableTagElem string `json:"unavailableTagElem"`
	ProtUnElem         string `json:"protUnElem"`
	ProtOverElem       string `json:"protOverElem"`
	ProtSforProcElem   string `json:"protSforProcElem"`
	ProtInitProcElem   string `json:"protInitProcElem"`
	GolUnElem          string `json:"golUnElem"`
	GolOverElem        string `json:"golOverElem"`
	GolSforProcElem    string `json:"golSforProcElem"`
	GolInitProcElem    string `json:"golInitProcElem"`
	FraUnElem          string `json:"fraUnElem"`
	FraSforProcElem    string `json:"fraSforProcElem"`
	FraInitProcElem    string `json:"fraInitProcElem"`
}

type FullRawJSON struct {
	Por01    string      `json:"por01"` // Дата в формате <div>2025-11-11</div>
	Por02    string      `json:"por02"` // Student ID в формате <div>123</div>
	Por03    string      `json:"por03"`
	Por04    string      `json:"por04"`
	Por05    string      `json:"por05"`
	Por06    string      `json:"por06"`
	NewAct01 ProcNumElem `json:"newAct01"`
	NewAct02 ProcNumElem `json:"newAct02"`
	NewAct03 ProcNumElem `json:"newAct03"`
	NewAct04 ProcNumElem `json:"newAct04"`

	DiagramBlock DiagramBlock `json:"diagramBlock"`

	BasicDictionary []BasicDictionaryItem `json:"basicDictionary"`
	DictBasicMore   []string              `json:"dictBasicMore"`
	DictSposObsh    []string              `json:"dictSposObsh"`
	DictWerbSlov    []string              `json:"dictWerbSlov"`

	ActBlock01 ActBlock01    `json:"actBlock01"`
	ActBlock02 ActBlockOther `json:"actBlock02,omitempty"`
	ActBlock03 ActBlockOther `json:"actBlock03,omitempty"`
	ActBlock04 ActBlockOther `json:"actBlock04,omitempty"`
	ActBlock05 ActBlockOther `json:"actBlock05,omitempty"`
	ActBlock08 ActBlockOther `json:"actBlock08,omitempty"`
	ActBlock09 ActBlockOther `json:"actBlock09,omitempty"`
	ActBlock10 ActBlockOther `json:"actBlock10,omitempty"`
	ActBlock11 ActBlockOther `json:"actBlock11,omitempty"`
	ActBlock14 ActBlockOther `json:"actBlock14,omitempty"`
	ActBlock15 ActBlockOther `json:"actBlock15,omitempty"`
	ActBlock16 ActBlockOther `json:"actBlock16,omitempty"`
	ActBlock18 ActBlockOther `json:"actBlock18,omitempty"`

	DictBystrSoobsh []string `json:"dictBystrSoobsh"`
}

type BasicDictionaryItem struct {
	ItemOffStyle string `json:"itemOffStyle"`
	Content      string `json:"content"`
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

	if levels.Preintentional.Initiative, err = parsePercent(raw.DiagramBlock.PredInitProcNumElem); err != nil {
		return fmt.Errorf("доинтенциональная инициатива: %w", err)
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
func (p *DocumentParser) parseVocabulary(raw *FullRawJSON, vocab *model.VocabularyData, doc *model.AssessmentDocument) error {
	// Подсчитываем активные слова (itemOffStyle == "")
	activeWordsCount := 0
	activeWords := []string{}
	for _, item := range raw.BasicDictionary {
		if item.ItemOffStyle == "" {
			activeWordsCount++
			activeWords = append(activeWords, item.Content)
		}
	}
	doc.ActiveWords = activeWords

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

	verbalWords := []string{}
	for _, word := range raw.DictWerbSlov {
		trimmedWord := strings.TrimSpace(word)
		if trimmedWord != "" {
			verbalWords = append(verbalWords, trimmedWord)
		}
	}
	doc.VerbalWords = verbalWords
	doc.AdditionalWords = additionalWords

	doc.FastMessages = raw.DictBystrSoobsh

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

func convertActBlockOther(block ActBlockOther) model.ActBlockOtherRaw {
	return model.ActBlockOtherRaw{
		BodyBlockElem:      block.BodyBlockElem,
		UnavailableTagElem: block.UnavailableTagElem,
		ProtUnElem:         block.ProtUnElem,
		ProtOverElem:       block.ProtOverElem,
		ProtSforProcElem:   block.ProtSforProcElem,
		ProtInitProcElem:   block.ProtInitProcElem,
		GolUnElem:          block.GolUnElem,
		GolOverElem:        block.GolOverElem,
		GolSforProcElem:    block.GolSforProcElem,
		GolInitProcElem:    block.GolInitProcElem,
		FraUnElem:          block.FraUnElem,
		FraSforProcElem:    block.FraSforProcElem,
		FraInitProcElem:    block.FraInitProcElem,
	}
}

func convertActBlock01(block ActBlock01) model.ActBlockOtherRaw {
	return model.ActBlockOtherRaw{
		BodyBlockElem:      block.BodyBlockElem,
		UnavailableTagElem: block.UnavailableTagElem,
		ProtUnElem:         block.ProtUnElem,
		ProtOverElem:       block.ProtOverElem,
		ProtSforProcElem:   block.ProtSforProcElem,
		ProtInitProcElem:   block.ProtInitProcElem,
		GolUnElem:          block.GolUnElem,
		GolOverElem:        block.GolOverElem,
		GolSforProcElem:    block.GolSforProcElem,
		GolInitProcElem:    block.GolInitProcElem,
		FraUnElem:          block.FraUnElem,
		FraSforProcElem:    block.FraSforProcElem,
		FraInitProcElem:    block.FraInitProcElem,
	}
}

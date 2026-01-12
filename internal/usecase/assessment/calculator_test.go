package assessment

import (
	"testing"
	"time"

	"github.com/Caritas-Team/reviewer/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffCalculator_CalculateGroupAverage(t *testing.T) {
	calc := &DiffCalculator{}

	// Helper для создания документа
	createDoc := func(studentID, dateStr string, langDev model.LanguageDevelopment, commFuncs model.CommunicativeFunctions, vocab model.VocabularyData) *model.AssessmentDocument {
		date, _ := time.Parse("2006-01-02", dateStr)
		return &model.AssessmentDocument{
			Metadata: model.AssessmentMetadata{
				StudentID: studentID,
				Date:      date,
			},
			LanguageLevels:     langDev,
			CommunicativeFuncs: commFuncs,
			Vocabulary:         vocab,
		}
	}

	t.Run("empty documents list returns error", func(t *testing.T) {
		avg, err := calc.CalculateGroupAverage([]*model.AssessmentDocument{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty documents")
		assert.Empty(t, avg.Date)
	})

	t.Run("single document calculates correctly", func(t *testing.T) {
		doc := createDoc(
			"student1",
			"2025-12-12",
			model.LanguageDevelopment{
				Preintentional: model.Preintentional{Activity: 50.0},
				Protolanguage:  model.LanguageActivity{Activity: 30.0, Initiative: 20.0},
				Holophrase:     model.LanguageActivity{Activity: 40.0, Initiative: 35.0},
				Phrase:         model.LanguageActivity{Activity: 60.0, Initiative: 55.0},
			},
			model.CommunicativeFunctions{
				Control:             10.0,
				ObtainingDesired:    15.0,
				SocialInteraction:   20.0,
				InformationExchange: 25.0,
			},
			model.VocabularyData{
				ActiveWordsCount: 100,
				TotalWordsCount:  150,
			},
		)

		avg, err := calc.CalculateGroupAverage([]*model.AssessmentDocument{doc})
		require.NoError(t, err)
		assert.Equal(t, "2025-12-12", avg.Date)
		assert.Equal(t, 1, avg.StudentsCount)
		assert.Equal(t, 50.0, avg.LanguageLevels.Preintentional.Activity)
		assert.Equal(t, 30.0, avg.LanguageLevels.Protolanguage.Activity)
		assert.Equal(t, 20.0, avg.LanguageLevels.Protolanguage.Initiative)
		assert.Equal(t, 10.0, avg.CommunicativeFuncs.Control)
		assert.Equal(t, 100, avg.Vocabulary.ActiveWordsCount)
		assert.Equal(t, 150, avg.Vocabulary.TotalWordsCount)
	})

	t.Run("three students on December 12th", func(t *testing.T) {
		docsDec12 := []*model.AssessmentDocument{
			// Student A
			createDoc(
				"A",
				"2025-12-12",
				model.LanguageDevelopment{
					Preintentional: model.Preintentional{Activity: 50.0},
					Protolanguage:  model.LanguageActivity{Activity: 30.0, Initiative: 20.0},
					Holophrase:     model.LanguageActivity{Activity: 40.0, Initiative: 35.0},
					Phrase:         model.LanguageActivity{Activity: 60.0, Initiative: 55.0},
				},
				model.CommunicativeFunctions{
					Control:             10.0,
					ObtainingDesired:    15.0,
					SocialInteraction:   20.0,
					InformationExchange: 25.0,
				},
				model.VocabularyData{
					ActiveWordsCount: 100,
					TotalWordsCount:  150,
				},
			),
			// Student B
			createDoc(
				"B",
				"2025-12-12",
				model.LanguageDevelopment{
					Preintentional: model.Preintentional{Activity: 60.0},
					Protolanguage:  model.LanguageActivity{Activity: 40.0, Initiative: 30.0},
					Holophrase:     model.LanguageActivity{Activity: 50.0, Initiative: 45.0},
					Phrase:         model.LanguageActivity{Activity: 70.0, Initiative: 65.0},
				},
				model.CommunicativeFunctions{
					Control:             20.0,
					ObtainingDesired:    25.0,
					SocialInteraction:   30.0,
					InformationExchange: 35.0,
				},
				model.VocabularyData{
					ActiveWordsCount: 120,
					TotalWordsCount:  180,
				},
			),
			// Student C
			createDoc(
				"C",
				"2025-12-12",
				model.LanguageDevelopment{
					Preintentional: model.Preintentional{Activity: 40.0},
					Protolanguage:  model.LanguageActivity{Activity: 20.0, Initiative: 10.0},
					Holophrase:     model.LanguageActivity{Activity: 30.0, Initiative: 25.0},
					Phrase:         model.LanguageActivity{Activity: 50.0, Initiative: 45.0},
				},
				model.CommunicativeFunctions{
					Control:             5.0,
					ObtainingDesired:    10.0,
					SocialInteraction:   15.0,
					InformationExchange: 20.0,
				},
				model.VocabularyData{
					ActiveWordsCount: 80,
					TotalWordsCount:  120,
				},
			),
		}

		avg, err := calc.CalculateGroupAverage(docsDec12)
		require.NoError(t, err)
		assert.Equal(t, "2025-12-12", avg.Date)
		assert.Equal(t, 3, avg.StudentsCount)

		// Средние значения: (50+60+40)/3 = 50.0
		assert.InDelta(t, 50.0, avg.LanguageLevels.Preintentional.Activity, 0.01)
		// (30+40+20)/3 = 30.0
		assert.InDelta(t, 30.0, avg.LanguageLevels.Protolanguage.Activity, 0.01)
		// (20+30+10)/3 = 20.0
		assert.InDelta(t, 20.0, avg.LanguageLevels.Protolanguage.Initiative, 0.01)
		// (40+50+30)/3 = 40.0
		assert.InDelta(t, 40.0, avg.LanguageLevels.Holophrase.Activity, 0.01)
		// (35+45+25)/3 = 35.0
		assert.InDelta(t, 35.0, avg.LanguageLevels.Holophrase.Initiative, 0.01)
		// (60+70+50)/3 = 60.0
		assert.InDelta(t, 60.0, avg.LanguageLevels.Phrase.Activity, 0.01)
		// (55+65+45)/3 = 55.0
		assert.InDelta(t, 55.0, avg.LanguageLevels.Phrase.Initiative, 0.01)

		// Коммуникативные функции: (10+20+5)/3 = 11.67
		assert.InDelta(t, 11.67, avg.CommunicativeFuncs.Control, 0.01)
		// (15+25+10)/3 = 16.67
		assert.InDelta(t, 16.67, avg.CommunicativeFuncs.ObtainingDesired, 0.01)
		// (20+30+15)/3 = 21.67
		assert.InDelta(t, 21.67, avg.CommunicativeFuncs.SocialInteraction, 0.01)
		// (25+35+20)/3 = 26.67
		assert.InDelta(t, 26.67, avg.CommunicativeFuncs.InformationExchange, 0.01)

		// Словарь: (100+120+80)/3 = 100
		assert.Equal(t, 100, avg.Vocabulary.ActiveWordsCount)
		// (150+180+120)/3 = 150
		assert.Equal(t, 150, avg.Vocabulary.TotalWordsCount)
		assert.Nil(t, avg.Vocabulary.AdditionalWords)
	})

	t.Run("three students on December 15th", func(t *testing.T) {
		docsDec15 := []*model.AssessmentDocument{
			// Student A
			createDoc(
				"A",
				"2025-12-15",
				model.LanguageDevelopment{
					Preintentional: model.Preintentional{Activity: 55.0},
					Protolanguage:  model.LanguageActivity{Activity: 35.0, Initiative: 25.0},
					Holophrase:     model.LanguageActivity{Activity: 45.0, Initiative: 40.0},
					Phrase:         model.LanguageActivity{Activity: 65.0, Initiative: 60.0},
				},
				model.CommunicativeFunctions{
					Control:             15.0,
					ObtainingDesired:    20.0,
					SocialInteraction:   25.0,
					InformationExchange: 30.0,
				},
				model.VocabularyData{
					ActiveWordsCount: 110,
					TotalWordsCount:  160,
				},
			),
			// Student B
			createDoc(
				"B",
				"2025-12-15",
				model.LanguageDevelopment{
					Preintentional: model.Preintentional{Activity: 65.0},
					Protolanguage:  model.LanguageActivity{Activity: 45.0, Initiative: 35.0},
					Holophrase:     model.LanguageActivity{Activity: 55.0, Initiative: 50.0},
					Phrase:         model.LanguageActivity{Activity: 75.0, Initiative: 70.0},
				},
				model.CommunicativeFunctions{
					Control:             25.0,
					ObtainingDesired:    30.0,
					SocialInteraction:   35.0,
					InformationExchange: 40.0,
				},
				model.VocabularyData{
					ActiveWordsCount: 130,
					TotalWordsCount:  190,
				},
			),
			// Student C
			createDoc(
				"C",
				"2025-12-15",
				model.LanguageDevelopment{
					Preintentional: model.Preintentional{Activity: 45.0},
					Protolanguage:  model.LanguageActivity{Activity: 25.0, Initiative: 15.0},
					Holophrase:     model.LanguageActivity{Activity: 35.0, Initiative: 30.0},
					Phrase:         model.LanguageActivity{Activity: 55.0, Initiative: 50.0},
				},
				model.CommunicativeFunctions{
					Control:             10.0,
					ObtainingDesired:    15.0,
					SocialInteraction:   20.0,
					InformationExchange: 25.0,
				},
				model.VocabularyData{
					ActiveWordsCount: 90,
					TotalWordsCount:  130,
				},
			),
		}

		avg, err := calc.CalculateGroupAverage(docsDec15)
		require.NoError(t, err)
		assert.Equal(t, "2025-12-15", avg.Date)
		assert.Equal(t, 3, avg.StudentsCount)

		// Средние значения: (55+65+45)/3 = 55.0
		assert.InDelta(t, 55.0, avg.LanguageLevels.Preintentional.Activity, 0.01)
		// (35+45+25)/3 = 35.0
		assert.InDelta(t, 35.0, avg.LanguageLevels.Protolanguage.Activity, 0.01)
		// (25+35+15)/3 = 25.0
		assert.InDelta(t, 25.0, avg.LanguageLevels.Protolanguage.Initiative, 0.01)
		// (45+55+35)/3 = 45.0
		assert.InDelta(t, 45.0, avg.LanguageLevels.Holophrase.Activity, 0.01)
		// (40+50+30)/3 = 40.0
		assert.InDelta(t, 40.0, avg.LanguageLevels.Holophrase.Initiative, 0.01)
		// (65+75+55)/3 = 65.0
		assert.InDelta(t, 65.0, avg.LanguageLevels.Phrase.Activity, 0.01)
		// (60+70+50)/3 = 60.0
		assert.InDelta(t, 60.0, avg.LanguageLevels.Phrase.Initiative, 0.01)

		// Коммуникативные функции: (15+25+10)/3 = 16.67
		assert.InDelta(t, 16.67, avg.CommunicativeFuncs.Control, 0.01)
		// (20+30+15)/3 = 21.67
		assert.InDelta(t, 21.67, avg.CommunicativeFuncs.ObtainingDesired, 0.01)
		// (25+35+20)/3 = 26.67
		assert.InDelta(t, 26.67, avg.CommunicativeFuncs.SocialInteraction, 0.01)
		// (30+40+25)/3 = 31.67
		assert.InDelta(t, 31.67, avg.CommunicativeFuncs.InformationExchange, 0.01)

		// Словарь: (110+130+90)/3 = 110
		assert.Equal(t, 110, avg.Vocabulary.ActiveWordsCount)
		// (160+190+130)/3 = 160
		assert.Equal(t, 160, avg.Vocabulary.TotalWordsCount)
		assert.Nil(t, avg.Vocabulary.AdditionalWords)
	})

	t.Run("documents with different dates returns error", func(t *testing.T) {
		docs := []*model.AssessmentDocument{
			createDoc(
				"A",
				"2025-12-12",
				model.LanguageDevelopment{},
				model.CommunicativeFunctions{},
				model.VocabularyData{},
			),
			createDoc(
				"B",
				"2025-12-15",
				model.LanguageDevelopment{},
				model.CommunicativeFunctions{},
				model.VocabularyData{},
			),
		}

		avg, err := calc.CalculateGroupAverage(docs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "different dates")
		assert.Empty(t, avg.Date)
	})
}

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
	createDoc := func(studentID, dateStr string, langDev model.LanguageDevelopment, commFuncs model.CommunicativeFunctions, vocab model.VocabularyData, actBlock model.ActBlockData) *model.AssessmentDocument {
		date, _ := time.Parse("2006-01-02", dateStr)
		return &model.AssessmentDocument{
			Metadata: model.AssessmentMetadata{
				StudentID: studentID,
				Date:      date,
			},
			LanguageLevels:     langDev,
			CommunicativeFuncs: commFuncs,
			Vocabulary:         vocab,
			ActBlock:           actBlock,
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
			model.ActBlockData{
				Prot: model.ActivityData{
					ActivityPercent:   90.0,
					InitiativePercent: 80.0,
					FrequencyPercent:  90.0,
				},
				Gol: model.ActivityData{
					ActivityPercent:   7.0,
					InitiativePercent: 33.0,
					FrequencyPercent:  90.0,
				},
				Fra: model.ActivityData{
					ActivityPercent:   0,
					InitiativePercent: 0,
					FrequencyPercent:  0,
				},
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

		assert.Equal(t, 90.0, avg.ActBlock.Prot.ActivityPercent)
		assert.Equal(t, 80.0, avg.ActBlock.Prot.InitiativePercent)
		assert.Equal(t, 90.0, avg.ActBlock.Prot.FrequencyPercent)
		assert.Equal(t, 7.0, avg.ActBlock.Gol.ActivityPercent)
		assert.Equal(t, 33.0, avg.ActBlock.Gol.InitiativePercent)
		assert.Equal(t, 90.0, avg.ActBlock.Gol.FrequencyPercent)
		assert.Equal(t, 0.0, avg.ActBlock.Fra.ActivityPercent)
		assert.Equal(t, 0.0, avg.ActBlock.Fra.InitiativePercent)
		assert.Equal(t, 0.0, avg.ActBlock.Fra.FrequencyPercent)
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
				model.ActBlockData{
					Prot: model.ActivityData{
						ActivityPercent:   90.0,
						InitiativePercent: 80.0,
						FrequencyPercent:  90.0,
					},
					Gol: model.ActivityData{
						ActivityPercent:   7.0,
						InitiativePercent: 33.0,
						FrequencyPercent:  90.0,
					},
					Fra: model.ActivityData{
						ActivityPercent:   0,
						InitiativePercent: 0,
						FrequencyPercent:  0,
					},
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
				model.ActBlockData{
					Prot: model.ActivityData{
						ActivityPercent:   68.0,
						InitiativePercent: 80.0,
						FrequencyPercent:  50.0,
					},
					Gol: model.ActivityData{
						ActivityPercent:   0.0,
						InitiativePercent: 0.0,
						FrequencyPercent:  0.0,
					},
					Fra: model.ActivityData{
						ActivityPercent:   0,
						InitiativePercent: 0,
						FrequencyPercent:  0,
					},
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
				model.ActBlockData{
					Prot: model.ActivityData{
						ActivityPercent:   75.0,
						InitiativePercent: 85.0,
						FrequencyPercent:  70.0,
					},
					Gol: model.ActivityData{
						ActivityPercent:   5.0,
						InitiativePercent: 20.0,
						FrequencyPercent:  60.0,
					},
					Fra: model.ActivityData{
						ActivityPercent:   3.0,
						InitiativePercent: 8.0,
						FrequencyPercent:  12.0,
					},
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
		assert.InDelta(t, 11.7, avg.CommunicativeFuncs.Control, 0.01)
		// (15+25+10)/3 = 16.67
		assert.InDelta(t, 16.7, avg.CommunicativeFuncs.ObtainingDesired, 0.01)
		// (20+30+15)/3 = 21.67
		assert.InDelta(t, 21.7, avg.CommunicativeFuncs.SocialInteraction, 0.01)
		// (25+35+20)/3 = 26.67
		assert.InDelta(t, 26.7, avg.CommunicativeFuncs.InformationExchange, 0.01)

		// Словарь: (100+120+80)/3 = 100
		assert.Equal(t, 100, avg.Vocabulary.ActiveWordsCount)
		// (150+180+120)/3 = 150
		assert.Equal(t, 150, avg.Vocabulary.TotalWordsCount)
		assert.Nil(t, avg.Vocabulary.AdditionalWords)

		// Prot Activity: (90 + 68 + 75) / 3 = 77.67
		assert.InDelta(t, 77.7, avg.ActBlock.Prot.ActivityPercent, 0.01)
		// Prot Initiative: (80 + 80 + 85) / 3 = 81.67
		assert.InDelta(t, 81.7, avg.ActBlock.Prot.InitiativePercent, 0.01)
		// Prot Frequency: (90 + 50 + 70) / 3 = 70.0
		assert.InDelta(t, 70.0, avg.ActBlock.Prot.FrequencyPercent, 0.01)

		// Gol Activity: (7 + 0 + 5) / 3 = 4.0
		assert.InDelta(t, 4.0, avg.ActBlock.Gol.ActivityPercent, 0.01)
		// Gol Initiative: (33 + 0 + 20) / 3 = 17.67
		assert.InDelta(t, 17.7, avg.ActBlock.Gol.InitiativePercent, 0.01)
		// Gol Frequency: (90 + 0 + 60) / 3 = 50.0
		assert.InDelta(t, 50.0, avg.ActBlock.Gol.FrequencyPercent, 0.01)

		// Fra Activity: (0 + 0 + 3) / 3 = 1.0
		assert.InDelta(t, 1.0, avg.ActBlock.Fra.ActivityPercent, 0.01)
		// Fra Initiative: (0 + 0 + 8) / 3 = 2.67
		assert.InDelta(t, 2.7, avg.ActBlock.Fra.InitiativePercent, 0.01)
		// Fra Frequency: (0 + 0 + 12) / 3 = 4.0
		assert.InDelta(t, 4.0, avg.ActBlock.Fra.FrequencyPercent, 0.01)
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
				model.ActBlockData{
					Prot: model.ActivityData{
						ActivityPercent:   68.0,
						InitiativePercent: 80.0,
						FrequencyPercent:  50.0,
					},
					Gol: model.ActivityData{
						ActivityPercent:   0.0,
						InitiativePercent: 0.0,
						FrequencyPercent:  0.0,
					},
					Fra: model.ActivityData{
						ActivityPercent:   0.0,
						InitiativePercent: 0.0,
						FrequencyPercent:  0.0,
					},
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
				model.ActBlockData{
					Prot: model.ActivityData{
						ActivityPercent:   75.0,
						InitiativePercent: 85.0,
						FrequencyPercent:  65.0,
					},
					Gol: model.ActivityData{
						ActivityPercent:   10.0,
						InitiativePercent: 40.0,
						FrequencyPercent:  75.0,
					},
					Fra: model.ActivityData{
						ActivityPercent:   5.0,
						InitiativePercent: 15.0,
						FrequencyPercent:  20.0,
					},
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
				model.ActBlockData{
					Prot: model.ActivityData{
						ActivityPercent:   60.0,
						InitiativePercent: 75.0,
						FrequencyPercent:  45.0,
					},
					Gol: model.ActivityData{
						ActivityPercent:   3.0,
						InitiativePercent: 25.0,
						FrequencyPercent:  50.0,
					},
					Fra: model.ActivityData{
						ActivityPercent:   0.0,
						InitiativePercent: 0.0,
						FrequencyPercent:  0.0,
					},
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
		assert.InDelta(t, 16.7, avg.CommunicativeFuncs.Control, 0.01)
		// (20+30+15)/3 = 21.67
		assert.InDelta(t, 21.7, avg.CommunicativeFuncs.ObtainingDesired, 0.01)
		// (25+35+20)/3 = 26.67
		assert.InDelta(t, 26.7, avg.CommunicativeFuncs.SocialInteraction, 0.01)
		// (30+40+25)/3 = 31.67
		assert.InDelta(t, 31.7, avg.CommunicativeFuncs.InformationExchange, 0.01)

		// Словарь: (110+130+90)/3 = 110
		assert.Equal(t, 110, avg.Vocabulary.ActiveWordsCount)
		// (160+190+130)/3 = 160
		assert.Equal(t, 160, avg.Vocabulary.TotalWordsCount)
		assert.Nil(t, avg.Vocabulary.AdditionalWords)

		// Prot Activity: (68 + 75 + 60) / 3 = 67.67
		assert.InDelta(t, 67.7, avg.ActBlock.Prot.ActivityPercent, 0.01)
		// Prot Initiative: (80 + 85 + 75) / 3 = 80.0
		assert.InDelta(t, 80.0, avg.ActBlock.Prot.InitiativePercent, 0.01)
		// Prot Frequency: (50 + 65 + 45) / 3 = 53.33
		assert.InDelta(t, 53.3, avg.ActBlock.Prot.FrequencyPercent, 0.01)

		// Gol Activity: (0 + 10 + 3) / 3 = 4.33
		assert.InDelta(t, 4.3, avg.ActBlock.Gol.ActivityPercent, 0.01)
		// Gol Initiative: (0 + 40 + 25) / 3 = 21.67
		assert.InDelta(t, 21.7, avg.ActBlock.Gol.InitiativePercent, 0.01)
		// Gol Frequency: (0 + 75 + 50) / 3 = 41.67
		assert.InDelta(t, 41.7, avg.ActBlock.Gol.FrequencyPercent, 0.01)

		// Fra Activity: (0 + 5 + 0) / 3 = 1.67
		assert.InDelta(t, 1.7, avg.ActBlock.Fra.ActivityPercent, 0.01)
		// Fra Initiative: (0 + 15 + 0) / 3 = 5.0
		assert.InDelta(t, 5.0, avg.ActBlock.Fra.InitiativePercent, 0.01)
		// Fra Frequency: (0 + 20 + 0) / 3 = 6.67
		assert.InDelta(t, 6.7, avg.ActBlock.Fra.FrequencyPercent, 0.01)
	})

	t.Run("documents with different dates returns error", func(t *testing.T) {
		docs := []*model.AssessmentDocument{
			createDoc(
				"A",
				"2025-12-12",
				model.LanguageDevelopment{},
				model.CommunicativeFunctions{},
				model.VocabularyData{},
				model.ActBlockData{
					Prot: model.ActivityData{
						ActivityPercent:   90.0,
						InitiativePercent: 80.0,
						FrequencyPercent:  90.0,
					},
					Gol: model.ActivityData{
						ActivityPercent:   7.0,
						InitiativePercent: 33.0,
						FrequencyPercent:  90.0,
					},
					Fra: model.ActivityData{
						ActivityPercent:   0.0,
						InitiativePercent: 0.0,
						FrequencyPercent:  0.0,
					},
				},
			),
			createDoc(
				"B",
				"2025-12-15",
				model.LanguageDevelopment{},
				model.CommunicativeFunctions{},
				model.VocabularyData{},
				model.ActBlockData{
					Prot: model.ActivityData{
						ActivityPercent:   68.0,
						InitiativePercent: 80.0,
						FrequencyPercent:  50.0,
					},
					Gol: model.ActivityData{
						ActivityPercent:   0.0,
						InitiativePercent: 0.0,
						FrequencyPercent:  0.0,
					},
					Fra: model.ActivityData{
						ActivityPercent:   0.0,
						InitiativePercent: 0.0,
						FrequencyPercent:  0.0,
					},
				},
			),
		}

		avg, err := calc.CalculateGroupAverage(docs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "different dates")
		assert.Empty(t, avg.Date)
	})
}
func TestDiffCalculator_CalculateGroupProgress(t *testing.T) {
	calc := &DiffCalculator{}

	// Тестовые средние значения
	earlierAvg := GroupAverage{
		Date: "2025-12-12",
		LanguageLevels: model.LanguageDevelopment{
			Preintentional: model.Preintentional{Activity: 50.0},
			Protolanguage:  model.LanguageActivity{Activity: 30.0, Initiative: 20.0},
			Holophrase:     model.LanguageActivity{Activity: 40.0, Initiative: 35.0},
			Phrase:         model.LanguageActivity{Activity: 60.0, Initiative: 55.0},
		},
		ActBlock: model.ActBlockData{
			Prot: model.ActivityData{
				ActivityPercent:   90.0,
				InitiativePercent: 80.0,
				FrequencyPercent:  90.0,
			},
			Gol: model.ActivityData{
				ActivityPercent:   7.0,
				InitiativePercent: 33.0,
				FrequencyPercent:  90.0,
			},
			Fra: model.ActivityData{
				ActivityPercent:   0.0,
				InitiativePercent: 0.0,
				FrequencyPercent:  0.0,
			},
		},
	}

	laterAvg := GroupAverage{
		Date: "2025-12-15",
		LanguageLevels: model.LanguageDevelopment{
			Preintentional: model.Preintentional{Activity: 55.0},
			Protolanguage:  model.LanguageActivity{Activity: 35.0, Initiative: 25.0},
			Holophrase:     model.LanguageActivity{Activity: 45.0, Initiative: 40.0},
			Phrase:         model.LanguageActivity{Activity: 65.0, Initiative: 60.0},
		},
		ActBlock: model.ActBlockData{
			Prot: model.ActivityData{
				ActivityPercent:   68.0,
				InitiativePercent: 80.0,
				FrequencyPercent:  50.0,
			},
			Gol: model.ActivityData{
				ActivityPercent:   0.0,
				InitiativePercent: 0.0,
				FrequencyPercent:  0.0,
			},
			Fra: model.ActivityData{
				ActivityPercent:   0.0,
				InitiativePercent: 0.0,
				FrequencyPercent:  0.0,
			},
		},
	}

	t.Run("calculate positive progress", func(t *testing.T) {
		progress, err := calc.CalculateGroupProgress(earlierAvg, laterAvg)
		require.NoError(t, err)
		assert.Equal(t, "2025-12-12", progress.PeriodStart)
		assert.Equal(t, "2025-12-15", progress.PeriodEnd)

		// Расчет процентов:
		// Preintentional: ((55-50)/50)*100 = 10%
		assert.InDelta(t, 10.0, progress.LanguageLevels.Preintentional.ActivityPercent, 0.01)
		// Protolanguage: ((35-30)/30)*100 = 16.67%
		assert.InDelta(t, 16.7, progress.LanguageLevels.Protolanguage.ActivityPercent, 0.01)
		// Holophrase: ((45-40)/40)*100 = 12.5%
		assert.InDelta(t, 12.5, progress.LanguageLevels.Holophrase.ActivityPercent, 0.01)
		// Phrase: ((65-60)/60)*100 = 8.33%
		assert.InDelta(t, 8.3, progress.LanguageLevels.Phrase.ActivityPercent, 0.01)

		// Prot Activity: 68.0 - 90.0 = -22.0
		assert.InDelta(t, -22.0, progress.ActBlockDiff.Prot.ActivityDelta, 0.01)
		// Prot Initiative: 80.0 - 80.0 = 0.0
		assert.InDelta(t, 0.0, progress.ActBlockDiff.Prot.InitiativeDelta, 0.01)
		// Prot Frequency: 50.0 - 90.0 = -40.0
		assert.InDelta(t, -40.0, progress.ActBlockDiff.Prot.FrequencyDelta, 0.01)

		// Gol Activity: 0.0 - 7.0 = -7.0
		assert.InDelta(t, -7.0, progress.ActBlockDiff.Gol.ActivityDelta, 0.01)
		// Gol Initiative: 0.0 - 33.0 = -33.0
		assert.InDelta(t, -33.0, progress.ActBlockDiff.Gol.InitiativeDelta, 0.01)
		// Gol Frequency: 0.0 - 90.0 = -90.0
		assert.InDelta(t, -90.0, progress.ActBlockDiff.Gol.FrequencyDelta, 0.01)

		// Fra Activity: 0.0 - 0.0 = 0.0
		assert.InDelta(t, 0.0, progress.ActBlockDiff.Fra.ActivityDelta, 0.01)
		assert.InDelta(t, 0.0, progress.ActBlockDiff.Fra.InitiativeDelta, 0.01)
		assert.InDelta(t, 0.0, progress.ActBlockDiff.Fra.FrequencyDelta, 0.01)
	})

	t.Run("calculate negative progress", func(t *testing.T) {
		// Ухудшение показателей
		negativeLaterAvg := GroupAverage{
			Date: "2025-12-15",
			LanguageLevels: model.LanguageDevelopment{
				Preintentional: model.Preintentional{Activity: 45.0},
				Protolanguage:  model.LanguageActivity{Activity: 25.0},
				Holophrase:     model.LanguageActivity{Activity: 35.0},
				Phrase:         model.LanguageActivity{Activity: 55.0},
			},
			ActBlock: model.ActBlockData{
				Prot: model.ActivityData{
					ActivityPercent:   60.0,
					InitiativePercent: 75.0,
					FrequencyPercent:  40.0,
				},
				Gol: model.ActivityData{
					ActivityPercent:   0.0,
					InitiativePercent: 0.0,
					FrequencyPercent:  0.0,
				},
				Fra: model.ActivityData{
					ActivityPercent:   0.0,
					InitiativePercent: 0.0,
					FrequencyPercent:  0.0,
				},
			},
		}

		progress, err := calc.CalculateGroupProgress(earlierAvg, negativeLaterAvg)
		require.NoError(t, err)

		// Расчет отрицательных процентов:
		// Preintentional: ((45-50)/50)*100 = -10%
		assert.InDelta(t, -10.0, progress.LanguageLevels.Preintentional.ActivityPercent, 0.01)
		// Protolanguage: ((25-30)/30)*100 = -16.67%
		assert.InDelta(t, -16.7, progress.LanguageLevels.Protolanguage.ActivityPercent, 0.01)
		// Holophrase: ((35-40)/40)*100 = -12.5%
		assert.InDelta(t, -12.5, progress.LanguageLevels.Holophrase.ActivityPercent, 0.01)
		// Phrase: ((55-60)/60)*100 = -8.33%
		assert.InDelta(t, -8.3, progress.LanguageLevels.Phrase.ActivityPercent, 0.01)

		// Prot Activity: 60.0 - 90.0 = -30.0
		assert.InDelta(t, -30.0, progress.ActBlockDiff.Prot.ActivityDelta, 0.01)
		// Prot Initiative: 75.0 - 80.0 = -5.0
		assert.InDelta(t, -5.0, progress.ActBlockDiff.Prot.InitiativeDelta, 0.01)
		// Prot Frequency: 40.0 - 90.0 = -50.0
		assert.InDelta(t, -50.0, progress.ActBlockDiff.Prot.FrequencyDelta, 0.01)
	})

	t.Run("error on empty dates", func(t *testing.T) {
		emptyDateAvg := GroupAverage{Date: ""}
		_, err := calc.CalculateGroupProgress(emptyDateAvg, laterAvg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "valid dates")
	})
}

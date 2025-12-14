package file

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseFullStructure(t *testing.T) {
	parser := &DocumentParser{}

	jsonContent := `{
		"por01": "<div>2023-01-15</div>",
		"por02": "<div>student123</div>",
		"newAct01": {"procNumElem": "16%"},
		"newAct02": {"procNumElem": "10%"},
		"newAct03": {"procNumElem": "5%"},
		"newAct04": {"procNumElem": "2%"},
		"diagramBlock": {
			"predActProcNumElem": "34%",
			"protActProcNumElem": "4%",
			"protInitProcNumElem": "0%",
			"golActProcNumElem": "2%",
			"golInitProcNumElem": "33%",
			"fraActProcNumElem": "1%",
			"fraInitProcNumElem": "25%"
		},
		"basicDictionary": [
			{"colorStyle": "res-dict__item-viol", "itemOffStyle": "res-dict__item-off", "content": "что?"},
			{"colorStyle": "res-dict__item-yellow", "itemOffStyle": "", "content": "я, мой"},
			{"colorStyle": "res-dict__item-green", "itemOffStyle": "", "content": "хотеть"}
		],
		"dictBasicMore": ["слово1", "слово2"]
	}`

	doc, err := parser.Parse(bytes.NewBufferString(jsonContent), "test.json")
	require.NoError(t, err)
	require.NotNil(t, doc)

	t.Log("=== Testing parsed document structure ===")

	// Проверяем метаданные
	assert.Equal(t, "student123", doc.Metadata.StudentID)
	assert.Equal(t, "2023-01-15", doc.Metadata.Date.Format("2006-01-02"))
	assert.Equal(t, "test.json", doc.Metadata.FileName)

	// Проверяем коммуникативные функции
	t.Logf("Communicative Functions:")
	t.Logf("  Control: %.1f%% (expected: 16.0%%)", doc.CommunicativeFuncs.Control)
	t.Logf("  ObtainingDesired: %.1f%% (expected: 10.0%%)", doc.CommunicativeFuncs.ObtainingDesired)
	t.Logf("  SocialInteraction: %.1f%% (expected: 5.0%%)", doc.CommunicativeFuncs.SocialInteraction)
	t.Logf("  InformationExchange: %.1f%% (expected: 2.0%%)", doc.CommunicativeFuncs.InformationExchange)

	assert.Equal(t, 16.0, doc.CommunicativeFuncs.Control)
	assert.Equal(t, 10.0, doc.CommunicativeFuncs.ObtainingDesired)
	assert.Equal(t, 5.0, doc.CommunicativeFuncs.SocialInteraction)
	assert.Equal(t, 2.0, doc.CommunicativeFuncs.InformationExchange)

	// Проверяем языковые уровни
	t.Logf("\nLanguage Levels:")
	t.Logf("  Preintentional: %.1f%%", doc.LanguageLevels.Preintentional.Activity)
	t.Logf("  Protolanguage Activity: %.1f%%", doc.LanguageLevels.Protolanguage.Activity)
	t.Logf("  Protolanguage Initiative: %.1f%%", doc.LanguageLevels.Protolanguage.Initiative)
	t.Logf("  Holophrase Activity: %.1f%%", doc.LanguageLevels.Holophrase.Activity)
	t.Logf("  Holophrase Initiative: %.1f%%", doc.LanguageLevels.Holophrase.Initiative)
	t.Logf("  Phrase Activity: %.1f%%", doc.LanguageLevels.Phrase.Activity)
	t.Logf("  Phrase Initiative: %.1f%%", doc.LanguageLevels.Phrase.Initiative)

	assert.Equal(t, 34.0, doc.LanguageLevels.Preintentional.Activity)
	assert.Equal(t, 4.0, doc.LanguageLevels.Protolanguage.Activity)
	assert.Equal(t, 0.0, doc.LanguageLevels.Protolanguage.Initiative)
	assert.Equal(t, 2.0, doc.LanguageLevels.Holophrase.Activity)
	assert.Equal(t, 33.0, doc.LanguageLevels.Holophrase.Initiative)
	assert.Equal(t, 1.0, doc.LanguageLevels.Phrase.Activity)
	assert.Equal(t, 25.0, doc.LanguageLevels.Phrase.Initiative)

	// Проверяем словарный запас
	t.Logf("\nVocabulary:")
	t.Logf("  ActiveWordsCount: %d (expected: 2)", doc.Vocabulary.ActiveWordsCount)
	t.Logf("  AdditionalWords: %v", doc.Vocabulary.AdditionalWords)
	t.Logf("  TotalWordsCount: %d (expected: 4)", doc.Vocabulary.TotalWordsCount)

	assert.Equal(t, 2, doc.Vocabulary.ActiveWordsCount) // 2 слова с itemOffStyle=""
	assert.Equal(t, []string{"слово1", "слово2"}, doc.Vocabulary.AdditionalWords)
	assert.Equal(t, 4, doc.Vocabulary.TotalWordsCount)

	t.Log("=== All checks passed ===")
}

func TestParser_CountActiveWords(t *testing.T) {
	parser := &DocumentParser{}

	testCases := []struct {
		name           string
		jsonContent    string
		expectedActive int
	}{
		{
			name: "All active words",
			jsonContent: `{
				"por01": "<div>2023-01-01</div>",
				"por02": "<div>123</div>",
				"basicDictionary": [
					{"itemOffStyle": ""},
					{"itemOffStyle": ""},
					{"itemOffStyle": ""}
				],
				"dictBasicMore": []
			}`,
			expectedActive: 3,
		},
		{
			name: "Mixed active/inactive",
			jsonContent: `{
				"por01": "<div>2023-01-01</div>",
				"por02": "<div>123</div>",
				"basicDictionary": [
					{"itemOffStyle": "res-dict__item-off"},
					{"itemOffStyle": ""},
					{"itemOffStyle": "res-dict__item-off"},
					{"itemOffStyle": ""}
				],
				"dictBasicMore": ["extra1", "extra2"]
			}`,
			expectedActive: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := parser.Parse(bytes.NewBufferString(tc.jsonContent), "test.json")
			require.NoError(t, err)
			assert.Equal(t, tc.expectedActive, doc.Vocabulary.ActiveWordsCount)

			// Проверяем общее количество слов
			expectedTotal := tc.expectedActive + len(doc.Vocabulary.AdditionalWords)
			assert.Equal(t, expectedTotal, doc.Vocabulary.TotalWordsCount)

			t.Logf("Test '%s': Active=%d, Additional=%d, Total=%d",
				tc.name,
				doc.Vocabulary.ActiveWordsCount,
				len(doc.Vocabulary.AdditionalWords),
				doc.Vocabulary.TotalWordsCount)
		})
	}
}

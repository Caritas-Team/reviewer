package file

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed test_file.json
var jsonContent []byte

func TestParser_ParseFullStructure(t *testing.T) {
	parser := &DocumentParser{}

	doc, err := parser.Parse(bytes.NewBuffer(jsonContent), "test.json")
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Проверяем метаданные
	assert.Equal(t, "student123", doc.Metadata.StudentID)
	assert.Equal(t, "2023-01-15", doc.Metadata.Date.Format("2006-01-02"))
	assert.Equal(t, "test.json", doc.Metadata.FileName)

	assert.Equal(t, 16.0, doc.CommunicativeFuncs.Control)
	assert.Equal(t, 10.0, doc.CommunicativeFuncs.ObtainingDesired)
	assert.Equal(t, 5.0, doc.CommunicativeFuncs.SocialInteraction)
	assert.Equal(t, 2.0, doc.CommunicativeFuncs.InformationExchange)

	assert.Equal(t, 34.0, doc.LanguageLevels.Preintentional.Activity)
	assert.Equal(t, 4.0, doc.LanguageLevels.Protolanguage.Activity)
	assert.Equal(t, 0.0, doc.LanguageLevels.Protolanguage.Initiative)
	assert.Equal(t, 2.0, doc.LanguageLevels.Holophrase.Activity)
	assert.Equal(t, 33.0, doc.LanguageLevels.Holophrase.Initiative)
	assert.Equal(t, 1.0, doc.LanguageLevels.Phrase.Activity)
	assert.Equal(t, 25.0, doc.LanguageLevels.Phrase.Initiative)

	assert.Equal(t, 2, doc.Vocabulary.ActiveWordsCount) // 2 слова с itemOffStyle=""
	assert.Equal(t, []string{"слово1", "слово2"}, doc.Vocabulary.AdditionalWords)
	assert.Equal(t, 4, doc.Vocabulary.TotalWordsCount)
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
		})
	}
}

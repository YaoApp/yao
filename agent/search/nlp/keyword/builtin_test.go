package keyword

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuiltinExtractor_Extract(t *testing.T) {
	extractor := NewBuiltinExtractor()

	tests := []struct {
		name     string
		text     string
		limit    int
		minCount int // minimum expected keywords
	}{
		{
			name:     "English text",
			text:     "The quick brown fox jumps over the lazy dog. The fox is very quick.",
			limit:    5,
			minCount: 3, // fox, quick, etc.
		},
		{
			name:     "Chinese text",
			text:     "人工智能技术正在快速发展，机器学习和深度学习是人工智能的核心技术",
			limit:    5,
			minCount: 2,
		},
		{
			name:     "Mixed text",
			text:     "AI人工智能 machine learning 机器学习 deep learning 深度学习",
			limit:    10,
			minCount: 3,
		},
		{
			name:     "Empty text",
			text:     "",
			limit:    5,
			minCount: 0,
		},
		{
			name:     "Only stop words",
			text:     "the a an is are was were",
			limit:    5,
			minCount: 0,
		},
		{
			name:     "Technical query",
			text:     "How to implement a search engine with Elasticsearch and Redis caching?",
			limit:    5,
			minCount: 3, // search, engine, elasticsearch, redis, caching
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := extractor.Extract(tt.text, tt.limit)
			assert.GreaterOrEqual(t, len(results), tt.minCount, "Expected at least %d keywords", tt.minCount)
			assert.LessOrEqual(t, len(results), tt.limit, "Should not exceed limit")

			// Check scores are valid
			for _, r := range results {
				assert.NotEmpty(t, r.Word)
				assert.GreaterOrEqual(t, r.Score, 0.0)
				assert.LessOrEqual(t, r.Score, 1.0)
			}
		})
	}
}

func TestBuiltinExtractor_ExtractAsStrings(t *testing.T) {
	extractor := NewBuiltinExtractor()

	text := "Machine learning and deep learning are subfields of artificial intelligence"
	keywords := extractor.ExtractAsStrings(text, 5)

	assert.NotEmpty(t, keywords)
	assert.LessOrEqual(t, len(keywords), 5)

	// Check that common ML terms are extracted
	keywordSet := make(map[string]bool)
	for _, k := range keywords {
		keywordSet[k] = true
	}
	assert.True(t, keywordSet["learning"] || keywordSet["machine"] || keywordSet["artificial"],
		"Expected at least one relevant keyword")
}

func TestBuiltinExtractor_StopWords(t *testing.T) {
	extractor := NewBuiltinExtractor()

	// Test that stop words are filtered
	text := "the quick brown fox is very lazy"
	results := extractor.Extract(text, 10)

	for _, r := range results {
		assert.NotEqual(t, "the", r.Word)
		assert.NotEqual(t, "is", r.Word)
		assert.NotEqual(t, "very", r.Word)
	}
}

func TestBuiltinExtractor_Frequency(t *testing.T) {
	extractor := NewBuiltinExtractor()

	// Word "search" appears 3 times, should rank higher
	text := "search engine optimization, search ranking, search results"
	results := extractor.Extract(text, 3)

	assert.NotEmpty(t, results)
	// "search" should be the top keyword
	assert.Equal(t, "search", results[0].Word)
	assert.Equal(t, 1.0, results[0].Score) // highest frequency = 1.0
}

func TestBuiltinExtractor_ZeroLimit(t *testing.T) {
	extractor := NewBuiltinExtractor()

	results := extractor.Extract("some text here", 0)
	assert.Empty(t, results)
}

func TestBuiltinExtractor_ChineseStopWords(t *testing.T) {
	extractor := NewBuiltinExtractor()

	// Test that Chinese stop words are filtered
	text := "这是一个关于人工智能的文章"
	results := extractor.Extract(text, 10)

	for _, r := range results {
		assert.NotEqual(t, "这", r.Word)
		assert.NotEqual(t, "是", r.Word)
		assert.NotEqual(t, "一个", r.Word)
		assert.NotEqual(t, "的", r.Word)
	}
}

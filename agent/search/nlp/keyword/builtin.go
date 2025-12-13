package keyword

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// BuiltinExtractor implements simple frequency-based keyword extraction
// This is a lightweight implementation with no external dependencies.
//
// Algorithm:
//  1. Tokenize text (split by whitespace and punctuation)
//  2. Normalize (lowercase, trim)
//  3. Filter stop words and short words
//  4. Count word frequency
//  5. Return top N words by frequency
//
// Limitations:
//   - No semantic understanding
//   - No phrase extraction (single words only)
//   - Basic Chinese support (splits by punctuation, no proper segmentation)
//
// For better results, use Agent or MCP mode with LLM-based extraction.
type BuiltinExtractor struct {
	stopWords map[string]bool
	minLength int // minimum word length to consider
}

// Result represents an extracted keyword with its score
type Result struct {
	Word  string  `json:"word"`
	Score float64 `json:"score"` // frequency-based score (0-1)
}

// NewBuiltinExtractor creates a new builtin keyword extractor
func NewBuiltinExtractor() *BuiltinExtractor {
	return &BuiltinExtractor{
		stopWords: defaultStopWords,
		minLength: 2,
	}
}

// Extract extracts keywords from text using frequency-based algorithm
func (e *BuiltinExtractor) Extract(text string, limit int) []Result {
	if text == "" || limit <= 0 {
		return []Result{}
	}

	// Step 1: Tokenize
	tokens := e.tokenize(text)

	// Step 2 & 3: Normalize and filter
	var words []string
	for _, token := range tokens {
		word := e.normalize(token)
		if e.shouldKeep(word) {
			words = append(words, word)
		}
	}

	if len(words) == 0 {
		return []Result{}
	}

	// Step 4: Count frequency
	freq := make(map[string]int)
	for _, word := range words {
		freq[word]++
	}

	// Step 5: Sort by frequency and return top N
	type wordFreq struct {
		word string
		freq int
	}
	var sorted []wordFreq
	for word, count := range freq {
		sorted = append(sorted, wordFreq{word, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].freq > sorted[j].freq
	})

	// Calculate max frequency for normalization
	maxFreq := 1
	if len(sorted) > 0 {
		maxFreq = sorted[0].freq
	}

	// Build result with normalized scores
	result := make([]Result, 0, limit)
	for i := 0; i < len(sorted) && i < limit; i++ {
		result = append(result, Result{
			Word:  sorted[i].word,
			Score: float64(sorted[i].freq) / float64(maxFreq),
		})
	}

	return result
}

// ExtractAsStrings is a convenience method that returns just the keyword strings
func (e *BuiltinExtractor) ExtractAsStrings(text string, limit int) []string {
	results := e.Extract(text, limit)
	words := make([]string, len(results))
	for i, r := range results {
		words[i] = r.Word
	}
	return words
}

// tokenize splits text into tokens
// Handles both English (space-separated) and Chinese (character-based with punctuation splits)
func (e *BuiltinExtractor) tokenize(text string) []string {
	// Split by whitespace and common punctuation
	splitter := regexp.MustCompile(`[\s\p{P}\p{S}]+`)
	tokens := splitter.Split(text, -1)

	// Further split mixed Chinese/English text
	var result []string
	for _, token := range tokens {
		if token == "" {
			continue
		}
		// Split Chinese characters as individual tokens (basic approach)
		// For proper Chinese segmentation, use Agent/MCP mode
		subTokens := e.splitMixedText(token)
		result = append(result, subTokens...)
	}

	return result
}

// splitMixedText handles mixed Chinese/English text
// Chinese characters are grouped together, English words stay as-is
func (e *BuiltinExtractor) splitMixedText(text string) []string {
	var result []string
	var current strings.Builder
	var lastType int // 0=none, 1=chinese, 2=other

	for _, r := range text {
		currentType := 0
		if unicode.Is(unicode.Han, r) {
			currentType = 1
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			currentType = 2
		}

		if currentType == 0 {
			// Non-word character, flush current
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
			lastType = 0
			continue
		}

		if lastType != 0 && lastType != currentType {
			// Type changed, flush current
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		}

		current.WriteRune(r)
		lastType = currentType
	}

	// Flush remaining
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// normalize converts word to lowercase and trims whitespace
func (e *BuiltinExtractor) normalize(word string) string {
	return strings.ToLower(strings.TrimSpace(word))
}

// shouldKeep checks if a word should be kept (not a stop word, meets length requirement)
func (e *BuiltinExtractor) shouldKeep(word string) bool {
	if len(word) < e.minLength {
		return false
	}
	if e.stopWords[word] {
		return false
	}
	// Keep if it contains at least one letter or Chinese character
	for _, r := range word {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// defaultStopWords contains common stop words for English and Chinese
// This is a minimal set to keep the implementation lightweight.
// For comprehensive stop word filtering, use Agent/MCP mode.
var defaultStopWords = map[string]bool{
	// English stop words (most common ~100)
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "do": true, "does": true, "did": true,
	"will": true, "would": true, "could": true, "should": true, "may": true, "might": true,
	"must": true, "shall": true, "can": true, "need": true, "dare": true,
	"i": true, "you": true, "he": true, "she": true, "it": true, "we": true, "they": true,
	"me": true, "him": true, "her": true, "us": true, "them": true,
	"my": true, "your": true, "his": true, "its": true, "our": true, "their": true,
	"mine": true, "yours": true, "hers": true, "ours": true, "theirs": true,
	"this": true, "that": true, "these": true, "those": true,
	"what": true, "which": true, "who": true, "whom": true, "whose": true,
	"where": true, "when": true, "why": true, "how": true,
	"all": true, "each": true, "every": true, "both": true, "few": true, "more": true,
	"most": true, "other": true, "some": true, "such": true, "no": true, "not": true,
	"only": true, "same": true, "so": true, "than": true, "too": true, "very": true,
	"just": true, "also": true, "now": true, "here": true, "there": true,
	"in": true, "on": true, "at": true, "by": true, "for": true, "with": true,
	"about": true, "against": true, "between": true, "into": true, "through": true,
	"during": true, "before": true, "after": true, "above": true, "below": true,
	"to": true, "from": true, "up": true, "down": true, "out": true, "off": true,
	"over": true, "under": true, "again": true, "further": true, "then": true, "once": true,
	"as": true, "if": true, "because": true, "until": true, "while": true,

	// Chinese stop words (most common ~50)
	"的": true, "了": true, "和": true, "是": true, "就": true,
	"都": true, "而": true, "及": true, "与": true, "着": true,
	"或": true, "一个": true, "没有": true, "我们": true, "你们": true,
	"他们": true, "它们": true, "这个": true, "那个": true, "这些": true,
	"那些": true, "这里": true, "那里": true, "什么": true, "怎么": true,
	"为什么": true, "哪里": true, "谁": true, "哪个": true, "多少": true,
	"在": true, "有": true, "个": true, "中": true, "为": true,
	"以": true, "于": true, "上": true, "下": true, "不": true,
	"也": true, "很": true, "到": true, "说": true, "要": true,
	"会": true, "可以": true, "这": true, "那": true, "但": true,
	"如果": true, "因为": true, "所以": true, "虽然": true, "但是": true,
}

package core

import (
	"testing"
)

func TestTokensFindStringSubmatch(t *testing.T) {
	propTokens := Tokens{
		{start: "{%", end: "%}"},
		{start: "[{", end: "}]"},
	}

	dataTokens := Tokens{
		{start: "{{", end: "}}"},
	}

	tests := []struct {
		name     string
		tokens   Tokens
		input    string
		expected []string
	}{
		{
			name:     "Match with propTokens - {% xxx %}",
			tokens:   propTokens,
			input:    "This is a test string {% match %} with tokens.",
			expected: []string{"{% match %}"},
		},
		{
			name:     "Match with propTokens - [{ xxx }]",
			tokens:   propTokens,
			input:    "This is a test string [{ match }] with tokens.",
			expected: []string{"[{ match }]"},
		},
		{
			name:     "Match with dataTokens - {{ xxx }}",
			tokens:   dataTokens,
			input:    "This is a test string {{ match }} with tokens.",
			expected: []string{"{{ match }}"},
		},
		{
			name:     "No match",
			tokens:   propTokens,
			input:    "This string has no matching tokens.",
			expected: nil,
		},
		{
			name:     "Partial match start token",
			tokens:   propTokens,
			input:    "This string has a partial {% match.",
			expected: nil,
		},
		{
			name:     "Nested tokens",
			tokens:   propTokens,
			input:    "This string has {% outer {% inner %} outer %} tokens.",
			expected: []string{"{% outer {% inner %} outer %}"},
		},
		{
			name:     "Multiple matches",
			tokens:   propTokens,
			input:    "This string has {% first %} and [{ second }] tokens.",
			expected: []string{"{% first %}", "[{ second }]"},
		},
		{
			name:     "Empty input",
			tokens:   propTokens,
			input:    "",
			expected: nil,
		},
		{
			name:     "Adjacent tokens",
			tokens:   propTokens,
			input:    "{% first %}{% second %}",
			expected: []string{"{% first %}", "{% second %}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tokens.FindStringSubmatch(tt.input)
			if !stringsEqual(result, tt.expected) {
				t.Errorf("%s: Expected %v, got %v", tt.input, tt.expected, result)
			}
		})
	}
}

func TestTokensFindAllStringSubmatch(t *testing.T) {
	propTokens := Tokens{
		{start: "{%", end: "%}"},
		{start: "[{", end: "}]"},
	}

	dataTokens := Tokens{
		{start: "{{", end: "}}"},
	}

	tests := []struct {
		name     string
		tokens   Tokens
		input    string
		n        int
		expected [][]string
	}{
		{
			name:     "Single match with propTokens - {% xxx %}",
			tokens:   propTokens,
			input:    "This is a test string {% match %} with tokens.",
			n:        -1, // -1 indicates no limit
			expected: [][]string{{"{% match %}", "match"}},
		},
		{
			name:     "Multiple matches with propTokens",
			tokens:   propTokens,
			input:    "This string has {% first %} and [{ second }] tokens.",
			n:        -1,
			expected: [][]string{{"{% first %}", "first"}, {"[{ second }]", "second"}},
		},
		{
			name:     "Nested tokens with propTokens",
			tokens:   propTokens,
			input:    "This string has {% outer {% inner %} outer %} tokens.",
			n:        -1,
			expected: [][]string{{"{% outer {% inner %} outer %}", "outer {% inner %} outer"}},
		},
		{
			name:     "Match with dataTokens - {{ xxx }}",
			tokens:   dataTokens,
			input:    "This is a test string {{ match }} with tokens.",
			n:        -1,
			expected: [][]string{{"{{ match }}", "match"}},
		},
		{
			name:     "Multiple matches with dataTokens",
			tokens:   dataTokens,
			input:    "This string has {{ first }} and {{ second }} tokens.",
			n:        -1,
			expected: [][]string{{"{{ first }}", "first"}, {"{{ second }}", "second"}},
		},
		{
			name:     "No match",
			tokens:   propTokens,
			input:    "This string has no matching tokens.",
			n:        -1,
			expected: nil,
		},
		{
			name:     "Partial match start token",
			tokens:   propTokens,
			input:    "This string has a partial {% match.",
			n:        -1,
			expected: nil,
		},
		{
			name:     "Empty input",
			tokens:   propTokens,
			input:    "",
			n:        -1,
			expected: nil,
		},
		{
			name:     "Limit matches",
			tokens:   propTokens,
			input:    "{% first %}{% second %}{% third %}",
			n:        2,
			expected: [][]string{{"{% first %}", "first"}, {"{% second %}", "second"}, {"{% third %}", "third"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tokens.FindAllStringSubmatch(tt.input, tt.n)
			if !string2dEqual(result, tt.expected) {
				t.Errorf("%s: expected %v, got %v", tt.input, tt.expected, result)
			}
		})
	}
}

func TestTokensMatchString(t *testing.T) {
	propTokens := Tokens{
		{start: "{%", end: "%}"},
		{start: "[{", end: "}]"},
	}

	dataTokens := Tokens{
		{start: "{{", end: "}}"},
	}

	tests := []struct {
		name     string
		tokens   Tokens
		input    string
		expected bool
	}{
		{
			name:     "Single match with propTokens - {% xxx %}",
			tokens:   propTokens,
			input:    "This is a test string {% match %} with tokens.",
			expected: true,
		},
		{
			name:     "Multiple matches with propTokens",
			tokens:   propTokens,
			input:    "This string has {% first %} and [{ second }] tokens.",
			expected: true,
		},
		{
			name:     "Nested tokens with propTokens",
			tokens:   propTokens,
			input:    "This string has {% outer {% inner %} outer %} tokens.",
			expected: true,
		},
		{
			name:     "Match with dataTokens - {{ xxx }}",
			tokens:   dataTokens,
			input:    "This is a test string {{ match }} with tokens.",
			expected: true,
		},
		{
			name:     "Multiple matches with dataTokens",
			tokens:   dataTokens,
			input:    "This string has {{ first }} and {{ second }} tokens.",
			expected: true,
		},
		{
			name:     "No match",
			tokens:   propTokens,
			input:    "This string has no matching tokens.",
			expected: false,
		},
		{
			name:     "Partial match start token",
			tokens:   propTokens,
			input:    "This string has a partial {% match.",
			expected: false,
		},
		{
			name:     "Empty input",
			tokens:   propTokens,
			input:    "",
			expected: false,
		},
		{
			name:     "Token at the end",
			tokens:   propTokens,
			input:    "Token at the end {% last %}",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tokens.MatchString(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTokensReplaceAllStringFunc(t *testing.T) {
	propTokens := Tokens{
		{start: "{%", end: "%}"},
		{start: "[{", end: "}]"},
	}

	dataTokens := Tokens{
		{start: "{{", end: "}}"},
	}

	tests := []struct {
		name     string
		tokens   Tokens
		input    string
		repl     func(string) string
		expected string
	}{
		{
			name:     "Single replacement with propTokens",
			tokens:   propTokens,
			input:    "This is a test string {% replace this %}.",
			repl:     func(s string) string { return "[REPLACED]" },
			expected: "This is a test string [REPLACED].",
		},
		{
			name:     "Multiple replacements with propTokens",
			tokens:   propTokens,
			input:    "This string {% first %} and [{ second }] will be replaced.",
			repl:     func(s string) string { return "REPLACED" },
			expected: "This string REPLACED and REPLACED will be replaced.",
		},
		{
			name:     "Nested replacements with propTokens",
			tokens:   propTokens,
			input:    "This string has {% outer {% inner %} outer %} tokens.",
			repl:     func(s string) string { return "[NESTED]" },
			expected: "This string has [NESTED] tokens.",
		},
		{
			name:     "Replacement with dataTokens",
			tokens:   dataTokens,
			input:    "This is a test string {{ replace this }}.",
			repl:     func(s string) string { return "REPLACED" },
			expected: "This is a test string REPLACED.",
		},
		{
			name:     "Multiple replacements with dataTokens",
			tokens:   dataTokens,
			input:    "This string has {{ first }} and {{ second }} tokens.",
			repl:     func(s string) string { return "REPLACED" },
			expected: "This string has REPLACED and REPLACED tokens.",
		},
		{
			name:     "No match",
			tokens:   propTokens,
			input:    "This string has no matching tokens.",
			repl:     func(s string) string { return "REPLACED" },
			expected: "This string has no matching tokens.",
		},
		{
			name:     "Partial match start token",
			tokens:   propTokens,
			input:    "This string has a partial {% match.",
			repl:     func(s string) string { return "REPLACED" },
			expected: "This string has a partial {% match.",
		},
		{
			name:     "Empty input",
			tokens:   propTokens,
			input:    "",
			repl:     func(s string) string { return "REPLACED" },
			expected: "",
		},
		{
			name:     "Token at the end",
			tokens:   propTokens,
			input:    "Token at the end {% last %}",
			repl:     func(s string) string { return "REPLACED" },
			expected: "Token at the end REPLACED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tokens.ReplaceAllStringFunc(tt.input, tt.repl)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}

}

func TestTokensReplaceAllString(t *testing.T) {

	propTokens := Tokens{
		{start: "{%", end: "%}"},
		{start: "[{", end: "}]"},
	}

	dataTokens := Tokens{
		{start: "{{", end: "}}"},
	}

	tests := []struct {
		name     string
		tokens   Tokens
		input    string
		repl     string
		expected string
	}{
		{
			name:     "Single replacement with propTokens",
			tokens:   propTokens,
			input:    "This is a test string {% replace this %}.",
			repl:     "[REPLACED]",
			expected: "This is a test string [REPLACED].",
		},
		{
			name:     "Multiple replacements with propTokens",
			tokens:   propTokens,
			input:    "This string {% first %} and [{ second }] will be replaced.",
			repl:     "REPLACED",
			expected: "This string REPLACED and REPLACED will be replaced.",
		},
		{
			name:     "Nested replacements with propTokens",
			tokens:   propTokens,
			input:    "This string has {% outer {% inner %} outer %} tokens.",
			repl:     "[NESTED]",
			expected: "This string has [NESTED] tokens.",
		},
		{
			name:     "Replacement with dataTokens",
			tokens:   dataTokens,
			input:    "This is a test string {{ replace this }}.",
			repl:     "REPLACED",
			expected: "This is a test string REPLACED.",
		},
		{
			name:     "Multiple replacements with dataTokens",
			tokens:   dataTokens,
			input:    "This string has {{ first }} and {{ second }} tokens.",
			repl:     "REPLACED",
			expected: "This string has REPLACED and REPLACED tokens.",
		},
		{
			name:     "No match",
			tokens:   propTokens,
			input:    "This string has no matching tokens.",
			repl:     "REPLACED",
			expected: "This string has no matching tokens.",
		},
		{
			name:     "Partial match start token",
			tokens:   propTokens,
			input:    "This string has a partial {% match.",
			repl:     "REPLACED",
			expected: "This string has a partial {% match.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tokens.ReplaceAllString(tt.input, tt.repl)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}

}

// Helper function to check equality of two slices of slices of strings
func string2dEqual(a, b [][]string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}

func stringsEqual(a, b []string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

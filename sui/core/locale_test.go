package core

import (
	"testing"
)

func TestLocaleMergeTranslations(t *testing.T) {
	tests := []struct {
		name         string
		locale       Locale
		translations []Translation
		prefix       string
		expectedKeys map[string]string
		expectedMsgs map[string]string
	}{
		{
			name: "Empty translations",
			locale: Locale{
				Keys:     map[string]string{},
				Messages: map[string]string{},
			},
			translations: []Translation{},
			prefix:       "",
			expectedKeys: map[string]string{},
			expectedMsgs: map[string]string{},
		},
		{
			name: "Nil Keys and Messages",
			locale: Locale{
				Keys:     nil,
				Messages: nil,
			},
			translations: []Translation{
				{Key: "greeting", Message: "Hello"},
			},
			prefix: "",
			expectedKeys: map[string]string{
				"greeting": "Hello",
			},
			expectedMsgs: map[string]string{
				"Hello": "Hello",
			},
		},
		{
			name: "With prefix",
			locale: Locale{
				Keys:     map[string]string{},
				Messages: map[string]string{},
			},
			translations: []Translation{
				{Key: "prefix_1", Message: "Hello"},
				{Key: "other_1", Message: "World"},
			},
			prefix: "prefix",
			expectedKeys: map[string]string{
				"prefix_1": "Hello",
			},
			expectedMsgs: map[string]string{
				"Hello": "Hello",
			},
		},
		{
			name: "Update existing keys and values",
			locale: Locale{
				Keys: map[string]string{
					"greeting": "Hi",
				},
				Messages: map[string]string{
					"Hi": "Hi",
				},
			},
			translations: []Translation{
				{Key: "greeting", Message: "Hello"},
			},
			prefix: "",
			expectedKeys: map[string]string{
				"greeting": "Hello",
			},
			expectedMsgs: map[string]string{
				"Hi":    "Hi",
				"Hello": "Hello",
			},
		},
		{
			name: "Duplicate messages",
			locale: Locale{
				Keys:     map[string]string{},
				Messages: map[string]string{},
			},
			translations: []Translation{
				{Key: "welcome", Message: "Hello"},
				{Key: "farewell", Message: "Hello"},
			},
			prefix: "",
			expectedKeys: map[string]string{
				"welcome":  "Hello",
				"farewell": "Hello",
			},
			expectedMsgs: map[string]string{
				"Hello": "Hello",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.locale.MergeTranslations(tt.translations, tt.prefix)
			if !testCompareMaps(tt.locale.Keys, tt.expectedKeys) {
				t.Errorf("expected keys %v, got %v", tt.expectedKeys, tt.locale.Keys)
			}
			if !testCompareMaps(tt.locale.Messages, tt.expectedMsgs) {
				t.Errorf("expected messages %v, got %v", tt.expectedMsgs, tt.locale.Messages)
			}
		})
	}
}

func TestLocaleMerge(t *testing.T) {
	tests := []struct {
		name         string
		locale       Locale
		locale2      Locale
		expectedKeys map[string]string
		expectedMsgs map[string]string
	}{
		{
			name: "Nil Keys and Messages in locale2",
			locale: Locale{
				Keys:     map[string]string{"greeting": "Hello"},
				Messages: map[string]string{"Hello": "Hello"},
			},
			locale2: Locale{
				Keys:     nil,
				Messages: nil,
			},
			expectedKeys: map[string]string{"greeting": "Hello"},
			expectedMsgs: map[string]string{"Hello": "Hello"},
		},
		{
			name: "Nil Keys and Messages in locale",
			locale: Locale{
				Keys:     nil,
				Messages: nil,
			},
			locale2: Locale{
				Keys:     map[string]string{"farewell": "Goodbye"},
				Messages: map[string]string{"Goodbye": "Goodbye"},
			},
			expectedKeys: map[string]string{"farewell": "Goodbye"},
			expectedMsgs: map[string]string{"Goodbye": "Goodbye"},
		},
		{
			name: "Merge non-existing keys and messages",
			locale: Locale{
				Keys:     map[string]string{"greeting": "Hello"},
				Messages: map[string]string{"Hello": "Hello"},
			},
			locale2: Locale{
				Keys:     map[string]string{"farewell": "Goodbye"},
				Messages: map[string]string{"Goodbye": "Goodbye"},
			},
			expectedKeys: map[string]string{
				"greeting": "Hello",
				"farewell": "Goodbye",
			},
			expectedMsgs: map[string]string{
				"Hello":   "Hello",
				"Goodbye": "Goodbye",
			},
		},
		{
			name: "Merge with existing keys and messages",
			locale: Locale{
				Keys:     map[string]string{"greeting": "Hello"},
				Messages: map[string]string{"Hello": "Hello"},
			},
			locale2: Locale{
				Keys:     map[string]string{"greeting": "Hi"},
				Messages: map[string]string{"Hello": "Hi"},
			},
			expectedKeys: map[string]string{"greeting": "Hello"},
			expectedMsgs: map[string]string{"Hello": "Hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.locale.Merge(tt.locale2)
			if !testCompareMaps(tt.locale.Keys, tt.expectedKeys) {
				t.Errorf("expected keys %v, got %v", tt.expectedKeys, tt.locale.Keys)
			}
			if !testCompareMaps(tt.locale.Messages, tt.expectedMsgs) {
				t.Errorf("expected messages %v, got %v", tt.expectedMsgs, tt.locale.Messages)
			}
		})
	}
}

func testCompareMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

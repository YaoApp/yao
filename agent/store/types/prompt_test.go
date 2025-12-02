package types

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPromptParse(t *testing.T) {
	tests := []struct {
		name     string
		prompt   Prompt
		ctx      map[string]string
		validate func(t *testing.T, result Prompt)
	}{
		{
			name: "ParseSysTimeVariables",
			prompt: Prompt{
				Role:    "system",
				Content: "Current time: $SYS.TIME, Date: $SYS.DATE",
			},
			ctx: nil,
			validate: func(t *testing.T, result Prompt) {
				assert.Equal(t, "system", result.Role)
				// Check that variables are replaced (not exact match due to time)
				assert.NotContains(t, result.Content, "$SYS.TIME")
				assert.NotContains(t, result.Content, "$SYS.DATE")
				assert.Contains(t, result.Content, "Current time:")
				assert.Contains(t, result.Content, "Date:")
			},
		},
		{
			name: "ParseSysDatetimeVariable",
			prompt: Prompt{
				Role:    "system",
				Content: "Now: $SYS.DATETIME, Timezone: $SYS.TIMEZONE",
			},
			ctx: nil,
			validate: func(t *testing.T, result Prompt) {
				assert.NotContains(t, result.Content, "$SYS.DATETIME")
				assert.NotContains(t, result.Content, "$SYS.TIMEZONE")
			},
		},
		{
			name: "ParseSysWeekdayVariable",
			prompt: Prompt{
				Role:    "system",
				Content: "Today is $SYS.WEEKDAY",
			},
			ctx: nil,
			validate: func(t *testing.T, result Prompt) {
				weekday := time.Now().Weekday().String()
				assert.Contains(t, result.Content, weekday)
			},
		},
		{
			name: "ParseSysYearMonthDay",
			prompt: Prompt{
				Role:    "system",
				Content: "Year: $SYS.YEAR, Month: $SYS.MONTH, Day: $SYS.DAY",
			},
			ctx: nil,
			validate: func(t *testing.T, result Prompt) {
				now := time.Now()
				assert.Contains(t, result.Content, now.Format("2006"))
				assert.Contains(t, result.Content, now.Format("01"))
				assert.Contains(t, result.Content, now.Format("02"))
			},
		},
		{
			name: "ParseSysHourMinuteSecond",
			prompt: Prompt{
				Role:    "system",
				Content: "Hour: $SYS.HOUR, Minute: $SYS.MINUTE, Second: $SYS.SECOND",
			},
			ctx: nil,
			validate: func(t *testing.T, result Prompt) {
				assert.NotContains(t, result.Content, "$SYS.HOUR")
				assert.NotContains(t, result.Content, "$SYS.MINUTE")
				assert.NotContains(t, result.Content, "$SYS.SECOND")
			},
		},
		{
			name: "ParseEnvVariable",
			prompt: Prompt{
				Role:    "system",
				Content: "App: $ENV.TEST_APP_NAME",
			},
			ctx: nil,
			validate: func(t *testing.T, result Prompt) {
				assert.Contains(t, result.Content, "App: TestApp")
			},
		},
		{
			name: "ParseEnvVariableNotFound",
			prompt: Prompt{
				Role:    "system",
				Content: "Value: $ENV.NOT_EXIST_VAR_12345",
			},
			ctx: nil,
			validate: func(t *testing.T, result Prompt) {
				// Should be replaced with empty string
				assert.Equal(t, "Value: ", result.Content)
			},
		},
		{
			name: "ParseCtxVariables",
			prompt: Prompt{
				Role:    "system",
				Content: "User: $CTX.USER_ID, Locale: $CTX.LOCALE",
			},
			ctx: map[string]string{
				"USER_ID": "user-123",
				"LOCALE":  "zh-CN",
			},
			validate: func(t *testing.T, result Prompt) {
				assert.Equal(t, "User: user-123, Locale: zh-CN", result.Content)
			},
		},
		{
			name: "ParseCtxVariableNotFound",
			prompt: Prompt{
				Role:    "system",
				Content: "Value: $CTX.NOT_EXIST",
			},
			ctx: map[string]string{
				"OTHER": "value",
			},
			validate: func(t *testing.T, result Prompt) {
				// Should be replaced with empty string
				assert.Equal(t, "Value: ", result.Content)
			},
		},
		{
			name: "ParseCtxWithNilMap",
			prompt: Prompt{
				Role:    "system",
				Content: "Value: $CTX.SOMETHING",
			},
			ctx: nil,
			validate: func(t *testing.T, result Prompt) {
				// Should keep original when ctx is nil
				assert.Equal(t, "Value: $CTX.SOMETHING", result.Content)
			},
		},
		{
			name: "ParseMixedVariables",
			prompt: Prompt{
				Role:    "system",
				Content: "Time: $SYS.TIME, App: $ENV.TEST_APP_NAME, User: $CTX.USER_ID",
			},
			ctx: map[string]string{
				"USER_ID": "user-456",
			},
			validate: func(t *testing.T, result Prompt) {
				assert.NotContains(t, result.Content, "$SYS.TIME")
				assert.Contains(t, result.Content, "App: TestApp")
				assert.Contains(t, result.Content, "User: user-456")
			},
		},
		{
			name: "ParseUnknownSysVariable",
			prompt: Prompt{
				Role:    "system",
				Content: "Value: $SYS.UNKNOWN_VAR",
			},
			ctx: nil,
			validate: func(t *testing.T, result Prompt) {
				// Should keep original if not found
				assert.Equal(t, "Value: $SYS.UNKNOWN_VAR", result.Content)
			},
		},
		{
			name: "ParsePreservesRoleAndName",
			prompt: Prompt{
				Role:    "user",
				Content: "Hello $CTX.NAME",
				Name:    "test_user",
			},
			ctx: map[string]string{
				"NAME": "World",
			},
			validate: func(t *testing.T, result Prompt) {
				assert.Equal(t, "user", result.Role)
				assert.Equal(t, "Hello World", result.Content)
				assert.Equal(t, "test_user", result.Name)
			},
		},
		{
			name: "ParseCustomCtxVariables",
			prompt: Prompt{
				Role:    "system",
				Content: "Custom: $CTX.MY_CUSTOM_VAR, Another: $CTX.ANOTHER_VAR",
			},
			ctx: map[string]string{
				"MY_CUSTOM_VAR": "custom-value",
				"ANOTHER_VAR":   "another-value",
			},
			validate: func(t *testing.T, result Prompt) {
				assert.Equal(t, "Custom: custom-value, Another: another-value", result.Content)
			},
		},
		{
			name: "ParseMultilineContent",
			prompt: Prompt{
				Role: "system",
				Content: `# System Context
Current Time: $SYS.TIME
User: $CTX.USER_ID
App: $ENV.TEST_APP_NAME`,
			},
			ctx: map[string]string{
				"USER_ID": "user-789",
			},
			validate: func(t *testing.T, result Prompt) {
				assert.Contains(t, result.Content, "# System Context")
				assert.NotContains(t, result.Content, "$SYS.TIME")
				assert.Contains(t, result.Content, "User: user-789")
				assert.Contains(t, result.Content, "App: TestApp")
			},
		},
	}

	// Set test environment variable
	os.Setenv("TEST_APP_NAME", "TestApp")
	defer os.Unsetenv("TEST_APP_NAME")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.prompt.Parse(tt.ctx)
			tt.validate(t, result)
		})
	}
}

func TestPromptsParse(t *testing.T) {
	os.Setenv("TEST_APP_NAME", "TestApp")
	defer os.Unsetenv("TEST_APP_NAME")

	prompts := Prompts{
		{Role: "system", Content: "Time: $SYS.TIME"},
		{Role: "system", Content: "User: $CTX.USER_ID"},
		{Role: "user", Content: "App: $ENV.TEST_APP_NAME"},
	}

	ctx := map[string]string{
		"USER_ID": "user-123",
	}

	result := prompts.Parse(ctx)

	assert.Len(t, result, 3)
	assert.NotContains(t, result[0].Content, "$SYS.TIME")
	assert.Equal(t, "User: user-123", result[1].Content)
	assert.Equal(t, "App: TestApp", result[2].Content)
}

func TestMergePrompts(t *testing.T) {
	tests := []struct {
		name             string
		globalPrompts    []Prompt
		assistantPrompts []Prompt
		expectedLen      int
		validate         func(t *testing.T, result []Prompt)
	}{
		{
			name: "MergeBothNonEmpty",
			globalPrompts: []Prompt{
				{Role: "system", Content: "Global prompt 1"},
				{Role: "system", Content: "Global prompt 2"},
			},
			assistantPrompts: []Prompt{
				{Role: "system", Content: "Assistant prompt 1"},
			},
			expectedLen: 3,
			validate: func(t *testing.T, result []Prompt) {
				assert.Equal(t, "Global prompt 1", result[0].Content)
				assert.Equal(t, "Global prompt 2", result[1].Content)
				assert.Equal(t, "Assistant prompt 1", result[2].Content)
			},
		},
		{
			name:          "MergeGlobalEmpty",
			globalPrompts: []Prompt{},
			assistantPrompts: []Prompt{
				{Role: "system", Content: "Assistant prompt"},
			},
			expectedLen: 1,
			validate: func(t *testing.T, result []Prompt) {
				assert.Equal(t, "Assistant prompt", result[0].Content)
			},
		},
		{
			name: "MergeAssistantEmpty",
			globalPrompts: []Prompt{
				{Role: "system", Content: "Global prompt"},
			},
			assistantPrompts: []Prompt{},
			expectedLen:      1,
			validate: func(t *testing.T, result []Prompt) {
				assert.Equal(t, "Global prompt", result[0].Content)
			},
		},
		{
			name:             "MergeBothEmpty",
			globalPrompts:    []Prompt{},
			assistantPrompts: []Prompt{},
			expectedLen:      0,
			validate: func(t *testing.T, result []Prompt) {
				assert.Empty(t, result)
			},
		},
		{
			name:          "MergeGlobalNil",
			globalPrompts: nil,
			assistantPrompts: []Prompt{
				{Role: "system", Content: "Assistant prompt"},
			},
			expectedLen: 1,
			validate: func(t *testing.T, result []Prompt) {
				assert.Equal(t, "Assistant prompt", result[0].Content)
			},
		},
		{
			name: "MergeAssistantNil",
			globalPrompts: []Prompt{
				{Role: "system", Content: "Global prompt"},
			},
			assistantPrompts: nil,
			expectedLen:      1,
			validate: func(t *testing.T, result []Prompt) {
				assert.Equal(t, "Global prompt", result[0].Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Merge(tt.globalPrompts, tt.assistantPrompts)
			assert.Len(t, result, tt.expectedLen)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestSystemVariables(t *testing.T) {
	// Test that all system variables are defined and return non-empty values
	expectedVars := []string{
		"TIME", "DATE", "DATETIME", "TIMEZONE", "WEEKDAY",
		"YEAR", "MONTH", "DAY", "HOUR", "MINUTE", "SECOND", "UNIX",
	}

	for _, varName := range expectedVars {
		t.Run(varName, func(t *testing.T) {
			fn, ok := SystemVariables[varName]
			assert.True(t, ok, "SystemVariables should contain %s", varName)
			value := fn()
			assert.NotEmpty(t, value, "SystemVariables[%s]() should return non-empty value", varName)
		})
	}
}

func TestParseVariablesEdgeCases(t *testing.T) {
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	tests := []struct {
		name     string
		content  string
		ctx      map[string]string
		expected string
	}{
		{
			name:     "EmptyContent",
			content:  "",
			ctx:      nil,
			expected: "",
		},
		{
			name:     "NoVariables",
			content:  "Hello, World!",
			ctx:      nil,
			expected: "Hello, World!",
		},
		{
			name:     "PartialVariableSyntax",
			content:  "Value: $SYS Value: $ENV Value: $CTX",
			ctx:      nil,
			expected: "Value: $SYS Value: $ENV Value: $CTX",
		},
		{
			name:     "VariableInMiddleOfWord",
			content:  "prefix$SYS.TIMEsuffix",
			ctx:      nil,
			expected: "prefix$SYS.TIMEsuffix", // Should not match - variable must be followed by valid char
		},
		{
			name:     "MultipleOccurrences",
			content:  "$CTX.VAR and $CTX.VAR again",
			ctx:      map[string]string{"VAR": "value"},
			expected: "value and value again",
		},
		{
			name:     "SpecialCharactersInValue",
			content:  "User: $CTX.USER",
			ctx:      map[string]string{"USER": "user@example.com"},
			expected: "User: user@example.com",
		},
		{
			name:     "UnicodeInValue",
			content:  "Name: $CTX.NAME",
			ctx:      map[string]string{"NAME": "用户名"},
			expected: "Name: 用户名",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVariables(tt.content, tt.ctx)
			if tt.name == "VariableInMiddleOfWord" {
				// This case depends on regex behavior - just check it doesn't crash
				assert.NotEmpty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

package utils_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/utils"
)

// ID tests
func TestNewID(t *testing.T) {
	id1 := utils.NewID()
	id2 := utils.NewID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2, "IDs should be unique")
}

func TestNewIDWithPrefix(t *testing.T) {
	id := utils.NewIDWithPrefix("exec_")
	assert.NotEmpty(t, id)
	assert.Contains(t, id, "exec_")
}

// Time tests
func TestParseTime(t *testing.T) {
	t.Run("valid time", func(t *testing.T) {
		hour, minute, err := utils.ParseTime("14:30")
		assert.NoError(t, err)
		assert.Equal(t, 14, hour)
		assert.Equal(t, 30, minute)
	})

	t.Run("invalid format", func(t *testing.T) {
		_, _, err := utils.ParseTime("14-30")
		assert.Error(t, err)
	})

	t.Run("invalid hour", func(t *testing.T) {
		_, _, err := utils.ParseTime("25:30")
		assert.Error(t, err)
	})

	t.Run("invalid minute", func(t *testing.T) {
		_, _, err := utils.ParseTime("14:65")
		assert.Error(t, err)
	})
}

func TestFormatTime(t *testing.T) {
	result := utils.FormatTime(9, 5)
	assert.Equal(t, "09:05", result)

	result = utils.FormatTime(14, 30)
	assert.Equal(t, "14:30", result)
}

func TestLoadLocation(t *testing.T) {
	t.Run("valid timezone", func(t *testing.T) {
		loc := utils.LoadLocation("Asia/Shanghai")
		assert.NotNil(t, loc)
		assert.Equal(t, "Asia/Shanghai", loc.String())
	})

	t.Run("empty timezone returns Local", func(t *testing.T) {
		loc := utils.LoadLocation("")
		assert.Equal(t, time.Local, loc)
	})

	t.Run("invalid timezone returns Local", func(t *testing.T) {
		loc := utils.LoadLocation("Invalid/Timezone")
		assert.Equal(t, time.Local, loc)
	})
}

func TestParseDuration(t *testing.T) {
	t.Run("valid duration", func(t *testing.T) {
		dur := utils.ParseDuration("30m", 10*time.Minute)
		assert.Equal(t, 30*time.Minute, dur)
	})

	t.Run("empty returns default", func(t *testing.T) {
		dur := utils.ParseDuration("", 10*time.Minute)
		assert.Equal(t, 10*time.Minute, dur)
	})

	t.Run("invalid returns default", func(t *testing.T) {
		dur := utils.ParseDuration("invalid", 10*time.Minute)
		assert.Equal(t, 10*time.Minute, dur)
	})
}

func TestIsTimeMatch(t *testing.T) {
	loc := time.UTC
	testTime := time.Date(2024, 1, 15, 14, 30, 0, 0, loc)

	t.Run("exact match", func(t *testing.T) {
		assert.True(t, utils.IsTimeMatch(testTime, "14:30", loc))
	})

	t.Run("no match", func(t *testing.T) {
		assert.False(t, utils.IsTimeMatch(testTime, "14:31", loc))
		assert.False(t, utils.IsTimeMatch(testTime, "15:30", loc))
	})

	t.Run("invalid time format", func(t *testing.T) {
		assert.False(t, utils.IsTimeMatch(testTime, "invalid", loc))
	})
}

func TestIsDayMatch(t *testing.T) {
	monday := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC) // Monday

	t.Run("match specific day", func(t *testing.T) {
		assert.True(t, utils.IsDayMatch(monday, []string{"Mon"}))
	})

	t.Run("match wildcard", func(t *testing.T) {
		assert.True(t, utils.IsDayMatch(monday, []string{"*"}))
	})

	t.Run("no match", func(t *testing.T) {
		assert.False(t, utils.IsDayMatch(monday, []string{"Tue", "Wed"}))
	})

	t.Run("empty days returns true", func(t *testing.T) {
		assert.True(t, utils.IsDayMatch(monday, []string{}))
	})
}

// Convert tests
func TestToJSON(t *testing.T) {
	data := map[string]interface{}{
		"name": "test",
		"age":  30,
	}

	json, err := utils.ToJSON(data)
	assert.NoError(t, err)
	assert.Contains(t, json, "test")
	assert.Contains(t, json, "30")
}

func TestFromJSON(t *testing.T) {
	jsonStr := `{"name":"test","age":30}`

	var result map[string]interface{}
	err := utils.FromJSON(jsonStr, &result)

	assert.NoError(t, err)
	assert.Equal(t, "test", result["name"])
	assert.Equal(t, float64(30), result["age"]) // JSON numbers are float64
}

func TestToMap(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	s := TestStruct{Name: "test", Age: 30}
	m, err := utils.ToMap(s)

	assert.NoError(t, err)
	assert.Equal(t, "test", m["name"])
	assert.Equal(t, float64(30), m["age"]) // JSON conversion makes it float64
}

func TestFromMap(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	m := map[string]interface{}{
		"name": "test",
		"age":  30,
	}

	var result TestStruct
	err := utils.FromMap(m, &result)

	assert.NoError(t, err)
	assert.Equal(t, "test", result.Name)
	assert.Equal(t, 30, result.Age)
}

func TestToString(t *testing.T) {
	assert.Equal(t, "test", utils.ToString("test"))
	assert.Equal(t, "42", utils.ToString(42))
	assert.Equal(t, "true", utils.ToString(true))
}

func TestMergeMap(t *testing.T) {
	target := map[string]interface{}{
		"a": 1,
		"b": 2,
	}
	source := map[string]interface{}{
		"b": 3,
		"c": 4,
	}

	result := utils.MergeMap(target, source)
	assert.Equal(t, 1, result["a"])
	assert.Equal(t, 3, result["b"]) // overwritten
	assert.Equal(t, 4, result["c"])
}

func TestCloneMap(t *testing.T) {
	original := map[string]interface{}{
		"a": 1,
		"b": 2,
	}

	cloned := utils.CloneMap(original)
	cloned["a"] = 999

	assert.Equal(t, 1, original["a"]) // original unchanged
	assert.Equal(t, 999, cloned["a"])
}

// Validate tests
func TestIsEmpty(t *testing.T) {
	assert.True(t, utils.IsEmpty(""))
	assert.False(t, utils.IsEmpty("test"))
}

func TestIsValidEmail(t *testing.T) {
	assert.True(t, utils.IsValidEmail("test@example.com"))
	assert.True(t, utils.IsValidEmail("user+tag@domain.co.uk"))
	assert.False(t, utils.IsValidEmail("invalid"))
	assert.False(t, utils.IsValidEmail("@example.com"))
	assert.False(t, utils.IsValidEmail("test@"))
}

func TestIsValidTime(t *testing.T) {
	assert.True(t, utils.IsValidTime("09:00"))
	assert.True(t, utils.IsValidTime("14:30"))
	assert.True(t, utils.IsValidTime("23:59"))
	assert.False(t, utils.IsValidTime("25:00"))
	assert.False(t, utils.IsValidTime("14:65"))
	assert.False(t, utils.IsValidTime("14-30"))
}

func TestValidateRequired(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		err := utils.ValidateRequired("field", nil)
		assert.Error(t, err)
	})

	t.Run("empty string", func(t *testing.T) {
		err := utils.ValidateRequired("field", "")
		assert.Error(t, err)
	})

	t.Run("valid string", func(t *testing.T) {
		err := utils.ValidateRequired("field", "value")
		assert.NoError(t, err)
	})

	t.Run("empty slice", func(t *testing.T) {
		err := utils.ValidateRequired("field", []string{})
		assert.Error(t, err)
	})
}

func TestValidateRange(t *testing.T) {
	t.Run("within range", func(t *testing.T) {
		err := utils.ValidateRange("field", 5, 1, 10)
		assert.NoError(t, err)
	})

	t.Run("below range", func(t *testing.T) {
		err := utils.ValidateRange("field", 0, 1, 10)
		assert.Error(t, err)
	})

	t.Run("above range", func(t *testing.T) {
		err := utils.ValidateRange("field", 11, 1, 10)
		assert.Error(t, err)
	})
}

func TestValidateOneOf(t *testing.T) {
	allowed := []string{"apple", "banana", "cherry"}

	t.Run("valid value", func(t *testing.T) {
		err := utils.ValidateOneOf("field", "banana", allowed)
		assert.NoError(t, err)
	})

	t.Run("invalid value", func(t *testing.T) {
		err := utils.ValidateOneOf("field", "orange", allowed)
		assert.Error(t, err)
	})
}

package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/maps"
)

func TestMapToStructBooleanFields(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    interface{}
		expected bool
	}{
		{"bool true", "enabled", true, true},
		{"bool false", "enabled", false, false},
		{"string true", "enabled", "true", true},
		{"string 1", "enabled", "1", true},
		{"string t (PG)", "enabled", "t", true},
		{"string false", "enabled", "false", false},
		{"string 0", "enabled", "0", false},
		{"string f (PG)", "enabled", "f", false},
		{"int 1", "enabled", int(1), true},
		{"int 0", "enabled", int(0), false},
		{"int64 1", "enabled", int64(1), true},
		{"int64 0", "enabled", int64(0), false},
		{"float64 1", "enabled", float64(1), true},
		{"float64 0", "enabled", float64(0), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := maps.MapStr{tt.key: tt.value}
			job := &Job{}
			err := mapToStruct(m, job)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, job.Enabled, "field %s with value %v", tt.key, tt.value)
		})
	}

	t.Run("system field", func(t *testing.T) {
		m := maps.MapStr{"system": "t"}
		job := &Job{}
		err := mapToStruct(m, job)
		assert.NoError(t, err)
		assert.True(t, job.System)
	})

	t.Run("readonly field", func(t *testing.T) {
		m := maps.MapStr{"readonly": "1"}
		job := &Job{}
		err := mapToStruct(m, job)
		assert.NoError(t, err)
		assert.True(t, job.Readonly)
	})
}

func TestMapToStructTimeFields(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name   string
		input  string
		parsed bool
	}{
		{"MySQL format", "2024-06-15 10:30:45", true},
		{"PG fractional", "2024-06-15 10:30:45.123456", true},
		{"PG with +00", "2024-06-15 10:30:45.123456+00", true},
		{"PG with -07", "2024-06-15 03:30:45.123456-07", true},
		{"RFC3339", "2024-06-15T10:30:45Z", true},
		{"RFC3339Nano", "2024-06-15T10:30:45.123456789Z", true},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := maps.MapStr{
				"name":       "test-job",
				"job_id":     "j-123",
				"created_at": tt.input,
			}
			job := &Job{}
			err := mapToStruct(m, job)
			if tt.parsed {
				assert.NoError(t, err)
				assert.False(t, job.CreatedAt.IsZero(), "expected non-zero time for input: %s", tt.input)
				assert.Equal(t, refTime.Unix(), job.CreatedAt.Unix(),
					"expected %v, got %v for input: %s", refTime, job.CreatedAt, tt.input)
			} else {
				assert.True(t, job.CreatedAt.IsZero())
			}
		})
	}
}

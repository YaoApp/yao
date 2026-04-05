package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDBTimestampToUnix(t *testing.T) {
	ref := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)
	refUnix := ref.Unix()

	tests := []struct {
		name   string
		input  interface{}
		expect interface{}
	}{
		{"nil", nil, nil},
		{"empty string", "", nil},
		{"MySQL DATETIME", "2024-06-15 10:30:45", refUnix},
		{"PG with fractional seconds", "2024-06-15 10:30:45.123456", refUnix},
		{"PG with +00 offset", "2024-06-15 10:30:45.123456+00", refUnix},
		{"PG with -07 offset", "2024-06-15 03:30:45.123456-07", refUnix},
		{"ISO Z suffix", "2024-06-15T10:30:45Z", refUnix},
		{"RFC3339", "2024-06-15T10:30:45+00:00", refUnix},
		{"RFC3339Nano", "2024-06-15T10:30:45.123456789+00:00", refUnix},
		{"*string nil", (*string)(nil), nil},
		{"*string valid", strPtr("2024-06-15 10:30:45"), refUnix},
		{"*string empty", strPtr(""), nil},
		{"non-string type", 12345, nil},
		{"unparseable string", "not-a-date", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DBTimestampToUnix(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestUnixToDBTimestamp(t *testing.T) {
	refUnix := int64(1718444445)
	expected := time.Unix(refUnix, 0).UTC().Format("2006-01-02 15:04:05")

	tests := []struct {
		name   string
		input  interface{}
		expect interface{}
	}{
		{"nil", nil, nil},
		{"int64", refUnix, expected},
		{"*int64 valid", int64Ptr(refUnix), expected},
		{"*int64 nil", (*int64)(nil), nil},
		{"int", int(refUnix), expected},
		{"float64", float64(refUnix), expected},
		{"unknown type", "not-a-number", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnixToDBTimestamp(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestToUnixTimestamp(t *testing.T) {
	refUnix := int64(1718444445)

	tests := []struct {
		name   string
		input  interface{}
		expect interface{}
	}{
		{"nil", nil, nil},
		{"int64", refUnix, refUnix},
		{"*int64 valid", int64Ptr(refUnix), refUnix},
		{"*int64 nil", (*int64)(nil), nil},
		{"int", int(refUnix), refUnix},
		{"float64", float64(refUnix), refUnix},
		{"string MySQL", "2024-06-15 10:00:45", time.Date(2024, 6, 15, 10, 0, 45, 0, time.UTC).Unix()},
		{"string PG fractional", "2024-06-15 10:00:45.123456", time.Date(2024, 6, 15, 10, 0, 45, 0, time.UTC).Unix()},
		{"string RFC3339", "2024-06-15T10:00:45Z", time.Date(2024, 6, 15, 10, 0, 45, 0, time.UTC).Unix()},
		{"*string valid", strPtr("2024-06-15 10:00:45"), time.Date(2024, 6, 15, 10, 0, 45, 0, time.UTC).Unix()},
		{"*string nil", (*string)(nil), nil},
		{"string empty", "", nil},
		{"unknown type", struct{}{}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToUnixTimestamp(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestToTimeStringWithPGFormats(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		expect string
	}{
		{"PG fractional", "2024-06-15 10:30:45.123456", "2024-06-15T10:30:45Z"},
		{"PG +00 offset", "2024-06-15 10:30:45.123456+00", "2024-06-15T10:30:45Z"},
		{"PG -07 offset", "2024-06-15 03:30:45.123456-07", "2024-06-15T03:30:45-07:00"},
		{"MySQL format", "2024-06-15 10:30:45", "2024-06-15T10:30:45Z"},
		{"RFC3339Nano", "2024-06-15T10:30:45.123456789+00:00", "2024-06-15T10:30:45Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToTimeString(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func strPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64 { return &i }

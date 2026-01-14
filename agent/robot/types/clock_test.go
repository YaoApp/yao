package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/types"
)

func TestNewClockContext(t *testing.T) {
	t.Run("basic clock context", func(t *testing.T) {
		// Test with a known date: 2024-01-15 14:30:00 (Monday)
		testTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
		ctx := types.NewClockContext(testTime, "UTC")

		assert.Equal(t, 14, ctx.Hour)
		assert.Equal(t, "Monday", ctx.DayOfWeek)
		assert.Equal(t, 15, ctx.DayOfMonth)
		assert.Equal(t, 1, ctx.Month)
		assert.Equal(t, 2024, ctx.Year)
		assert.False(t, ctx.IsWeekend)
		assert.False(t, ctx.IsMonthStart)
		assert.False(t, ctx.IsMonthEnd)
		assert.False(t, ctx.IsQuarterEnd)
		assert.False(t, ctx.IsYearEnd)
	})

	t.Run("weekend detection", func(t *testing.T) {
		// Saturday
		saturday := time.Date(2024, 1, 13, 10, 0, 0, 0, time.UTC)
		ctx := types.NewClockContext(saturday, "")
		assert.True(t, ctx.IsWeekend)

		// Sunday
		sunday := time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)
		ctx = types.NewClockContext(sunday, "")
		assert.True(t, ctx.IsWeekend)
	})

	t.Run("month start detection", func(t *testing.T) {
		// 1st day
		day1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		ctx := types.NewClockContext(day1, "")
		assert.True(t, ctx.IsMonthStart)

		// 3rd day
		day3 := time.Date(2024, 1, 3, 10, 0, 0, 0, time.UTC)
		ctx = types.NewClockContext(day3, "")
		assert.True(t, ctx.IsMonthStart)

		// 4th day - not month start
		day4 := time.Date(2024, 1, 4, 10, 0, 0, 0, time.UTC)
		ctx = types.NewClockContext(day4, "")
		assert.False(t, ctx.IsMonthStart)
	})

	t.Run("month end detection", func(t *testing.T) {
		// Last day of January (31st)
		lastDay := time.Date(2024, 1, 31, 10, 0, 0, 0, time.UTC)
		ctx := types.NewClockContext(lastDay, "")
		assert.True(t, ctx.IsMonthEnd)

		// 29th day of January (31 days total)
		day29 := time.Date(2024, 1, 29, 10, 0, 0, 0, time.UTC)
		ctx = types.NewClockContext(day29, "")
		assert.True(t, ctx.IsMonthEnd)

		// 28th day of January - not month end
		day28 := time.Date(2024, 1, 28, 10, 0, 0, 0, time.UTC)
		ctx = types.NewClockContext(day28, "")
		assert.False(t, ctx.IsMonthEnd)
	})

	t.Run("quarter end detection", func(t *testing.T) {
		// March 31 - Q1 end
		q1End := time.Date(2024, 3, 31, 10, 0, 0, 0, time.UTC)
		ctx := types.NewClockContext(q1End, "")
		assert.True(t, ctx.IsQuarterEnd)

		// June 30 - Q2 end
		q2End := time.Date(2024, 6, 30, 10, 0, 0, 0, time.UTC)
		ctx = types.NewClockContext(q2End, "")
		assert.True(t, ctx.IsQuarterEnd)

		// September 30 - Q3 end
		q3End := time.Date(2024, 9, 30, 10, 0, 0, 0, time.UTC)
		ctx = types.NewClockContext(q3End, "")
		assert.True(t, ctx.IsQuarterEnd)

		// December 31 - Q4 end
		q4End := time.Date(2024, 12, 31, 10, 0, 0, 0, time.UTC)
		ctx = types.NewClockContext(q4End, "")
		assert.True(t, ctx.IsQuarterEnd)

		// Not quarter end
		notQEnd := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		ctx = types.NewClockContext(notQEnd, "")
		assert.False(t, ctx.IsQuarterEnd)
	})

	t.Run("year end detection", func(t *testing.T) {
		// December 29
		dec29 := time.Date(2024, 12, 29, 10, 0, 0, 0, time.UTC)
		ctx := types.NewClockContext(dec29, "")
		assert.True(t, ctx.IsYearEnd)

		// December 31
		dec31 := time.Date(2024, 12, 31, 10, 0, 0, 0, time.UTC)
		ctx = types.NewClockContext(dec31, "")
		assert.True(t, ctx.IsYearEnd)

		// December 28 - not year end
		dec28 := time.Date(2024, 12, 28, 10, 0, 0, 0, time.UTC)
		ctx = types.NewClockContext(dec28, "")
		assert.False(t, ctx.IsYearEnd)

		// January - not year end
		jan1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		ctx = types.NewClockContext(jan1, "")
		assert.False(t, ctx.IsYearEnd)
	})

	t.Run("timezone handling", func(t *testing.T) {
		testTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

		// With Asia/Shanghai timezone
		ctx := types.NewClockContext(testTime, "Asia/Shanghai")
		assert.Equal(t, "Asia/Shanghai", ctx.TZ)
		// Time should be converted to Shanghai timezone
		assert.NotEqual(t, testTime, ctx.Now)
		assert.Equal(t, 22, ctx.Hour) // UTC 14:00 = Shanghai 22:00 (UTC+8)

		// With invalid timezone - should fall back to local
		ctx = types.NewClockContext(testTime, "Invalid/Timezone")
		assert.NotEmpty(t, ctx.TZ)
	})

	t.Run("week of year", func(t *testing.T) {
		// First week of 2024
		jan1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		ctx := types.NewClockContext(jan1, "")
		assert.Equal(t, 1, ctx.WeekOfYear)

		// Mid year
		july15 := time.Date(2024, 7, 15, 10, 0, 0, 0, time.UTC)
		ctx = types.NewClockContext(july15, "")
		assert.Greater(t, ctx.WeekOfYear, 20)
		assert.Less(t, ctx.WeekOfYear, 35)
	})
}

func TestClockContextFields(t *testing.T) {
	// Test all fields are populated correctly
	testTime := time.Date(2024, 12, 30, 23, 45, 30, 0, time.UTC)
	ctx := types.NewClockContext(testTime, "UTC")

	assert.NotZero(t, ctx.Now)
	assert.Equal(t, 23, ctx.Hour)
	assert.Equal(t, "Monday", ctx.DayOfWeek)
	assert.Equal(t, 30, ctx.DayOfMonth)
	assert.Equal(t, 1, ctx.WeekOfYear) // Dec 30, 2024 is week 1 of 2025
	assert.Equal(t, 12, ctx.Month)
	assert.Equal(t, 2024, ctx.Year)
	assert.False(t, ctx.IsWeekend) // Monday
	assert.False(t, ctx.IsMonthStart)
	assert.True(t, ctx.IsMonthEnd)
	assert.True(t, ctx.IsQuarterEnd)
	assert.True(t, ctx.IsYearEnd)
	assert.Equal(t, "UTC", ctx.TZ)
}

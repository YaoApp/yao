package types

import "time"

// ClockContext - time context for P0 inspiration
type ClockContext struct {
	Now          time.Time `json:"now"`
	Hour         int       `json:"hour"`         // 0-23
	DayOfWeek    string    `json:"day_of_week"`  // Monday, Tuesday...
	DayOfMonth   int       `json:"day_of_month"` // 1-31
	WeekOfYear   int       `json:"week_of_year"` // 1-52
	Month        int       `json:"month"`        // 1-12
	Year         int       `json:"year"`
	IsWeekend    bool      `json:"is_weekend"`
	IsMonthStart bool      `json:"is_month_start"` // 1st-3rd
	IsMonthEnd   bool      `json:"is_month_end"`   // last 3 days
	IsQuarterEnd bool      `json:"is_quarter_end"`
	IsYearEnd    bool      `json:"is_year_end"`
	TZ           string    `json:"tz"`
}

// NewClockContext creates clock context from time
func NewClockContext(t time.Time, tz string) *ClockContext {
	loc := time.Local
	if tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}
	t = t.In(loc)

	_, week := t.ISOWeek()
	dayOfMonth := t.Day()
	lastDay := time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, loc).Day()

	return &ClockContext{
		Now:          t,
		Hour:         t.Hour(),
		DayOfWeek:    t.Weekday().String(),
		DayOfMonth:   dayOfMonth,
		WeekOfYear:   week,
		Month:        int(t.Month()),
		Year:         t.Year(),
		IsWeekend:    t.Weekday() == time.Saturday || t.Weekday() == time.Sunday,
		IsMonthStart: dayOfMonth <= 3,
		IsMonthEnd:   dayOfMonth >= lastDay-2,
		IsQuarterEnd: (t.Month()%3 == 0) && dayOfMonth >= lastDay-2,
		IsYearEnd:    t.Month() == 12 && dayOfMonth >= 29,
		TZ:           loc.String(),
	}
}

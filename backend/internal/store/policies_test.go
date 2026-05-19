package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTruncateToPeriod(t *testing.T) {
	loc := time.UTC

	tests := []struct {
		name   string
		input  time.Time
		period Period
		want   time.Time
	}{
		// Hour truncation
		{
			name:   "hour: mid-minute truncates to start of hour",
			input:  time.Date(2025, 6, 15, 14, 37, 42, 123456789, loc),
			period: PeriodHour,
			want:   time.Date(2025, 6, 15, 14, 0, 0, 0, loc),
		},
		{
			name:   "hour: already at start of hour stays unchanged",
			input:  time.Date(2025, 6, 15, 14, 0, 0, 0, loc),
			period: PeriodHour,
			want:   time.Date(2025, 6, 15, 14, 0, 0, 0, loc),
		},
		{
			name:   "hour: midnight stays midnight",
			input:  time.Date(2025, 6, 15, 0, 0, 0, 0, loc),
			period: PeriodHour,
			want:   time.Date(2025, 6, 15, 0, 0, 0, 0, loc),
		},
		{
			name:   "hour: last second of hour truncates to start",
			input:  time.Date(2025, 6, 15, 23, 59, 59, 999999999, loc),
			period: PeriodHour,
			want:   time.Date(2025, 6, 15, 23, 0, 0, 0, loc),
		},

		// Day truncation
		{
			name:   "day: mid-day truncates to midnight",
			input:  time.Date(2025, 6, 15, 14, 37, 42, 123456789, loc),
			period: PeriodDay,
			want:   time.Date(2025, 6, 15, 0, 0, 0, 0, loc),
		},
		{
			name:   "day: already at midnight stays unchanged",
			input:  time.Date(2025, 6, 15, 0, 0, 0, 0, loc),
			period: PeriodDay,
			want:   time.Date(2025, 6, 15, 0, 0, 0, 0, loc),
		},
		{
			name:   "day: new year boundary",
			input:  time.Date(2025, 1, 1, 3, 45, 0, 0, loc),
			period: PeriodDay,
			want:   time.Date(2025, 1, 1, 0, 0, 0, 0, loc),
		},
		{
			name:   "day: last moment of year",
			input:  time.Date(2025, 12, 31, 23, 59, 59, 999999999, loc),
			period: PeriodDay,
			want:   time.Date(2025, 12, 31, 0, 0, 0, 0, loc),
		},

		// Month truncation
		{
			name:   "month: mid-month truncates to first of month",
			input:  time.Date(2025, 6, 15, 14, 37, 42, 0, loc),
			period: PeriodMonth,
			want:   time.Date(2025, 6, 1, 0, 0, 0, 0, loc),
		},
		{
			name:   "month: already first of month at midnight stays unchanged",
			input:  time.Date(2025, 6, 1, 0, 0, 0, 0, loc),
			period: PeriodMonth,
			want:   time.Date(2025, 6, 1, 0, 0, 0, 0, loc),
		},
		{
			name:   "month: last day of month truncates to first",
			input:  time.Date(2025, 1, 31, 23, 59, 59, 0, loc),
			period: PeriodMonth,
			want:   time.Date(2025, 1, 1, 0, 0, 0, 0, loc),
		},
		{
			name:   "month: february leap year",
			input:  time.Date(2024, 2, 29, 12, 0, 0, 0, loc),
			period: PeriodMonth,
			want:   time.Date(2024, 2, 1, 0, 0, 0, 0, loc),
		},
		{
			name:   "month: january 1 (new year boundary)",
			input:  time.Date(2025, 1, 1, 0, 0, 0, 0, loc),
			period: PeriodMonth,
			want:   time.Date(2025, 1, 1, 0, 0, 0, 0, loc),
		},

		// Year / unknown period returns input unchanged
		{
			name:   "year: returns input unchanged",
			input:  time.Date(2025, 6, 15, 14, 37, 42, 123456789, loc),
			period: PeriodYear,
			want:   time.Date(2025, 6, 15, 14, 37, 42, 123456789, loc),
		},
		{
			name:   "unknown period: returns input unchanged",
			input:  time.Date(2025, 6, 15, 14, 37, 42, 0, loc),
			period: Period("week"),
			want:   time.Date(2025, 6, 15, 14, 37, 42, 0, loc),
		},
		{
			name:   "empty period: returns input unchanged",
			input:  time.Date(2025, 6, 15, 14, 37, 42, 0, loc),
			period: Period(""),
			want:   time.Date(2025, 6, 15, 14, 37, 42, 0, loc),
		},

		// Non-UTC location
		{
			name:   "day: respects non-UTC location",
			input:  time.Date(2025, 6, 15, 14, 37, 42, 0, time.FixedZone("EST", -5*3600)),
			period: PeriodDay,
			want:   time.Date(2025, 6, 15, 0, 0, 0, 0, time.FixedZone("EST", -5*3600)),
		},
		{
			name:   "month: respects non-UTC location",
			input:  time.Date(2025, 6, 15, 14, 37, 42, 0, time.FixedZone("EST", -5*3600)),
			period: PeriodMonth,
			want:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.FixedZone("EST", -5*3600)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateToPeriod(tt.input, tt.period)
			assert.True(t, tt.want.Equal(got), "truncateToPeriod(%v, %q) = %v, want %v", tt.input, tt.period, got, tt.want)
		})
	}
}

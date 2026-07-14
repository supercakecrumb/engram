package engram

import (
	"reflect"
	"testing"
	"time"
)

func TestAccuracy(t *testing.T) {
	tests := []struct {
		name string
		log  []Review
		want float64
	}{
		{"empty log", nil, 0},
		{"three of four correct", []Review{
			{Rating: Again}, {Rating: Good}, {Rating: Hard}, {Rating: Easy},
		}, 0.75},
		{"all again", []Review{{Rating: Again}, {Rating: Again}}, 0},
		{"all correct", []Review{{Rating: Good}, {Rating: Easy}, {Rating: Hard}}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Accuracy(tt.log); got != tt.want {
				t.Errorf("Accuracy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStreak(t *testing.T) {
	nyLoc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("America/New_York tzdata unavailable: %v", err)
	}

	// mkLog builds one review per given day-of-January-2026, at local noon
	// (to stay well clear of any day-boundary ambiguity).
	mkLog := func(loc *time.Location, days ...int) []Review {
		log := make([]Review, 0, len(days))
		for _, d := range days {
			log = append(log, Review{ReviewedAt: time.Date(2026, 1, d, 12, 0, 0, 0, loc)})
		}
		return log
	}

	tests := []struct {
		name string
		loc  *time.Location
		log  []Review
		now  time.Time
		want int
	}{
		{
			name: "today plus several consecutive prior days",
			loc:  time.UTC,
			log:  mkLog(time.UTC, 5, 6, 7),
			now:  time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC),
			want: 3,
		},
		{
			name: "a gap breaks the streak",
			loc:  time.UTC,
			log:  mkLog(time.UTC, 3, 5, 6, 7), // day 4 missing
			now:  time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC),
			want: 3, // 7,6,5 count; day 3 is isolated by the gap on day 4
		},
		{
			name: "today has no review but yesterday and prior do",
			loc:  nyLoc,
			log:  mkLog(nyLoc, 5, 6),
			now:  time.Date(2026, 1, 7, 12, 0, 0, 0, nyLoc),
			want: 2, // counts from yesterday (6) back through 5
		},
		{
			name: "neither today nor yesterday has a review",
			loc:  time.UTC,
			log:  mkLog(time.UTC, 1, 2, 3),
			now:  time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC),
			want: 0,
		},
		{
			name: "empty log",
			loc:  time.UTC,
			log:  nil,
			now:  time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC),
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Streak(tt.log, tt.now, tt.loc); got != tt.want {
				t.Errorf("Streak() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDueForecast(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cards := []CardState{
		{Due: now.Add(-72 * time.Hour)},                     // well overdue -> bucket 0
		{Due: time.Date(2026, 1, 1, 23, 0, 0, 0, time.UTC)}, // due later today -> bucket 0
		{Due: time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC)},  // +1 day -> bucket 1
		{Due: time.Date(2026, 1, 3, 9, 0, 0, 0, time.UTC)},  // +2 days -> bucket 2
		{Due: time.Date(2026, 1, 4, 9, 0, 0, 0, time.UTC)},  // +3 days -> beyond a 3-bucket window, dropped
		{Due: time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)}, // far beyond -> dropped
	}

	got := DueForecast(cards, 3, now, time.UTC)
	want := []int{2, 1, 1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DueForecast() = %v, want %v", got, want)
	}
}

func TestDueForecastNonPositiveDays(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cards := []CardState{{Due: now}}

	for _, days := range []int{0, -1, -10} {
		got := DueForecast(cards, days, now, time.UTC)
		if got == nil {
			t.Errorf("days=%d: DueForecast() = nil, want non-nil empty slice", days)
		}
		if len(got) != 0 {
			t.Errorf("days=%d: DueForecast() = %v, want empty slice", days, got)
		}
	}
}

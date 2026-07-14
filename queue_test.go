package engram

import (
	"testing"
	"time"
)

var queueNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestNextDueDueFirstOrdering(t *testing.T) {
	items := []QueueItem{
		{Skill: Skill{ID: "new-card"}, Card: CardState{State: StateNew, Due: queueNow.Add(-time.Hour)}},
		{Skill: Skill{ID: "later"}, Card: CardState{State: StateReview, Due: queueNow.Add(-30 * time.Minute)}},
		{Skill: Skill{ID: "earliest"}, Card: CardState{State: StateReview, Due: queueNow.Add(-2 * time.Hour)}},
		{Skill: Skill{ID: "not-yet-due"}, Card: CardState{State: StateReview, Due: queueNow.Add(time.Hour)}},
	}

	got, ok := NextDue(items, 0, QueueConfig{}, queueNow)
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if got.Skill.ID != "earliest" {
		t.Errorf("Skill.ID = %q, want %q (earliest-due review card, not the new card)", got.Skill.ID, "earliest")
	}
}

func TestNextDueNewCardOnlyWhenNothingDue(t *testing.T) {
	items := []QueueItem{
		{Skill: Skill{ID: "due-review"}, Card: CardState{State: StateReview, Due: queueNow.Add(-time.Hour)}},
		{Skill: Skill{ID: "new-card"}, Card: CardState{State: StateNew, Due: queueNow.Add(-time.Hour)}},
	}

	// Plenty of room under the new-card cap, but a due review card must
	// still win.
	got, ok := NextDue(items, 0, QueueConfig{MaxNewPerDay: 100}, queueNow)
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if got.Skill.ID != "due-review" {
		t.Errorf("Skill.ID = %q, want %q (due review card preferred over new card)", got.Skill.ID, "due-review")
	}
}

func TestNextDueMaxNewPerDay(t *testing.T) {
	items := []QueueItem{
		{Skill: Skill{ID: "n1"}, Card: CardState{State: StateNew, Due: queueNow.Add(-time.Hour)}},
	}

	tests := []struct {
		name               string
		newIntroducedToday int
		cfg                QueueConfig
		wantOK             bool
	}{
		{"under cap", 1, QueueConfig{MaxNewPerDay: 5}, true},
		{"exactly at cap", 5, QueueConfig{MaxNewPerDay: 5}, false},
		{"over cap", 6, QueueConfig{MaxNewPerDay: 5}, false},
		{"zero cap means unlimited", 1000, QueueConfig{MaxNewPerDay: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NextDue(items, tt.newIntroducedToday, tt.cfg, queueNow)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if tt.wantOK && got.Skill.ID != "n1" {
				t.Errorf("Skill.ID = %q, want %q", got.Skill.ID, "n1")
			}
		})
	}
}

func TestNextDueEmptyItems(t *testing.T) {
	got, ok := NextDue(nil, 0, QueueConfig{}, queueNow)
	if ok {
		t.Fatalf("ok = true, want false for empty items")
	}
	if got != (QueueItem{}) {
		t.Errorf("QueueItem = %+v, want zero value", got)
	}
}

// TestCountNewIntroducedDayBoundaryNonUTC exercises the calendar-day
// conversion across a UTC/local-time day boundary: instants are chosen so
// that they land on different calendar days in UTC than in
// America/New_York, and the count must follow the New York day since that's
// the loc passed in.
func TestCountNewIntroducedDayBoundaryNonUTC(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("America/New_York tzdata unavailable: %v", err)
	}

	// now is 2026-01-01 12:00 in New York.
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, loc)

	// 2026-01-02 02:00 UTC = 2026-01-01 21:00 New York -> same NY day as now.
	sameNYDay := time.Date(2026, 1, 2, 2, 0, 0, 0, time.UTC)
	// 2026-01-01 04:00 UTC = 2025-12-31 23:00 New York -> different NY day.
	diffNYDay := time.Date(2026, 1, 1, 4, 0, 0, 0, time.UTC)

	log := []Review{
		{SkillID: "a", StateBefore: StateNew, ReviewedAt: sameNYDay},
		{SkillID: "b", StateBefore: StateNew, ReviewedAt: diffNYDay},
		// Same instant as the included StateNew entry, but StateReview:
		// must never be counted regardless of its day.
		{SkillID: "c", StateBefore: StateReview, ReviewedAt: sameNYDay},
	}

	got := CountNewIntroduced(log, now, loc)
	if got != 1 {
		t.Errorf("CountNewIntroduced = %d, want 1 (only the same-New-York-day StateNew review)", got)
	}
}

func TestCountNewIntroducedNilLocDefaultsToUTC(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	sameUTCDay := time.Date(2026, 1, 1, 23, 0, 0, 0, time.UTC)
	diffUTCDay := time.Date(2026, 1, 2, 1, 0, 0, 0, time.UTC)

	log := []Review{
		{StateBefore: StateNew, ReviewedAt: sameUTCDay},
		{StateBefore: StateNew, ReviewedAt: diffUTCDay},
	}

	got := CountNewIntroduced(log, now, nil)
	if got != 1 {
		t.Errorf("CountNewIntroduced with nil loc = %d, want 1 (defaults to UTC day boundary)", got)
	}
}

func TestCountNewIntroducedIgnoresNonNewState(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	log := []Review{
		{StateBefore: StateReview, ReviewedAt: now},
		{StateBefore: StateLearning, ReviewedAt: now},
		{StateBefore: StateRelearning, ReviewedAt: now},
	}

	if got := CountNewIntroduced(log, now, nil); got != 0 {
		t.Errorf("CountNewIntroduced = %d, want 0 (no StateBefore==StateNew entries)", got)
	}
}

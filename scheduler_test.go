package engram

import (
	"testing"
	"time"
)

// baseNow is a fixed, deterministic reference instant used throughout the
// scheduler tests. Never use time.Now() here: the engine's contract is that
// it never reads the clock itself.
var baseNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestNewCardState(t *testing.T) {
	sched := NewScheduler()
	cs := sched.NewCardState(baseNow)

	if !cs.Due.Equal(baseNow) {
		t.Errorf("Due = %v, want %v", cs.Due, baseNow)
	}
	if cs.State != StateNew {
		t.Errorf("State = %v, want StateNew", cs.State)
	}
	if cs.Reps != 0 {
		t.Errorf("Reps = %d, want 0", cs.Reps)
	}
	if cs.Lapses != 0 {
		t.Errorf("Lapses = %d, want 0", cs.Lapses)
	}
	if !cs.LastReview.IsZero() {
		t.Errorf("LastReview = %v, want zero value", cs.LastReview)
	}
}

// TestGoodGrowsInterval drives a card New -> Learning -> Review with Good,
// advancing now to each returned Due, then does several more Good reviews
// while in Review state. Once in StateReview, each successive Good must
// produce a strictly larger interval than the previous review's interval,
// and Stability must strictly increase every step.
func TestGoodGrowsInterval(t *testing.T) {
	sched := NewScheduler()
	now := baseNow
	cs := sched.NewCardState(now)

	// New -> Learning.
	next, rev := sched.Next(cs, now, Good)
	if next.State != StateLearning {
		t.Fatalf("after first Good: State = %v, want StateLearning", next.State)
	}
	if rev.StateBefore != StateNew {
		t.Fatalf("after first Good: Review.StateBefore = %v, want StateNew", rev.StateBefore)
	}
	cs, now = next, next.Due

	// Learning -> Review.
	next, rev = sched.Next(cs, now, Good)
	if next.State != StateReview {
		t.Fatalf("after second Good: State = %v, want StateReview", next.State)
	}
	if rev.StateBefore != StateLearning {
		t.Fatalf("after second Good: Review.StateBefore = %v, want StateLearning", rev.StateBefore)
	}
	cs, now = next, next.Due

	// Several more Good reviews while in Review state: interval and
	// stability must both grow every step.
	const rounds = 4
	prevInterval := time.Duration(-1)
	for i := 0; i < rounds; i++ {
		next, rev := sched.Next(cs, now, Good)
		if next.State != StateReview {
			t.Fatalf("round %d: State = %v, want StateReview", i, next.State)
		}
		if rev.StateBefore != StateReview {
			t.Fatalf("round %d: Review.StateBefore = %v, want StateReview", i, rev.StateBefore)
		}

		interval := next.Due.Sub(now)
		if !next.Due.After(now) {
			t.Fatalf("round %d: Due %v is not after now %v", i, next.Due, now)
		}
		if next.Stability <= cs.Stability {
			t.Errorf("round %d: Stability did not increase: before=%v after=%v", i, cs.Stability, next.Stability)
		}
		if prevInterval >= 0 && interval <= prevInterval {
			t.Errorf("round %d: interval did not grow: previous=%v current=%v", i, prevInterval, interval)
		}

		prevInterval = interval
		cs, now = next, next.Due
	}
}

// TestAgainResetsFromReview takes a card that has reached StateReview with a
// sizeable interval and checks that an Again rating snaps it back to a
// Relearning card due within minutes, far short of the interval that got it
// there, while incrementing Lapses.
func TestAgainResetsFromReview(t *testing.T) {
	sched := NewScheduler()
	now := baseNow
	cs := sched.NewCardState(now)

	// Walk New -> Learning -> Review, then a couple more Good reviews to
	// build up a large interval.
	cs, _ = sched.Next(cs, now, Good)
	now = cs.Due
	cs, _ = sched.Next(cs, now, Good)
	now = cs.Due
	if cs.State != StateReview {
		t.Fatalf("setup: expected StateReview, got %v", cs.State)
	}

	prevNow := now
	cs, _ = sched.Next(cs, now, Good)
	now = cs.Due
	cs, _ = sched.Next(cs, now, Good)
	now = cs.Due
	bigInterval := now.Sub(prevNow)
	if bigInterval < 24*time.Hour {
		t.Fatalf("setup: expected a large pre-Again interval, got %v", bigInterval)
	}

	lapsesBefore := cs.Lapses
	next, rev := sched.Next(cs, now, Again)

	if next.State != StateRelearning {
		t.Errorf("State = %v, want StateRelearning", next.State)
	}
	if delta := next.Due.Sub(now); delta <= 0 || delta > time.Hour {
		t.Errorf("Due - now = %v, want a small positive duration (<= 1h)", delta)
	}
	if got := next.Due.Sub(now); got >= bigInterval {
		t.Errorf("Again interval %v should be far smaller than the prior scheduled interval %v", got, bigInterval)
	}
	if next.Lapses != lapsesBefore+1 {
		t.Errorf("Lapses = %d, want %d", next.Lapses, lapsesBefore+1)
	}
	if rev.StateBefore != StateReview {
		t.Errorf("Review.StateBefore = %v, want StateReview", rev.StateBefore)
	}
}

// TestReviewStateDueAlwaysAfterNow checks that for a card already in
// StateReview, every possible rating produces a Due strictly after now.
func TestReviewStateDueAlwaysAfterNow(t *testing.T) {
	sched := NewScheduler()
	now := baseNow
	cs := sched.NewCardState(now)
	cs, _ = sched.Next(cs, now, Good)
	now = cs.Due
	cs, _ = sched.Next(cs, now, Good)
	now = cs.Due
	if cs.State != StateReview {
		t.Fatalf("setup: expected StateReview, got %v", cs.State)
	}

	tests := []struct {
		name   string
		rating Rating
	}{
		{"Again", Again},
		{"Hard", Hard},
		{"Good", Good},
		{"Easy", Easy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, _ := sched.Next(cs, now, tt.rating)
			if !next.Due.After(now) {
				t.Errorf("rating %v: Due = %v, want strictly after now = %v", tt.rating, next.Due, now)
			}
		})
	}
}

func TestRatingForAnswer(t *testing.T) {
	tests := []struct {
		name    string
		correct bool
		want    Rating
	}{
		{"correct answer maps to Good", true, Good},
		{"incorrect answer maps to Again", false, Again},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RatingForAnswer(tt.correct); got != tt.want {
				t.Errorf("RatingForAnswer(%v) = %v, want %v", tt.correct, got, tt.want)
			}
		})
	}
}

// TestDeterminism checks that two independently constructed schedulers,
// given identical inputs, produce bit-identical outputs: fuzz is always off,
// so there is no hidden source of randomness.
func TestDeterminism(t *testing.T) {
	build := func() (*Scheduler, CardState, time.Time) {
		s := NewScheduler(WithRetention(0.9), WithMaxInterval(200))
		now := baseNow
		return s, s.NewCardState(now), now
	}

	sched1, cs1, now1 := build()
	sched2, cs2, now2 := build()

	ratings := []Rating{Good, Good, Hard, Good, Again, Good}
	for i, r := range ratings {
		next1, rev1 := sched1.Next(cs1, now1, r)
		next2, rev2 := sched2.Next(cs2, now2, r)

		if next1 != next2 {
			t.Fatalf("step %d: CardState diverged: %+v vs %+v", i, next1, next2)
		}
		if rev1 != rev2 {
			t.Fatalf("step %d: Review diverged: %+v vs %+v", i, rev1, rev2)
		}

		cs1, now1 = next1, next1.Due
		cs2, now2 = next2, next2.Due
	}
}

func TestSchedulerOptions(t *testing.T) {
	t.Run("WithMaxInterval caps growth", func(t *testing.T) {
		const maxDays = 10
		sched := NewScheduler(WithMaxInterval(maxDays))
		now := baseNow
		cs := sched.NewCardState(now)
		cs, _ = sched.Next(cs, now, Good)
		now = cs.Due
		cs, _ = sched.Next(cs, now, Good)
		now = cs.Due

		var lastDays int
		for i := 0; i < 6; i++ {
			next, rev := sched.Next(cs, now, Good)
			if rev.ScheduledDays > maxDays+5 {
				t.Errorf("round %d: ScheduledDays = %d, want <= cap(%d)+slack", i, rev.ScheduledDays, maxDays)
			}
			lastDays = rev.ScheduledDays
			cs, now = next, next.Due
		}
		if lastDays < maxDays {
			t.Errorf("expected the interval to saturate near the cap of %d days, last was %d", maxDays, lastDays)
		}
	})

	t.Run("WithRetention is accepted", func(t *testing.T) {
		sched := NewScheduler(WithRetention(0.95))
		now := baseNow
		cs := sched.NewCardState(now)
		next, _ := sched.Next(cs, now, Good)
		if !next.Due.After(now) {
			t.Errorf("Due = %v, want strictly after now = %v", next.Due, now)
		}
	})
}

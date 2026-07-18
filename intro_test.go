package engram

import (
	"math"
	"testing"
	"time"
)

var introNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestIntroduceGotIt(t *testing.T) {
	sched := NewScheduler()

	life, cs, hasCard := sched.Introduce(IntroGotIt, introNow)

	if life != LifecycleIntroduced {
		t.Errorf("life = %v, want LifecycleIntroduced", life)
	}
	if !hasCard {
		t.Errorf("hasCard = false, want true")
	}
	if want := sched.NewCardState(introNow); cs != want {
		t.Errorf("cs = %+v, want %+v (NewCardState equivalence)", cs, want)
	}
}

func TestIntroduceKnown(t *testing.T) {
	sched := NewScheduler()

	life, cs, hasCard := sched.Introduce(IntroKnown, introNow)

	if life != LifecycleKnown {
		t.Errorf("life = %v, want LifecycleKnown", life)
	}
	if hasCard {
		t.Errorf("hasCard = true, want false")
	}
	if cs != (CardState{}) {
		t.Errorf("cs = %+v, want zero value", cs)
	}
}

func TestIntroduceTestMe(t *testing.T) {
	sched := NewScheduler()

	life, cs, hasCard := sched.Introduce(IntroTestMe, introNow)

	if life != LifecycleReviewing {
		t.Errorf("life = %v, want LifecycleReviewing", life)
	}
	if !hasCard {
		t.Errorf("hasCard = false, want true")
	}
	if cs.State != StateReview {
		t.Errorf("cs.State = %v, want StateReview", cs.State)
	}
	if cs.Stability <= 10 {
		t.Errorf("cs.Stability = %v, want > 10 days", cs.Stability)
	}
	if cs.Reps != 1 {
		t.Errorf("cs.Reps = %d, want 1", cs.Reps)
	}
	minDue := introNow.Add(14 * 24 * time.Hour)
	maxDue := introNow.Add(21 * 24 * time.Hour)
	if cs.Due.Before(minDue) || cs.Due.After(maxDue) {
		t.Errorf("cs.Due = %v, want between %v and %v (~2-3 weeks out)", cs.Due, minDue, maxDue)
	}
}

// TestSeedCardKnownEquivalence checks the documented equivalence: SeedCardKnown
// is exactly Next(NewCardState(now), now, Easy) with Reps forced to 1 and no
// review-log entry produced.
func TestSeedCardKnownEquivalence(t *testing.T) {
	sched := NewScheduler()

	got := sched.SeedCardKnown(introNow)
	want, _ := sched.Next(sched.NewCardState(introNow), introNow, Easy)
	want.Reps = 1

	if got != want {
		t.Errorf("SeedCardKnown = %+v, want %+v", got, want)
	}
}

// TestSeedCardKnownTracksRetentionOverride asserts SeedCardKnown differs when
// WithRetention differs, proving it goes through the live scheduler weights
// rather than hardcoding a magic Stability/Due.
func TestSeedCardKnownTracksRetentionOverride(t *testing.T) {
	default09 := NewScheduler(WithRetention(0.9)).SeedCardKnown(introNow)
	higher := NewScheduler(WithRetention(0.97)).SeedCardKnown(introNow)

	if default09 == higher {
		t.Errorf("SeedCardKnown produced identical CardState for retention 0.9 and 0.97: %+v", default09)
	}
}

func TestNextIntroductionsBudgetAndOrder(t *testing.T) {
	items := []IntroItem{
		{SkillID: "a", Lifecycle: LifecycleNew},
		{SkillID: "b", Lifecycle: LifecycleIntroduced}, // filtered: not New
		{SkillID: "c", Lifecycle: LifecycleNew},
		{SkillID: "d", Lifecycle: LifecycleKnown}, // filtered: not New
		{SkillID: "e", Lifecycle: LifecycleNew},
	}

	got := NextIntroductions(items, 0, IntroConfig{MaxNewPerDay: 2})

	want := []SkillID{"a", "c"}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d: got=%+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].SkillID != w {
			t.Errorf("got[%d].SkillID = %v, want %v (order must be preserved)", i, got[i].SkillID, w)
		}
	}
}

func TestNextIntroductionsCapZeroUnlimited(t *testing.T) {
	items := []IntroItem{
		{SkillID: "a", Lifecycle: LifecycleNew},
		{SkillID: "b", Lifecycle: LifecycleNew},
		{SkillID: "c", Lifecycle: LifecycleNew},
	}

	got := NextIntroductions(items, 1000, IntroConfig{MaxNewPerDay: 0})
	if len(got) != len(items) {
		t.Errorf("len(got) = %d, want %d (cap=0 is unlimited)", len(got), len(items))
	}
}

func TestNextIntroductionsExhaustedReturnsEmptyNonNil(t *testing.T) {
	items := []IntroItem{
		{SkillID: "a", Lifecycle: LifecycleNew},
	}

	got := NextIntroductions(items, 5, IntroConfig{MaxNewPerDay: 5})
	if got == nil {
		t.Fatal("NextIntroductions = nil, want non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0", len(got))
	}
}

func TestNextIntroductionsNoNewItemsReturnsEmptyNonNil(t *testing.T) {
	items := []IntroItem{
		{SkillID: "a", Lifecycle: LifecycleIntroduced},
		{SkillID: "b", Lifecycle: LifecycleReviewing},
		{SkillID: "c", Lifecycle: LifecycleKnown},
	}

	got := NextIntroductions(items, 0, IntroConfig{MaxNewPerDay: 10})
	if got == nil {
		t.Fatal("NextIntroductions = nil, want non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0", len(got))
	}
}

func TestRemainingIntroBudget(t *testing.T) {
	tests := []struct {
		name            string
		cap             int
		introducedToday int
		want            int
	}{
		{"zero cap is unlimited", 0, 1000, math.MaxInt},
		{"under cap", 10, 3, 7},
		{"exactly at cap", 10, 10, 0},
		{"over cap clamps to zero", 10, 15, 0},
		{"nothing introduced yet", 10, 0, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemainingIntroBudget(tt.cap, tt.introducedToday); got != tt.want {
				t.Errorf("RemainingIntroBudget(%d, %d) = %d, want %d", tt.cap, tt.introducedToday, got, tt.want)
			}
		})
	}
}

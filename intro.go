package engram

import (
	"math"
	"time"
)

// IntroOutcome is the user's response to an introduction card (the three
// buttons).
type IntroOutcome int8

const (
	IntroGotIt  IntroOutcome = iota // → Introduced, fresh FSRS card (StateNew, due now)
	IntroKnown                      // → Known, no card, never quizzed
	IntroTestMe                     // → Reviewing, FSRS card seeded with high initial stability
)

// Introduce applies an introduction outcome at now and returns the resulting
// lifecycle, the CardState to persist, and hasCard (false for IntroKnown,
// where the returned CardState is the zero value). Deterministic; depends
// only on the scheduler's weights and now.
func (s *Scheduler) Introduce(o IntroOutcome, now time.Time) (life Lifecycle, cs CardState, hasCard bool) {
	switch o {
	case IntroKnown:
		return LifecycleKnown, CardState{}, false
	case IntroTestMe:
		return LifecycleReviewing, s.SeedCardKnown(now), true
	default: // IntroGotIt
		return LifecycleIntroduced, s.NewCardState(now), true
	}
}

// SeedCardKnown returns a CardState for an item the user asserts they
// already know but still wants tested occasionally: equivalent to a fresh
// card that received a first Easy rating at now. Concretely (go-fsrs default
// weights): State=StateReview, Stability≈w[3]≈15.69 (days), Difficulty=the
// Easy-initial difficulty, Reps=1, Lapses=0, LastReview=now, Due≈now+16d at
// retention 0.9. NO review-log entry is produced — an introduction is not a
// review. Deterministic. Because SeedCardKnown reuses Next(_, Easy), it
// automatically tracks any WithWeights/WithRetention override.
func (s *Scheduler) SeedCardKnown(now time.Time) CardState {
	cs, _ := s.Next(s.NewCardState(now), now, Easy)
	cs.Reps = 1 // an introduction seed is one exposure, not a scored review
	return cs
}

// IntroItem pairs an item id with its lifecycle for intro selection.
type IntroItem struct {
	SkillID   SkillID
	Lifecycle Lifecycle
}

// IntroConfig configures the daily introduction budget.
type IntroConfig struct {
	MaxNewPerDay int // 0 = unlimited
}

// NextIntroductions returns up to the remaining daily budget of items whose
// Lifecycle == LifecycleNew, PRESERVING the caller's slice order (which
// encodes priority). introducedToday is how many intros were already
// delivered today. Deterministic: no shuffling, stable selection. Returns an
// empty non-nil slice when the budget is exhausted or nothing is New.
func NextIntroductions(items []IntroItem, introducedToday int, cfg IntroConfig) []IntroItem {
	budget := RemainingIntroBudget(cfg.MaxNewPerDay, introducedToday)

	result := make([]IntroItem, 0, len(items))
	for _, it := range items {
		if len(result) >= budget {
			break
		}
		if it.Lifecycle != LifecycleNew {
			continue
		}
		result = append(result, it)
	}
	return result
}

// RemainingIntroBudget returns max(0, cap-introducedToday); cap==0 →
// math.MaxInt (unlimited). Small pure helper mirroring the review-cap
// arithmetic.
func RemainingIntroBudget(cap, introducedToday int) int {
	if cap == 0 {
		return math.MaxInt
	}
	remaining := cap - introducedToday
	if remaining < 0 {
		return 0
	}
	return remaining
}

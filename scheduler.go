package engram

import (
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

// Scheduler wraps go-fsrs/v3 and applies engram's fixed defaults
// (RequestRetention 0.9, MaximumInterval 365 days, fuzz disabled for
// determinism).
type Scheduler struct {
	params fsrs.Parameters
	fsrs   *fsrs.FSRS
}

// SchedulerOption configures a Scheduler built by NewScheduler.
type SchedulerOption func(*Scheduler)

// WithRetention sets the target request retention (default 0.9).
func WithRetention(r float64) SchedulerOption {
	return func(s *Scheduler) {
		s.params.RequestRetention = r
	}
}

// WithMaxInterval sets the maximum interval in days between reviews
// (default 365).
func WithMaxInterval(days int) SchedulerOption {
	return func(s *Scheduler) {
		s.params.MaximumInterval = float64(days)
	}
}

// WithWeights overrides the FSRS model weights (default: go-fsrs defaults).
// If w has fewer than 19 elements, only the leading len(w) default weights
// are overridden and the rest keep their default values; if w has more than
// 19, the extras are ignored. This never panics.
func WithWeights(w []float64) SchedulerOption {
	return func(s *Scheduler) {
		n := len(w)
		if n > len(s.params.W) {
			n = len(s.params.W)
		}
		copy(s.params.W[:n], w[:n])
	}
}

// NewScheduler builds a Scheduler starting from go-fsrs's defaults, then
// overrides MaximumInterval to engram's contract default of 365 days
// (go-fsrs defaults to 36500), and finally applies opts. EnableFuzz is
// always left false: fuzz injects its own randomness outside the caller's
// control, which would break engram's determinism guarantee.
func NewScheduler(opts ...SchedulerOption) *Scheduler {
	s := &Scheduler{params: fsrs.DefaultParam()}
	s.params.MaximumInterval = 365
	s.params.EnableFuzz = false

	for _, opt := range opts {
		opt(s)
	}
	// Fuzz must stay disabled regardless of options, since no option
	// exposes it and determinism is a hard requirement.
	s.params.EnableFuzz = false

	s.fsrs = fsrs.NewFSRS(s.params)
	return s
}

// NewCardState returns the state for a never-reviewed skill: StateNew, zero
// reps/lapses/stability/difficulty, zero LastReview, and Due set to now (due
// immediately).
func (s *Scheduler) NewCardState(now time.Time) CardState {
	cs := fromFSRSCard(fsrs.NewCard())
	cs.Due = now
	return cs
}

// Next applies rating r to cs at time now and returns the updated card state
// plus the corresponding review-log entry. Review.SkillID is left zero; the
// caller fills it in.
func (s *Scheduler) Next(cs CardState, now time.Time, r Rating) (CardState, Review) {
	card := toFSRSCard(cs)
	info := s.fsrs.Next(card, now, fsrs.Rating(r))
	next := fromFSRSCard(info.Card)

	review := Review{
		Rating:           r,
		ReviewedAt:       now,
		StabilityBefore:  cs.Stability,
		DifficultyBefore: cs.Difficulty,
		StabilityAfter:   next.Stability,
		DifficultyAfter:  next.Difficulty,
		StateBefore:      cs.State,
		ScheduledDays:    int(info.Card.ScheduledDays),
		ElapsedDays:      int(info.ReviewLog.ElapsedDays),
	}
	return next, review
}

// RatingForAnswer is the v1 answer→rating policy: wrong → Again,
// correct → Good.
func RatingForAnswer(correct bool) Rating {
	if correct {
		return Good
	}
	return Again
}

// toFSRSCard converts engram's CardState into a fsrs.Card. ElapsedDays and
// ScheduledDays are left zero: fsrs.(*FSRS).Next recomputes elapsed days
// internally from now - card.LastReview and does not consume the input
// card's ElapsedDays/ScheduledDays, so CardState (which carries neither
// field) round-trips losslessly for scheduling purposes.
func toFSRSCard(cs CardState) fsrs.Card {
	return fsrs.Card{
		Due:        cs.Due,
		Stability:  cs.Stability,
		Difficulty: cs.Difficulty,
		Reps:       uint64(cs.Reps),
		Lapses:     uint64(cs.Lapses),
		State:      fsrs.State(cs.State),
		LastReview: cs.LastReview,
	}
}

// fromFSRSCard converts a fsrs.Card into engram's CardState.
func fromFSRSCard(c fsrs.Card) CardState {
	return CardState{
		Due:        c.Due,
		Stability:  c.Stability,
		Difficulty: c.Difficulty,
		Reps:       int(c.Reps),
		Lapses:     int(c.Lapses),
		State:      State(c.State),
		LastReview: c.LastReview,
	}
}

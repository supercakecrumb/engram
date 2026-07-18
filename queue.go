package engram

import "time"

// QueueItem pairs a Skill with its current FSRS card state, as loaded from a
// SkillStore for the skills in a deck.
type QueueItem struct {
	Skill Skill
	Card  CardState
}

// QueueConfig configures NextDue's new-card cap.
type QueueConfig struct {
	MaxNewPerDay int // 0 = unlimited
}

// NextDue picks the next item to study out of items. A due-or-overdue
// review-state card (State != StateNew, Due <= now) always wins, choosing
// the earliest Due (ties keep whichever was encountered first, for
// determinism). Only when no review-state card is due is a new (StateNew)
// card offered instead — again the earliest-Due one whose Due <= now — and
// only while newIntroducedToday is below cfg.MaxNewPerDay (0 means
// unlimited). ok=false with a zero QueueItem means nothing is studyable
// right now.
func NextDue(items []QueueItem, newIntroducedToday int, cfg QueueConfig, now time.Time) (item QueueItem, ok bool) {
	// Pass 1: earliest-due review-state candidate.
	var due QueueItem
	dueFound := false
	for _, it := range items {
		if it.Card.State == StateNew {
			continue
		}
		if it.Card.Due.After(now) {
			continue
		}
		if !dueFound || it.Card.Due.Before(due.Card.Due) {
			due = it
			dueFound = true
		}
	}
	if dueFound {
		return due, true
	}

	// Pass 2: new card, subject to the daily cap.
	if cfg.MaxNewPerDay != 0 && newIntroducedToday >= cfg.MaxNewPerDay {
		return QueueItem{}, false
	}
	var next QueueItem
	nextFound := false
	for _, it := range items {
		if it.Card.State != StateNew {
			continue
		}
		if it.Card.Due.After(now) {
			continue
		}
		if !nextFound || it.Card.Due.Before(next.Card.Due) {
			next = it
			nextFound = true
		}
	}
	if !nextFound {
		return QueueItem{}, false
	}
	return next, true
}

// NextReview picks the next due card from items, all of which already carry
// an FSRS card (Lifecycle Introduced or Reviewing — the app filters
// New/Known out before calling). A due review/relearning card wins by
// earliest Due; otherwise the earliest-due Learning-or-New card. There is NO
// daily-new cap: novelty is gated upstream at introduction. ok=false means
// nothing is due at now.
func NextReview(items []QueueItem, now time.Time) (QueueItem, bool) {
	// Pass 1: earliest-due Review/Relearning candidate.
	var due QueueItem
	dueFound := false
	for _, it := range items {
		if it.Card.State != StateReview && it.Card.State != StateRelearning {
			continue
		}
		if it.Card.Due.After(now) {
			continue
		}
		if !dueFound || it.Card.Due.Before(due.Card.Due) {
			due = it
			dueFound = true
		}
	}
	if dueFound {
		return due, true
	}

	// Pass 2: earliest-due Learning-or-New candidate. No cap: the daily-new
	// gate lives upstream at introduction time in v2.
	var next QueueItem
	nextFound := false
	for _, it := range items {
		if it.Card.State != StateLearning && it.Card.State != StateNew {
			continue
		}
		if it.Card.Due.After(now) {
			continue
		}
		if !nextFound || it.Card.Due.Before(next.Card.Due) {
			next = it
			nextFound = true
		}
	}
	if !nextFound {
		return QueueItem{}, false
	}
	return next, true
}

// CountNewIntroduced counts entries in log whose StateBefore is StateNew and
// whose ReviewedAt falls on the same calendar day as now, both converted to
// loc (nil defaults to time.UTC).
func CountNewIntroduced(log []Review, now time.Time, loc *time.Location) int {
	if loc == nil {
		loc = time.UTC
	}
	today := dayNumber(now, loc)
	count := 0
	for _, r := range log {
		if r.StateBefore != StateNew {
			continue
		}
		if dayNumber(r.ReviewedAt, loc) == today {
			count++
		}
	}
	return count
}

package engram

// Lifecycle is the gating state of an item for one user. It is ORTHOGONAL to
// CardState (the FSRS memory state) and is persisted as a sibling field, never
// inside CardState — CardState must remain a lossless mirror of fsrs.Card.
type Lifecycle int8

const (
	LifecycleNew        Lifecycle = iota // 0: not introduced; no card; eligible for the intro queue; never quizzed
	LifecycleIntroduced                  // 1: intro acknowledged; card exists; FSRS State New/Learning (not yet graduated)
	LifecycleReviewing                   // 2: graduated into durable review; FSRS State Review/Relearning
	LifecycleKnown                       // 3: user asserted mastery; terminal; no active card; reversible later
)

// InFSRS reports whether an item in this lifecycle carries a live FSRS card
// (Introduced or Reviewing). New and Known items do not.
func (l Lifecycle) InFSRS() bool { return l == LifecycleIntroduced || l == LifecycleReviewing }

// LifecycleFor derives the Introduced/Reviewing split from a card's FSRS state.
// Known and New are lifecycle-only (no card) and are never returned here; the
// caller sets those explicitly on the intro outcome. This keeps the coarse
// progress label cheap to store yet always consistent with the card.
func LifecycleFor(cs CardState) Lifecycle {
	if cs.State == StateReview || cs.State == StateRelearning {
		return LifecycleReviewing
	}
	return LifecycleIntroduced
}

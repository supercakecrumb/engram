package engram

import (
	"context"
	"time"
)

// Stores are scoped to ONE user: the application wraps its own database per
// user and passes user-scoped implementations to engram. engram never sees
// user identity itself.

// SkillStore persists the set of skills in a deck and the per-skill card
// state for the single user this store is scoped to.
type SkillStore interface {
	// Skills returns every skill in the given deck.
	Skills(ctx context.Context, deck DeckID) ([]Skill, error)
	// Card returns the stored card state for a skill. The bool is false when
	// the skill has never been seen (no row), in which case the caller should
	// treat it as a fresh card via Scheduler.NewCardState.
	Card(ctx context.Context, skill SkillID) (CardState, bool, error)
	// PutCard upserts the card state for a skill.
	PutCard(ctx context.Context, skill SkillID, cs CardState) error
}

// ReviewStore is the append-only review log for the single user this store is
// scoped to.
type ReviewStore interface {
	// Append records one review-log entry.
	Append(ctx context.Context, r Review) error
	// Log returns the review entries at or after since, in the order the store
	// chooses (callers that need ordering should sort).
	Log(ctx context.Context, since time.Time) ([]Review, error)
}

// Package memstore is an in-memory implementation of engram's SkillStore and
// ReviewStore interfaces, intended for tests and small apps. It is safe for
// concurrent use.
package memstore

import (
	"context"
	"sync"
	"time"

	"github.com/supercakecrumb/engram"
)

// Store is an in-memory, concurrency-safe implementation of
// engram.SkillStore and engram.ReviewStore, scoped to a single user (per the
// contract of those interfaces).
type Store struct {
	mu      sync.Mutex
	skills  map[engram.DeckID][]engram.Skill
	cards   map[engram.SkillID]engram.CardState
	reviews []engram.Review
}

var (
	_ engram.SkillStore  = (*Store)(nil)
	_ engram.ReviewStore = (*Store)(nil)
)

// New returns an empty Store ready for use.
func New() *Store {
	return &Store{
		skills: make(map[engram.DeckID][]engram.Skill),
		cards:  make(map[engram.SkillID]engram.CardState),
	}
}

// AddSkill registers a skill (indexed by its DeckID) so Skills can return
// it. This is an additive helper beyond the pinned SkillStore/ReviewStore
// contract: memstore needs some way to be populated for tests and small
// apps.
func (s *Store) AddSkill(sk engram.Skill) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.skills[sk.DeckID] = append(s.skills[sk.DeckID], sk)
}

// Skills returns a copy of the skills registered for deck, in the order
// they were added via AddSkill. Returns an empty non-nil slice if none are
// registered.
func (s *Store) Skills(ctx context.Context, deck engram.DeckID) ([]engram.Skill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src := s.skills[deck]
	out := make([]engram.Skill, len(src))
	copy(out, src)
	return out, nil
}

// Card returns the stored card state for skill, or a zero CardState and
// false if it has never been put.
func (s *Store) Card(ctx context.Context, skill engram.SkillID) (engram.CardState, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cs, ok := s.cards[skill]
	return cs, ok, nil
}

// PutCard upserts the card state for skill.
func (s *Store) PutCard(ctx context.Context, skill engram.SkillID, cs engram.CardState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cards[skill] = cs
	return nil
}

// Append records a copy of r in the review log.
func (s *Store) Append(ctx context.Context, r engram.Review) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reviews = append(s.reviews, r)
	return nil
}

// Log returns a copy of the reviews with ReviewedAt at or after since, in
// the order they were appended.
func (s *Store) Log(ctx context.Context, since time.Time) ([]engram.Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]engram.Review, 0, len(s.reviews))
	for _, r := range s.reviews {
		if !r.ReviewedAt.Before(since) {
			out = append(out, r)
		}
	}
	return out, nil
}

// Package engram is a storage-agnostic spaced-repetition engine that wraps
// go-fsrs (the Free Spaced Repetition Scheduler). It knows nothing about any
// particular storage backend, UI, or transport — callers own persistence via
// the SkillStore and ReviewStore interfaces and drive the engine explicitly.
//
// The package is deterministic by design: time is always passed in as an
// explicit now time.Time parameter rather than read from the clock, and any
// randomness needed by callers (e.g. in the quiz subpackage) is supplied via
// an injected *rand.Rand. This makes scheduling, queueing, and quiz
// generation fully reproducible in tests.
package engram

import "time"

// DeckID identifies a Deck.
type DeckID string

// SkillID identifies a Skill.
type SkillID string

// ContentID identifies a piece of quiz content.
type ContentID string

// Deck groups related skills (e.g. "Romance languages").
type Deck struct {
	ID   DeckID
	Name string
}

// Skill is a single trackable unit of spaced repetition (e.g. "recognise
// Portuguese in the Romance group").
type Skill struct {
	ID     SkillID
	DeckID DeckID
	Key    string // app-defined answer key, e.g. ISO-639-3 "por"
	Label  string // human label, e.g. "Portuguese"
}

// State is the FSRS memory state of a card.
type State int

const (
	StateNew State = iota
	StateLearning
	StateReview
	StateRelearning
)

// Rating is the grade a reviewer gives when answering a card.
type Rating int

const (
	Again Rating = iota + 1
	Hard
	Good
	Easy
)

// CardState is the per-user FSRS memory state for one skill.
type CardState struct {
	Due        time.Time
	Stability  float64
	Difficulty float64
	Reps       int
	Lapses     int
	State      State
	LastReview time.Time // zero if never reviewed
}

// Review is one append-only review-log entry. SkillID is set by the caller.
type Review struct {
	SkillID          SkillID
	Rating           Rating
	ReviewedAt       time.Time
	StabilityBefore  float64
	DifficultyBefore float64
	StabilityAfter   float64
	DifficultyAfter  float64
	StateBefore      State
	ScheduledDays    int
	ElapsedDays      int
}

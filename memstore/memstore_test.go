package memstore

import (
	"context"
	"testing"
	"time"

	"github.com/supercakecrumb/engram"
)

// Compile-time-ish check that *Store satisfies both storage interfaces (also
// asserted in memstore.go itself; repeating it here documents the contract
// from the test suite's point of view).
var (
	_ engram.SkillStore  = (*Store)(nil)
	_ engram.ReviewStore = (*Store)(nil)
)

func TestStoreSkillsRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := New()

	deckA := engram.DeckID("romance")
	deckB := engram.DeckID("germanic")

	skillsA := []engram.Skill{
		{ID: "por", DeckID: deckA, Key: "por", Label: "Portuguese"},
		{ID: "spa", DeckID: deckA, Key: "spa", Label: "Spanish"},
		{ID: "ita", DeckID: deckA, Key: "ita", Label: "Italian"},
	}
	skillB := engram.Skill{ID: "deu", DeckID: deckB, Key: "deu", Label: "German"}

	for _, sk := range skillsA {
		s.AddSkill(sk)
	}
	s.AddSkill(skillB)

	gotA, err := s.Skills(ctx, deckA)
	if err != nil {
		t.Fatalf("Skills(deckA) error: %v", err)
	}
	if len(gotA) != len(skillsA) {
		t.Fatalf("Skills(deckA) len = %d, want %d", len(gotA), len(skillsA))
	}
	for i, sk := range skillsA {
		if gotA[i] != sk {
			t.Errorf("Skills(deckA)[%d] = %+v, want %+v (insertion order)", i, gotA[i], sk)
		}
	}

	gotB, err := s.Skills(ctx, deckB)
	if err != nil {
		t.Fatalf("Skills(deckB) error: %v", err)
	}
	if len(gotB) != 1 || gotB[0] != skillB {
		t.Errorf("Skills(deckB) = %+v, want [%+v]", gotB, skillB)
	}

	unknown, err := s.Skills(ctx, engram.DeckID("nonexistent"))
	if err != nil {
		t.Fatalf("Skills(unknown deck) error: %v", err)
	}
	if unknown == nil {
		t.Error("Skills(unknown deck) = nil, want non-nil empty slice")
	}
	if len(unknown) != 0 {
		t.Errorf("Skills(unknown deck) = %+v, want empty slice", unknown)
	}
}

func TestStoreSkillsReturnsCopies(t *testing.T) {
	ctx := context.Background()
	s := New()
	deck := engram.DeckID("romance")
	s.AddSkill(engram.Skill{ID: "por", DeckID: deck, Key: "por", Label: "Portuguese"})

	got, err := s.Skills(ctx, deck)
	if err != nil {
		t.Fatalf("Skills error: %v", err)
	}
	got[0].Label = "mutated"

	got2, err := s.Skills(ctx, deck)
	if err != nil {
		t.Fatalf("Skills error: %v", err)
	}
	if got2[0].Label != "Portuguese" {
		t.Errorf("mutating a returned Skills slice affected the store: got %+v", got2[0])
	}
}

func TestStoreCardRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := New()
	skillID := engram.SkillID("por")

	_, ok, err := s.Card(ctx, skillID)
	if err != nil {
		t.Fatalf("Card() error: %v", err)
	}
	if ok {
		t.Fatalf("Card() ok = true before any PutCard, want false")
	}

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cs1 := engram.CardState{Due: now, State: engram.StateNew}
	if err := s.PutCard(ctx, skillID, cs1); err != nil {
		t.Fatalf("PutCard error: %v", err)
	}

	got, ok, err := s.Card(ctx, skillID)
	if err != nil {
		t.Fatalf("Card() error: %v", err)
	}
	if !ok {
		t.Fatalf("Card() ok = false after PutCard, want true")
	}
	if got != cs1 {
		t.Errorf("Card() = %+v, want %+v", got, cs1)
	}

	cs2 := engram.CardState{
		Due:       now.Add(24 * time.Hour),
		State:     engram.StateReview,
		Stability: 4.5,
		Reps:      2,
	}
	if err := s.PutCard(ctx, skillID, cs2); err != nil {
		t.Fatalf("PutCard (update) error: %v", err)
	}
	got2, ok, err := s.Card(ctx, skillID)
	if err != nil {
		t.Fatalf("Card() error: %v", err)
	}
	if !ok {
		t.Fatalf("Card() ok = false after second PutCard, want true")
	}
	if got2 != cs2 {
		t.Errorf("Card() after update = %+v, want %+v", got2, cs2)
	}
}

func TestStoreLogFiltersAndPreservesOrder(t *testing.T) {
	ctx := context.Background()
	s := New()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	reviews := []engram.Review{
		{SkillID: "a", ReviewedAt: base},
		{SkillID: "b", ReviewedAt: base.Add(1 * time.Hour)},
		{SkillID: "c", ReviewedAt: base.Add(2 * time.Hour)},
		{SkillID: "d", ReviewedAt: base.Add(3 * time.Hour)},
	}
	for _, r := range reviews {
		if err := s.Append(ctx, r); err != nil {
			t.Fatalf("Append error: %v", err)
		}
	}

	since := base.Add(1 * time.Hour)
	got, err := s.Log(ctx, since)
	if err != nil {
		t.Fatalf("Log error: %v", err)
	}
	want := reviews[1:]
	if len(got) != len(want) {
		t.Fatalf("Log(since) returned %d entries, want %d: got=%+v", len(got), len(want), got)
	}
	for i, r := range want {
		if got[i] != r {
			t.Errorf("Log(since)[%d] = %+v, want %+v (append order)", i, got[i], r)
		}
	}
}

func TestStoreLogReturnsCopies(t *testing.T) {
	ctx := context.Background()
	s := New()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	if err := s.Append(ctx, engram.Review{SkillID: "a", ReviewedAt: base}); err != nil {
		t.Fatalf("Append error: %v", err)
	}

	got, err := s.Log(ctx, base)
	if err != nil {
		t.Fatalf("Log error: %v", err)
	}
	got[0].SkillID = "mutated"

	got2, err := s.Log(ctx, base)
	if err != nil {
		t.Fatalf("Log error: %v", err)
	}
	if got2[0].SkillID != "a" {
		t.Errorf("mutating a returned Log slice affected the store: got %+v", got2[0])
	}
}

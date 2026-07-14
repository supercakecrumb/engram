package quiz

import (
	"math/rand"
	"testing"
	"time"

	"github.com/supercakecrumb/engram"
)

func romanceDeckSkills() []engram.Skill {
	return []engram.Skill{
		{ID: "sk-por", DeckID: "romance", Key: "por", Label: "Portuguese"},
		{ID: "sk-spa", DeckID: "romance", Key: "spa", Label: "Spanish"},
		{ID: "sk-ita", DeckID: "romance", Key: "ita", Label: "Italian"},
		{ID: "sk-fra", DeckID: "romance", Key: "fra", Label: "French"},
	}
}

func TestGenerateExactlyOneCorrectOption(t *testing.T) {
	deckSkills := romanceDeckSkills()
	target := deckSkills[0] // por
	content := Content{ID: "c1", Payload: "Ola, tudo bem?"}

	rng := rand.New(rand.NewSource(1))
	ex := Generate(rng, target, deckSkills, content)

	if len(ex.Options) != len(deckSkills) {
		t.Fatalf("len(Options) = %d, want %d", len(ex.Options), len(deckSkills))
	}
	if ex.SkillID != target.ID {
		t.Errorf("Exercise.SkillID = %v, want %v", ex.SkillID, target.ID)
	}

	seenKeys := make(map[string]int, len(deckSkills))
	correctCount := 0
	for _, o := range ex.Options {
		seenKeys[o.Key]++
		if o.SkillID == target.ID {
			correctCount++
			if o.Key != target.Key {
				t.Errorf("correct option Key = %q, want %q", o.Key, target.Key)
			}
		}
	}
	if correctCount != 1 {
		t.Errorf("options with SkillID == target.ID = %d, want exactly 1", correctCount)
	}
	for _, sk := range deckSkills {
		if seenKeys[sk.Key] != 1 {
			t.Errorf("deck skill key %q appears %d times among Options, want exactly 1", sk.Key, seenKeys[sk.Key])
		}
	}
}

func TestGenerateDeterministicShuffle(t *testing.T) {
	deckSkills := romanceDeckSkills()
	target := deckSkills[2] // ita
	content := Content{ID: "c2", Payload: "Come stai?"}

	rng1 := rand.New(rand.NewSource(42))
	ex1 := Generate(rng1, target, deckSkills, content)

	rng2 := rand.New(rand.NewSource(42))
	ex2 := Generate(rng2, target, deckSkills, content)

	if len(ex1.Options) != len(ex2.Options) {
		t.Fatalf("option counts differ between identical-seed runs: %d vs %d", len(ex1.Options), len(ex2.Options))
	}
	for i := range ex1.Options {
		if ex1.Options[i] != ex2.Options[i] {
			t.Errorf("option %d differs between same-seed runs: %+v vs %+v", i, ex1.Options[i], ex2.Options[i])
		}
	}
}

// TestGenerateDifferentSeedsCanDiffer proves the shuffle actually reorders
// options rather than being a no-op: seeds 1 and 9999 are known (verified
// empirically for this 4-element input) to produce different orderings.
// The assertion itself only checks that shuffling is capable of producing a
// different order for some seed pair, not that any two arbitrary seeds must
// differ.
func TestGenerateDifferentSeedsCanDiffer(t *testing.T) {
	deckSkills := romanceDeckSkills()
	target := deckSkills[2]
	content := Content{ID: "c3", Payload: "Come va?"}

	rngA := rand.New(rand.NewSource(1))
	exA := Generate(rngA, target, deckSkills, content)

	rngB := rand.New(rand.NewSource(9999))
	exB := Generate(rngB, target, deckSkills, content)

	differs := false
	for i := range exA.Options {
		if exA.Options[i] != exB.Options[i] {
			differs = true
			break
		}
	}
	if !differs {
		t.Errorf("expected seeds 1 and 9999 to produce different option orders, got identical orders: %+v", exA.Options)
	}
}

func TestExerciseGrade(t *testing.T) {
	deckSkills := romanceDeckSkills()
	target := deckSkills[1] // spa
	content := Content{ID: "c4", Payload: "Hola, que tal?"}

	rng := rand.New(rand.NewSource(7))
	ex := Generate(rng, target, deckSkills, content)
	ex.ID = "ex-1"

	for _, opt := range ex.Options {
		opt := opt
		t.Run("choose_"+opt.Key, func(t *testing.T) {
			correct, rating := ex.Grade(opt.Key)

			wantCorrect := opt.SkillID == target.ID
			if correct != wantCorrect {
				t.Errorf("Grade(%q) correct = %v, want %v", opt.Key, correct, wantCorrect)
			}

			wantRating := engram.Again
			if wantCorrect {
				wantRating = engram.Good
			}
			if rating != wantRating {
				t.Errorf("Grade(%q) rating = %v, want %v", opt.Key, rating, wantRating)
			}
		})
	}
}

func TestConfusion(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	var attempts []Attempt
	// target "por": 5 attempts total - 2 correct, 2 chose "spa", 1 chose "ita".
	porChoices := []struct {
		chosen  string
		correct bool
	}{
		{"por", true},
		{"por", true},
		{"spa", false},
		{"spa", false},
		{"ita", false},
	}
	for i, c := range porChoices {
		attempts = append(attempts, Attempt{
			SkillID:    "sk-por",
			ChosenKey:  c.chosen,
			CorrectKey: "por",
			Correct:    c.correct,
			AnsweredAt: now.Add(time.Duration(i) * time.Minute),
		})
	}
	// target "fra": 2 attempts, both correct -> no confusion pair for fra.
	for i := 0; i < 2; i++ {
		attempts = append(attempts, Attempt{
			SkillID:    "sk-fra",
			ChosenKey:  "fra",
			CorrectKey: "fra",
			Correct:    true,
			AnsweredAt: now.Add(time.Duration(10+i) * time.Minute),
		})
	}

	got := Confusion(attempts)

	want := []ConfusionPair{
		{TargetKey: "por", ChosenKey: "spa", Count: 2, Share: 2.0 / 5.0},
		{TargetKey: "por", ChosenKey: "ita", Count: 1, Share: 1.0 / 5.0},
	}

	if len(got) != len(want) {
		t.Fatalf("Confusion() returned %d pairs, want %d: got=%+v", len(got), len(want), got)
	}

	const tol = 1e-9
	for i, w := range want {
		g := got[i]
		if g.TargetKey != w.TargetKey || g.ChosenKey != w.ChosenKey || g.Count != w.Count {
			t.Errorf("pair %d = %+v, want %+v", i, g, w)
		}
		if diff := g.Share - w.Share; diff < -tol || diff > tol {
			t.Errorf("pair %d Share = %v, want %v (tolerance %v)", i, g.Share, w.Share, tol)
		}
	}
}

func TestConfusionEmpty(t *testing.T) {
	got := Confusion(nil)
	if got == nil {
		t.Fatal("Confusion(nil) = nil, want non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("Confusion(nil) = %+v, want empty slice", got)
	}
}

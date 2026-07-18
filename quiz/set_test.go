package quiz

import (
	"math/rand"
	"testing"

	"github.com/supercakecrumb/engram"
)

func TestCanonSetSortsAndDedups(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want KeySet
	}{
		{"already sorted, no dups", []string{"dan", "nor"}, KeySet{"dan", "nor"}},
		{"unsorted", []string{"nor", "dan"}, KeySet{"dan", "nor"}},
		{"duplicates removed", []string{"nor", "dan", "nor", "dan", "dan"}, KeySet{"dan", "nor"}},
		{"single key", []string{"nor"}, KeySet{"nor"}},
		{"empty input", nil, KeySet{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanonSet(tt.in...)
			if got == nil {
				t.Fatal("CanonSet returned nil, want non-nil")
			}
			if len(got) != len(tt.want) {
				t.Fatalf("CanonSet(%v) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("CanonSet(%v)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func norDanTarget() SetOption {
	return SetOption{Keys: CanonSet("nor", "dan"), Label: "Norwegian / Danish"}
}

func setCandidates() []SetOption {
	return []SetOption{
		{Keys: CanonSet("swe"), Label: "Swedish"},
		{Keys: CanonSet("nor", "dan"), Label: "Norwegian / Danish"},
		{Keys: CanonSet("isl"), Label: "Icelandic"},
		{Keys: CanonSet("fin"), Label: "Finnish"},
	}
}

func TestGenerateSetTargetAlreadyPresent(t *testing.T) {
	target := norDanTarget()
	candidates := setCandidates()
	content := Content{ID: "c1", Payload: "ø"}

	rng := rand.New(rand.NewSource(1))
	ex := GenerateSet(rng, target, candidates, content)

	if len(ex.Options) != len(candidates) {
		t.Fatalf("len(Options) = %d, want %d (target already present, nothing appended)", len(ex.Options), len(candidates))
	}

	correctCount := 0
	for _, o := range ex.Options {
		if setEqual(o.Keys, target.Keys) {
			correctCount++
		}
	}
	if correctCount != 1 {
		t.Errorf("options set-equal to target = %d, want exactly 1", correctCount)
	}
}

func TestGenerateSetAppendsMissingTarget(t *testing.T) {
	target := SetOption{Keys: CanonSet("ces", "slk"), Label: "Czech / Slovak"}
	candidates := setCandidates() // does not contain target
	content := Content{ID: "c2", Payload: "ě"}

	rng := rand.New(rand.NewSource(2))
	ex := GenerateSet(rng, target, candidates, content)

	if len(ex.Options) != len(candidates)+1 {
		t.Fatalf("len(Options) = %d, want %d (target appended)", len(ex.Options), len(candidates)+1)
	}

	found := false
	for _, o := range ex.Options {
		if setEqual(o.Keys, target.Keys) {
			found = true
		}
	}
	if !found {
		t.Error("appended target set not found among Options")
	}
}

func TestGenerateSetDeterministicShuffle(t *testing.T) {
	target := norDanTarget()
	candidates := setCandidates()
	content := Content{ID: "c3", Payload: "ø"}

	rng1 := rand.New(rand.NewSource(42))
	ex1 := GenerateSet(rng1, target, candidates, content)

	rng2 := rand.New(rand.NewSource(42))
	ex2 := GenerateSet(rng2, target, candidates, content)

	if len(ex1.Options) != len(ex2.Options) {
		t.Fatalf("option counts differ between identical-seed runs: %d vs %d", len(ex1.Options), len(ex2.Options))
	}
	for i := range ex1.Options {
		o1, o2 := ex1.Options[i], ex2.Options[i]
		if o1.Label != o2.Label || !setEqual(o1.Keys, o2.Keys) {
			t.Errorf("option %d differs between same-seed runs: %+v vs %+v", i, o1, o2)
		}
	}
}

func TestSetExerciseGradeIndex(t *testing.T) {
	target := norDanTarget()
	candidates := setCandidates()
	content := Content{ID: "c4", Payload: "ø"}

	rng := rand.New(rand.NewSource(7))
	ex := GenerateSet(rng, target, candidates, content)
	ex.ID = "ex-1"

	for i, opt := range ex.Options {
		i, opt := i, opt
		t.Run(opt.Label, func(t *testing.T) {
			correct, rating := ex.GradeIndex(i)

			wantCorrect := setEqual(opt.Keys, target.Keys)
			if correct != wantCorrect {
				t.Errorf("GradeIndex(%d) correct = %v, want %v", i, correct, wantCorrect)
			}

			wantRating := engram.Again
			if wantCorrect {
				wantRating = engram.Good
			}
			if rating != wantRating {
				t.Errorf("GradeIndex(%d) rating = %v, want %v", i, rating, wantRating)
			}
		})
	}
}

// TestSetExerciseGradeIndexSetEqualityIsOrderInsensitive proves grading
// compares sets, not slice order: a differently-ordered-but-equal-keys
// SetOption at the target's own index (via a hand-built exercise, not
// GenerateSet) must still grade correct.
func TestSetExerciseGradeIndexSetEqualityIsOrderInsensitive(t *testing.T) {
	ex := SetExercise{
		Target: SetOption{Keys: KeySet{"dan", "nor"}, Label: "target"},
		Options: []SetOption{
			{Keys: KeySet{"nor", "dan"}, Label: "reordered, still equal"}, // same set, different order
			{Keys: KeySet{"swe"}, Label: "wrong"},
		},
	}

	correct, _ := ex.GradeIndex(0)
	if !correct {
		t.Error("GradeIndex(0) = false, want true (set-equality must be order-insensitive)")
	}

	correct, _ = ex.GradeIndex(1)
	if correct {
		t.Error("GradeIndex(1) = true, want false (disjoint set)")
	}
}

func TestSetExerciseGradeIndexOutOfRange(t *testing.T) {
	ex := SetExercise{
		Target: norDanTarget(),
		Options: []SetOption{
			{Keys: CanonSet("nor", "dan"), Label: "correct"},
			{Keys: CanonSet("swe"), Label: "wrong"},
		},
	}

	tests := []struct {
		name string
		i    int
	}{
		{"negative index", -1},
		{"index equal to length", 2},
		{"index far past length", 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			correct, rating := ex.GradeIndex(tt.i)
			if correct {
				t.Errorf("GradeIndex(%d) correct = true, want false (out of range)", tt.i)
			}
			if rating != engram.Again {
				t.Errorf("GradeIndex(%d) rating = %v, want Again", tt.i, rating)
			}
		})
	}
}

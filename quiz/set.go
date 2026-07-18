package quiz

import (
	"math/rand"
	"sort"

	"github.com/supercakecrumb/engram"
)

// KeySet is a canonicalized (sorted, deduplicated) set of answer keys. Build
// one via CanonSet rather than a raw literal, so equality comparisons are
// order-insensitive by construction.
type KeySet []string

// CanonSet canonicalizes keys into a KeySet: sorted ascending, duplicates
// removed. Returns an empty non-nil KeySet for no input.
func CanonSet(keys ...string) KeySet {
	sorted := append([]string(nil), keys...)
	sort.Strings(sorted)

	out := make(KeySet, 0, len(sorted))
	for i, k := range sorted {
		if i > 0 && k == sorted[i-1] {
			continue
		}
		out = append(out, k)
	}
	return out
}

// setEqual reports whether a and b represent the same set of keys,
// regardless of the order or duplication either was constructed with.
func setEqual(a, b KeySet) bool {
	ca, cb := CanonSet(a...), CanonSet(b...)
	if len(ca) != len(cb) {
		return false
	}
	for i := range ca {
		if ca[i] != cb[i] {
			return false
		}
	}
	return true
}

// SetOption is one selectable answer in a SetExercise: a button that asserts
// a set of keys (e.g. "ø" -> {nor,dan}).
type SetOption struct {
	Keys  KeySet // the set this button asserts
	Label string // e.g. "Norwegian / Danish"
}

// SetExercise is a set-choice question: some Content plus a shuffled set of
// Options, exactly one of which set-equals Target.
type SetExercise struct {
	ID      string // opaque, set by caller (e.g. uuid)
	Content Content
	Target  SetOption   // the correct set
	Options []SetOption // shuffled candidates; exactly one equals Target (set-equality)
}

// GenerateSet builds a set-choice exercise: candidates must already include
// the target set (appended if missing, mirroring Generate). Shuffled with
// the injected rng — same seed + input order ⇒ same output order.
// SetExercise.ID is left empty for the caller to fill in.
func GenerateSet(rng *rand.Rand, target SetOption, candidates []SetOption, c Content) SetExercise {
	opts := make([]SetOption, 0, len(candidates)+1)
	targetPresent := false
	for _, cand := range candidates {
		opts = append(opts, cand)
		if setEqual(cand.Keys, target.Keys) {
			targetPresent = true
		}
	}
	if !targetPresent {
		opts = append(opts, target)
	}

	rng.Shuffle(len(opts), func(i, j int) {
		opts[i], opts[j] = opts[j], opts[i]
	})

	return SetExercise{
		Content: c,
		Target:  target,
		Options: opts,
	}
}

// GradeIndex grades the tapped option position (Telegram sends an index, not
// the set — sets don't fit the 64-byte callback budget). correct =
// Options[i].Keys equals Target.Keys (order-insensitive). Out-of-range ⇒
// wrong.
func (e SetExercise) GradeIndex(i int) (correct bool, r engram.Rating) {
	if i < 0 || i >= len(e.Options) {
		return false, engram.RatingForAnswer(false)
	}
	correct = setEqual(e.Options[i].Keys, e.Target.Keys)
	return correct, engram.RatingForAnswer(correct)
}

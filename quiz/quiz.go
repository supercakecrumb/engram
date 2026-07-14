// Package quiz builds multiple-choice exercises over engram skills and
// grades the answers. It is deterministic: shuffling uses an injected
// *rand.Rand rather than any package-level or global source of randomness.
package quiz

import (
	"context"
	"math/rand"
	"sort"
	"time"

	"github.com/supercakecrumb/engram"
)

// Content is one piece of quiz material (e.g. a sentence) shown to the user
// alongside the multiple-choice options. Interpretation of Payload is
// app-defined.
type Content struct {
	ID      engram.ContentID
	Payload string // e.g. the sentence; interpretation is app-defined
}

// ContentProvider samples fresh Content for a skill, excluding content the
// caller has already seen recently.
type ContentProvider interface {
	Sample(ctx context.Context, skillKey string, exclude []engram.ContentID) (Content, error)
}

// Option is one selectable answer in an Exercise.
type Option struct {
	SkillID engram.SkillID
	Key     string
	Label   string
}

// Exercise is a single multiple-choice question: some Content plus a
// shuffled set of Options, exactly one of which is correct.
type Exercise struct {
	ID      string // opaque, set by caller (e.g. uuid)
	SkillID engram.SkillID
	Content Content
	Options []Option // shuffled, exactly one correct
}

// Generate builds a multiple-choice Exercise for target: every skill in
// deckSkills becomes one Option. target is expected to be present in
// deckSkills (matched by Key); if it isn't, an Option for target is appended
// so there is always exactly one correct answer. Options are shuffled with
// the injected rng, so the same seed and input order always produce the same
// output order. Exercise.ID is left empty for the caller to fill in.
func Generate(rng *rand.Rand, target engram.Skill, deckSkills []engram.Skill, c Content) Exercise {
	opts := make([]Option, 0, len(deckSkills)+1)
	targetPresent := false
	for _, sk := range deckSkills {
		opts = append(opts, Option{SkillID: sk.ID, Key: sk.Key, Label: sk.Label})
		if sk.Key == target.Key {
			targetPresent = true
		}
	}
	if !targetPresent {
		opts = append(opts, Option{SkillID: target.ID, Key: target.Key, Label: target.Label})
	}

	rng.Shuffle(len(opts), func(i, j int) {
		opts[i], opts[j] = opts[j], opts[i]
	})

	return Exercise{
		SkillID: target.ID,
		Content: c,
		Options: opts,
	}
}

// Grade reports whether chosenKey matches the target skill's key, and the
// FSRS rating that should be applied for the answer.
func (e Exercise) Grade(chosenKey string) (correct bool, r engram.Rating) {
	correctKey, ok := "", false
	for _, o := range e.Options {
		if o.SkillID == e.SkillID {
			correctKey = o.Key
			ok = true
			break
		}
	}
	if !ok {
		return false, engram.RatingForAnswer(false)
	}
	correct = chosenKey == correctKey
	return correct, engram.RatingForAnswer(correct)
}

// Attempt is one answered Exercise, as needed for confusion analysis.
type Attempt struct {
	SkillID    engram.SkillID
	ChosenKey  string
	CorrectKey string
	Correct    bool
	AnsweredAt time.Time
	ResponseMS int
}

// ConfusionPair summarizes how often ChosenKey was picked when TargetKey was
// the correct answer.
type ConfusionPair struct {
	TargetKey string
	ChosenKey string
	Count     int
	Share     float64 // Count / attempts where target == TargetKey
}

// confusionKey groups wrong attempts by (target, chosen) key pair.
type confusionKey struct {
	TargetKey string
	ChosenKey string
}

// Confusion aggregates wrong answers from attempts into ConfusionPairs,
// sorted by Count descending. Share is Count divided by the number of
// attempts (correct or wrong) where CorrectKey == TargetKey, i.e. every time
// that target was the correct answer, not just the wrong ones. Ties in Count
// are broken by TargetKey then ChosenKey, both ascending, so the result is
// stable regardless of input order. Returns an empty non-nil slice when
// there are no wrong attempts.
func Confusion(attempts []Attempt) []ConfusionPair {
	totalsByTarget := make(map[string]int, len(attempts))
	for _, a := range attempts {
		totalsByTarget[a.CorrectKey]++
	}

	counts := make(map[confusionKey]int)
	for _, a := range attempts {
		if a.Correct {
			continue
		}
		counts[confusionKey{TargetKey: a.CorrectKey, ChosenKey: a.ChosenKey}]++
	}

	pairs := make([]ConfusionPair, 0, len(counts))
	for k, cnt := range counts {
		var share float64
		if total := totalsByTarget[k.TargetKey]; total > 0 {
			share = float64(cnt) / float64(total)
		}
		pairs = append(pairs, ConfusionPair{
			TargetKey: k.TargetKey,
			ChosenKey: k.ChosenKey,
			Count:     cnt,
			Share:     share,
		})
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count != pairs[j].Count {
			return pairs[i].Count > pairs[j].Count
		}
		if pairs[i].TargetKey != pairs[j].TargetKey {
			return pairs[i].TargetKey < pairs[j].TargetKey
		}
		return pairs[i].ChosenKey < pairs[j].ChosenKey
	})

	return pairs
}

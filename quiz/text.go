package quiz

import (
	"strings"

	"github.com/supercakecrumb/engram"
)

// TextMatcher grades a free-typed answer against a set of accepted
// spellings.
type TextMatcher struct {
	Accept   []string // canonical + alias spellings, e.g. ["Norwegian","Norsk","NO"]
	MaxEdits int      // Levenshtein tolerance (e.g. 2); 0 = exact after normalization
}

// Match reports whether typed matches any accepted spelling after Normalize
// (Unicode casefold + trim + internal-space collapse) within a Levenshtein
// tolerance. The tolerance for each accepted spelling is
// min(MaxEdits, max(0, len(spelling)-2)), so short spellings (e.g. 2-letter
// ISO codes) require near-exact matches and cannot be reached by unrelated
// short input on distance alone. Deterministic, no rng. Empty typed ⇒ false.
func (m TextMatcher) Match(typed string) (correct bool, r engram.Rating) {
	nt := Normalize(typed)
	if nt == "" {
		return false, engram.RatingForAnswer(false)
	}

	for _, a := range m.Accept {
		na := Normalize(a)
		// Scale the edit tolerance to the length of the accepted spelling:
		// a spelling of L runes tolerates at most L-2 edits (so a 3-char
		// code allows 1 edit and a 1- or 2-char code requires an exact
		// match), floored at 0 and capped at MaxEdits. Real words (L>=4)
		// keep the full MaxEdits tolerance, while very short accepted
		// spellings (e.g. 2-letter ISO codes like "pk") can no longer be
		// reached by unrelated short input such as "idk" on distance alone.
		tol := m.MaxEdits
		if limit := len([]rune(na)) - 2; limit < tol {
			if limit < 0 {
				limit = 0
			}
			tol = limit
		}
		if Levenshtein(nt, na) <= tol {
			return true, engram.RatingForAnswer(true)
		}
	}
	return false, engram.RatingForAnswer(false)
}

// Normalize canonicalizes s for matching: Unicode casefold (simple
// lowercase), leading/trailing trim, and collapsing runs of internal
// whitespace to a single space.
func Normalize(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// Levenshtein returns the edit distance between a and b, counted rune by
// rune: the minimum number of single-rune insertions, deletions, or
// substitutions needed to transform a into b.
func Levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = minInt(del, minInt(ins, sub))
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

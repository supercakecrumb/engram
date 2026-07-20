package quiz

import (
	"testing"

	"github.com/supercakecrumb/engram"
)

func TestNormalizeTrimAndCasefold(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"already normalized", "norwegian", "norwegian"},
		{"uppercase", "NORWEGIAN", "norwegian"},
		{"mixed case", "NorWegian", "norwegian"},
		{"leading/trailing whitespace trimmed", "  norwegian  ", "norwegian"},
		{"internal whitespace collapsed", "New   York", "new york"},
		{"tabs and newlines collapsed", "New\t\nYork", "new york"},
		{"empty string", "", ""},
		{"whitespace only", "   \t  ", ""},
		// Go's simple (non-full) Unicode case mapping folds U+0130 (LATIN
		// CAPITAL LETTER I WITH DOT ABOVE) straight to plain 'i', which is
		// exactly the behavior Normalize needs here.
		{"Unicode casefold: Turkish dotted I", "İstanbul", "istanbul"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Normalize(tt.in); got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestLevenshteinBasics(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"identical strings", "kitten", "kitten", 0},
		{"classic kitten/sitting", "kitten", "sitting", 3},
		{"one substitution", "cat", "cot", 1},
		{"one insertion", "cat", "cats", 1},
		{"one deletion", "cats", "cat", 1},
		{"empty a", "", "abc", 3},
		{"empty b", "abc", "", 3},
		{"both empty", "", "", 0},
		{"completely different", "abc", "xyz", 3},
		{"unicode runes counted, not bytes", "ø", "o", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Levenshtein(tt.a, tt.b); got != tt.want {
				t.Errorf("Levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestLevenshteinSymmetric(t *testing.T) {
	pairs := [][2]string{{"kitten", "sitting"}, {"norwegian", "norsk"}, {"", "abc"}}
	for _, p := range pairs {
		got1 := Levenshtein(p[0], p[1])
		got2 := Levenshtein(p[1], p[0])
		if got1 != got2 {
			t.Errorf("Levenshtein(%q,%q)=%d != Levenshtein(%q,%q)=%d, want symmetric", p[0], p[1], got1, p[1], p[0], got2)
		}
	}
}

func TestTextMatcherMatch(t *testing.T) {
	m := TextMatcher{Accept: []string{"Norwegian", "Norsk", "NO"}, MaxEdits: 2}

	tests := []struct {
		name        string
		typed       string
		wantCorrect bool
	}{
		{"exact match", "Norwegian", true},
		{"case-insensitive match", "norwegian", true},
		{"alias hit", "Norsk", true},
		{"short alias hit, case-insensitive", "no", true},
		{"1 edit, within MaxEdits=2 tolerance", "Norwegin", true},      // 1 deletion from "Norwegian"
		{"3 edits, exceeds MaxEdits=2 tolerance", "Norwegain!", false}, // 3 edits from the nearest accepted spelling
		{"completely wrong", "Swedish", false},
		{"empty typed", "", false},
		{"whitespace-only typed", "   ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			correct, rating := m.Match(tt.typed)
			if correct != tt.wantCorrect {
				t.Errorf("Match(%q) correct = %v, want %v", tt.typed, correct, tt.wantCorrect)
			}
			wantRating := engram.RatingForAnswer(tt.wantCorrect)
			if rating != wantRating {
				t.Errorf("Match(%q) rating = %v, want %v", tt.typed, rating, wantRating)
			}
		})
	}
}

func TestTextMatcherShortCodesRequireNearExact(t *testing.T) {
	tests := []struct {
		name        string
		accept      []string
		typed       string
		wantCorrect bool
	}{
		// False positives to kill: "idk" is within Levenshtein 2 of some
		// 2-letter ISO codes, but must not grade correct.
		{"idk vs Pakistan/PK/PAK", []string{"Pakistan", "PK", "PAK"}, "idk", false},
		{"idk vs Taiwan/TW/TWN", []string{"Taiwan", "TW", "TWN"}, "idk", false},
		{"idk vs Bangladesh/BD/BGD", []string{"Bangladesh", "BD", "BGD"}, "idk", false},

		// Real words must still tolerate typos.
		{"spain (case-fold)", []string{"Spain", "ES", "ESP"}, "spain", true},
		{"Spain exact", []string{"Spain", "ES", "ESP"}, "Spain", true},
		{"germony (1 typo)", []string{"Germany", "DE", "DEU"}, "germony", true},
		{"germeny (edit-dist 2)", []string{"Germany", "DE", "DEU"}, "germeny", true},

		// Exact short codes still work.
		{"pk vs PK (case-fold)", []string{"PK"}, "pk", true},
		{"PK exact", []string{"PK"}, "PK", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := TextMatcher{Accept: tt.accept, MaxEdits: 2}
			correct, rating := m.Match(tt.typed)
			if correct != tt.wantCorrect {
				t.Errorf("Match(%q) against %v correct = %v, want %v", tt.typed, tt.accept, correct, tt.wantCorrect)
			}
			if wantRating := engram.RatingForAnswer(tt.wantCorrect); rating != wantRating {
				t.Errorf("Match(%q) rating = %v, want %v", tt.typed, rating, wantRating)
			}
		})
	}
}

func TestTextMatcherMaxEditsExactlyAtBoundary(t *testing.T) {
	m := TextMatcher{Accept: []string{"kitten"}, MaxEdits: 3}

	// "sitting" is exactly edit distance 3 from "kitten".
	correct, _ := m.Match("sitting")
	if !correct {
		t.Error("Match at exactly MaxEdits boundary = false, want true (<=  MaxEdits)")
	}

	mTighter := TextMatcher{Accept: []string{"kitten"}, MaxEdits: 2}
	correct, _ = mTighter.Match("sitting")
	if correct {
		t.Error("Match one edit beyond MaxEdits = true, want false")
	}
}

func TestTextMatcherWhitespaceCollapseAndCasefold(t *testing.T) {
	m := TextMatcher{Accept: []string{"New York"}, MaxEdits: 0}

	tests := []string{"New York", "new york", "  New   York  ", "NEW\tYORK"}
	for _, typed := range tests {
		t.Run(typed, func(t *testing.T) {
			correct, _ := m.Match(typed)
			if !correct {
				t.Errorf("Match(%q) = false, want true (whitespace collapse + casefold)", typed)
			}
		})
	}
}

func TestTextMatcherUnicodeCasefold(t *testing.T) {
	m := TextMatcher{Accept: []string{"İstanbul"}, MaxEdits: 0}

	correct, _ := m.Match("istanbul")
	if !correct {
		t.Error("Match(\"istanbul\") against accepted \"İstanbul\" = false, want true (Unicode casefold)")
	}
}

func TestTextMatcherZeroMaxEditsRequiresExact(t *testing.T) {
	m := TextMatcher{Accept: []string{"French"}, MaxEdits: 0}

	correct, _ := m.Match("Frnch") // 1 edit away
	if correct {
		t.Error("Match with MaxEdits=0 accepted a 1-edit-away typo, want exact match only")
	}

	correct, _ = m.Match("French")
	if !correct {
		t.Error("Match with MaxEdits=0 rejected an exact (post-normalize) match")
	}
}

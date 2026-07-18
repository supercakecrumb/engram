package quiz

import "testing"

// TestModeZeroValueIsSingle asserts the zero value of Mode is ModeSingle, the
// property that keeps TipRequest.Mode additive: existing callers that never
// set it get single-choice behavior unchanged.
func TestModeZeroValueIsSingle(t *testing.T) {
	var m Mode
	if m != ModeSingle {
		t.Errorf("zero value of Mode = %v, want ModeSingle", m)
	}
}

func TestModeValuesAreDistinct(t *testing.T) {
	modes := map[Mode]string{
		ModeSingle: "ModeSingle",
		ModeSet:    "ModeSet",
		ModeText:   "ModeText",
	}
	if len(modes) != 3 {
		t.Errorf("expected 3 distinct Mode values, got %d", len(modes))
	}
}

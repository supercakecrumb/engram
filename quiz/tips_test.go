package quiz

import "testing"

func TestTipFunc(t *testing.T) {
	var got TipRequest
	p := TipFunc(func(req TipRequest) string {
		got = req
		return "tip for " + req.CorrectKey
	})

	req := TipRequest{
		ContentPayload: "Não fales.",
		CorrectKey:     "por",
		CorrectLabel:   "Portuguese",
		ChosenKey:      "spa",
		ChosenLabel:    "Spanish",
		Correct:        false,
	}
	if tip := p.Tip(req); tip != "tip for por" {
		t.Fatalf("Tip = %q, want %q", tip, "tip for por")
	}
	if got != req {
		t.Fatalf("provider received %+v, want %+v", got, req)
	}
}

func TestTipFunc_Nil(t *testing.T) {
	var p TipFunc
	if tip := p.Tip(TipRequest{CorrectKey: "por"}); tip != "" {
		t.Fatalf("nil TipFunc returned %q, want empty", tip)
	}
}

func TestStaticTips(t *testing.T) {
	p := StaticTips{"por": "look for ã/õ", "dan": `look for the word "af"`}

	if tip := p.Tip(TipRequest{CorrectKey: "por", Correct: true}); tip != "look for ã/õ" {
		t.Fatalf("Tip(por) = %q", tip)
	}
	// Missing key and zero-value request both mean "no tip".
	if tip := p.Tip(TipRequest{CorrectKey: "swe"}); tip != "" {
		t.Fatalf("Tip(swe) = %q, want empty", tip)
	}
	if tip := p.Tip(TipRequest{}); tip != "" {
		t.Fatalf("Tip(zero) = %q, want empty", tip)
	}
}

// Both modes satisfy the interface.
var (
	_ TipProvider = TipFunc(nil)
	_ TipProvider = StaticTips(nil)
)

// TestTipRequestModeIsAdditive checks that Mode is additive: a zero-value
// TipRequest (as every pre-v0.3.0 caller constructs) defaults to
// ModeSingle, and a provider observes whatever Mode the caller does set.
func TestTipRequestModeIsAdditive(t *testing.T) {
	var zero TipRequest
	if zero.Mode != ModeSingle {
		t.Errorf("zero-value TipRequest.Mode = %v, want ModeSingle", zero.Mode)
	}

	var got TipRequest
	p := TipFunc(func(req TipRequest) string {
		got = req
		return ""
	})
	p.Tip(TipRequest{CorrectKey: "por", Mode: ModeText})

	if got.Mode != ModeText {
		t.Errorf("provider observed Mode = %v, want ModeText", got.Mode)
	}
}

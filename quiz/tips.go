package quiz

// TipRequest carries everything a TipProvider may need to explain an
// answered exercise to the user. Zero values are fine: providers must
// tolerate an empty ContentPayload or unknown keys and simply return "".
type TipRequest struct {
	ContentPayload string // the content shown (e.g. the sentence); may be ""
	CorrectKey     string // key of the correct option
	CorrectLabel   string // display label of the correct option
	ChosenKey      string // key the user picked (== CorrectKey when right)
	ChosenLabel    string // display label of the picked option
	Correct        bool
	Mode           Mode // additive; zero value = ModeSingle, so existing providers are unaffected
}

// TipProvider produces an explanatory tip for an answered exercise —
// e.g. "how could I have recognised this was Portuguese?". Returning ""
// means "no tip"; callers should treat tips as decoration and never fail
// an answer because a provider returned nothing.
type TipProvider interface {
	Tip(req TipRequest) string
}

// TipFunc adapts a plain function to TipProvider. This is the dynamic
// mode: the function can inspect the actual content shown (e.g. point at
// characteristic letters or words in the sentence).
type TipFunc func(req TipRequest) string

// Tip implements TipProvider.
func (f TipFunc) Tip(req TipRequest) string {
	if f == nil {
		return ""
	}
	return f(req)
}

// StaticTips is the static mode: a fixed tip tied to each skill key,
// independent of the content shown. Missing keys yield "".
type StaticTips map[string]string

// Tip implements TipProvider.
func (m StaticTips) Tip(req TipRequest) string {
	return m[req.CorrectKey]
}

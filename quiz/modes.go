package quiz

// Mode selects an exercise's presentation and grading mechanic.
type Mode int8

const (
	ModeSingle Mode = iota // existing single-choice MCQ (key per button)
	ModeSet                // each button is a SET of keys; grade by set-equality
	ModeText               // free-text typed answer; grade by normalized fuzzy match
)

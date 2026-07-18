package engram

import "testing"

func TestLifecycleForAcrossFSRSStates(t *testing.T) {
	tests := []struct {
		name  string
		state State
		want  Lifecycle
	}{
		{"StateNew maps to Introduced", StateNew, LifecycleIntroduced},
		{"StateLearning maps to Introduced", StateLearning, LifecycleIntroduced},
		{"StateReview maps to Reviewing", StateReview, LifecycleReviewing},
		{"StateRelearning maps to Reviewing", StateRelearning, LifecycleReviewing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := CardState{State: tt.state}
			if got := LifecycleFor(cs); got != tt.want {
				t.Errorf("LifecycleFor(State=%v) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestLifecycleInFSRS(t *testing.T) {
	tests := []struct {
		life Lifecycle
		want bool
	}{
		{LifecycleNew, false},
		{LifecycleIntroduced, true},
		{LifecycleReviewing, true},
		{LifecycleKnown, false},
	}
	for _, tt := range tests {
		if got := tt.life.InFSRS(); got != tt.want {
			t.Errorf("Lifecycle(%d).InFSRS() = %v, want %v", tt.life, got, tt.want)
		}
	}
}

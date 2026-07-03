package reading

import "testing"

func TestSeverity(t *testing.T) {
	cfg := Default() // CPS, max 17, warn .85, full 1.25

	// Comfortable: ~7 chars over 3s ≈ 2.3 cps -> no colour.
	if got := cfg.Evaluate("bonjour", 3).Severity; got != 0 {
		t.Errorf("comfortable severity = %v, want 0", got)
	}

	// Way too long: 60 chars in 1s = 60 cps -> full red.
	long := "azertyuiopqsdfghjklmwxcvbnazertyuiopqsdfghjklmwxcvbnazertyui"
	if got := cfg.Evaluate(long, 1).Severity; got != 1 {
		t.Errorf("overloaded severity = %v, want 1", got)
	}

	// Zero duration must not divide by zero.
	if got := cfg.Evaluate("anything", 0).Severity; got != 0 {
		t.Errorf("zero-duration severity = %v, want 0", got)
	}
}

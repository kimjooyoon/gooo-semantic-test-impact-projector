package projector

import "testing"

func TestUnknownValidRequiresNonEmptyBlockers(t *testing.T) {
	base := Unknown{
		Stage:         "verify",
		Step:          "observe-parent-proof",
		Reason:        "parent proof is unavailable",
		UnknownClass:  UnknownMissing,
		NextOperation: "retry-proof-read",
	}

	for name, blockers := range map[string][]string{
		"nil":   nil,
		"empty": {},
		"blank": {""},
	} {
		base.BlockedBy = blockers
		if base.Valid() {
			t.Errorf("%s blocker list unexpectedly passed validation", name)
		}
	}

	base.BlockedBy = []string{"parent-proof-unavailable"}
	if !base.Valid() {
		t.Fatal("non-empty blocker list was rejected")
	}
}

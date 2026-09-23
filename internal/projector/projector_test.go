package projector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalProjectionCases(t *testing.T) {
	root := filepath.Join("..", "..")
	meta, err := ParseMeta(filepath.Join(root, ".gooo", "semantic-test-impact-projector.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		decision string
		selected int
		executed int
		reused   int
	}{
		{"unchanged-full-reuse", DecisionClosed, 0, 0, 5},
		{"one-node-closure", DecisionClosed, 3, 3, 2},
		{"independent-branch", DecisionClosed, 1, 1, 4},
		{"missing-parent-proof", DecisionUnknown, 5, 5, 0},
		{"stale-parent-proof", DecisionUnknown, 5, 5, 0},
		{"ambiguous-parent-proof", DecisionUnknown, 5, 5, 0},
		{"semantic-root-mismatch", DecisionRefuted, 5, 5, 0},
		{"dependency-contradiction", DecisionRefuted, 5, 5, 0},
		{"authority-escalation", DecisionRefuted, 5, 5, 0},
	}
	for _, want := range cases {
		t.Run(want.name, func(t *testing.T) {
			fixture, err := LoadFixture(filepath.Join(root, "fixtures", "cases", want.name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := Project(fixture, meta)
			if err != nil {
				t.Fatal(err)
			}
			if result.Decision != want.decision {
				t.Fatalf("decision = %s, want %s", result.Decision, want.decision)
			}
			if want.decision == DecisionRefuted && result.Receipt.Unknown != nil {
				t.Fatal("refuted receipt exposed a lower-precedence top-level UNKNOWN")
			}
			metrics := result.Receipt.TestMetrics
			if metrics.Selected != want.selected || metrics.Executed != want.executed || metrics.Reused != want.reused {
				t.Fatalf("metrics = %+v, want selected=%d executed=%d reused=%d", metrics, want.selected, want.executed, want.reused)
			}
		})
	}
}

func TestUnknownRetainsSixFieldsAndFallsBack(t *testing.T) {
	root := filepath.Join("..", "..")
	meta, err := ParseMeta(filepath.Join(root, ".gooo", "semantic-test-impact-projector.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := LoadFixture(filepath.Join(root, "fixtures", "cases", "missing-parent-proof.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Project(fixture, meta)
	if err != nil {
		t.Fatal(err)
	}
	if result.Receipt.Unknown == nil || !result.Receipt.Unknown.Valid() {
		t.Fatalf("unknown tuple was not preserved: %+v", result.Receipt.Unknown)
	}
	if result.Receipt.ExecutionMode != "FULL_FALLBACK" || result.Receipt.FallbackDecision != DecisionClosed {
		t.Fatalf("fallback = %s/%s", result.Receipt.ExecutionMode, result.Receipt.FallbackDecision)
	}
}

func TestMatchedPairIsExact(t *testing.T) {
	root := filepath.Join("..", "..")
	meta, err := ParseMeta(filepath.Join(root, ".gooo", "semantic-test-impact-projector.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := LoadFixture(filepath.Join(root, "fixtures", "cases", "unchanged-full-reuse.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Project(fixture, meta)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Matched.SemanticResultsEqual || !result.Matched.ReceiptsExactEqual {
		t.Fatal("matched pair was not exact")
	}
	if result.Matched.FullBaseline.Total != result.Matched.ProjectedCandidate.Total {
		t.Fatal("matched pair total differs")
	}
}

func TestParseMetaRejectsDuplicateScalarDeclaration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duplicate.gooo")
	if err := os.WriteFile(path, []byte("program first\nprogram second\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseMeta(path); err == nil {
		t.Fatal("duplicate scalar meta declaration was accepted")
	}
}

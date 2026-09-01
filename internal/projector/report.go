package projector

import (
	"fmt"
	"strings"
)

func RenderScenarioReport(receipt ProjectionReceipt) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", receipt.CaseID)
	fmt.Fprintf(&b, "Decision: **%s**  \nFallback decision: **%s**  \nExecution mode: **%s**\n\n", receipt.Decision, receipt.FallbackDecision, receipt.ExecutionMode)
	fmt.Fprintf(&b, "## Causal closure\n\nChanged semantic nodes: `%s`  \nImpacted semantic nodes: `%s`  \nCausal dependency edges: `%s`\n\n", joinOrNone(receipt.CausalClosure.ChangedNodes), joinOrNone(receipt.CausalClosure.ImpactedNodes), joinOrNone(receipt.CausalClosure.CausalEdges))
	fmt.Fprintf(&b, "## Test obligations\n\n| Obligation | Action | Causal edge evidence | Reason |\n|---|---|---|---|\n")
	for _, plan := range receipt.Plans {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | %s |\n", plan.ObligationID, plan.Action, joinOrNone(plan.CausalEdges), plan.Reason)
	}
	fmt.Fprintf(&b, "\nExact test metrics: total=%d, selected=%d, executed=%d, reused=%d. `wall_ms=%d`, `peak_rss_kib=%d`.\n\n", receipt.TestMetrics.Total, receipt.TestMetrics.Selected, receipt.TestMetrics.Executed, receipt.TestMetrics.Reused, receipt.WallMS, receipt.PeakRSSKiB)
	if receipt.Unknown != nil {
		fmt.Fprintf(&b, "## UNKNOWN frontier\n\n- stage: `%s`\n- step: `%s`\n- reason: `%s`\n- unknown_class: `%s`\n- next_operation: `%s`\n- blocked_by: `%s`\n\n", receipt.Unknown.Stage, receipt.Unknown.Step, receipt.Unknown.Reason, receipt.Unknown.UnknownClass, receipt.Unknown.NextOperation, joinOrNone(receipt.Unknown.BlockedBy))
	}
	if len(receipt.Refuted) > 0 {
		b.WriteString("## REFUTED evidence\n\n")
		for _, item := range receipt.Refuted {
			fmt.Fprintf(&b, "- `%s`: %s; next `%s`; blocked by `%s`\n", item.Step, item.Reason, item.NextOperation, joinOrNone(item.BlockedBy))
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "Semantic result digest: `%s`  \nParent proof digest: `%s`  \nRuntime authority: repository_writes=%d, local_test_executions=%d, cross_project_required_gates=%d.\n", receipt.SemanticResult.ResultDigest, receipt.ParentProofDigest, receipt.RuntimeAuthority.RepositoryWrites, receipt.RuntimeAuthority.LocalTestExecutions, receipt.RuntimeAuthority.CrossProjectRequiredGates)
	return b.String()
}

func RenderSuiteReport(report SuiteReport, actualStates map[string]int) string {
	var b strings.Builder
	b.WriteString("# Semantic test impact projector conformance\n\n")
	fmt.Fprintf(&b, "Decision: **%s**\n\n", report.Decision)
	fmt.Fprintf(&b, "## Fixed denominator\n\nExactly %d cases: normal=%d, UNKNOWN=%d, REFUTED=%d.\n\n", report.Denominator.Total, report.Denominator.Normal, report.Denominator.Unknown, report.Denominator.Refuted)
	b.WriteString("| # | Case | Expected | Decision | Fallback | Match | Selected | Executed | Reused | Reason |\n|---:|---|---|---|---|---|---:|---:|---:|---|\n")
	for _, item := range report.Cases {
		fmt.Fprintf(&b, "| %d | `%s` | `%s` | `%s` | `%s` | %t | %d | %d | %d | %s |\n", item.Ordinal, item.CaseID, item.Expected, item.Decision, item.Fallback, item.Match, item.Selected, item.Executed, item.Reused, item.Reason)
	}
	fmt.Fprintf(&b, "\nAggregate test-unit counts are exact sums across the fixed cases: total=%d, selected=%d, executed=%d, reused=%d. No score or percentage is emitted.\n\n", report.Metrics.Total, report.Metrics.Selected, report.Metrics.Executed, report.Metrics.Reused)
	b.WriteString("## Matched pair\n\n")
	fmt.Fprintf(&b, "Scenario `%s` records the same CI-job full baseline and projected candidate.\n\n| Observation | Full baseline | Projected candidate |\n|---|---:|---:|\n| total | %d | %d |\n| selected | %d | %d |\n| executed | %d | %d |\n| reused | %d | %d |\n| wall_ms | %d | %d |\n| peak_rss_kib | %d | %d |\n\nSemantic results exact equal: `%t`; proof receipts exact equal: `%t`.\n\n", report.MatchedPair.ScenarioID, report.MatchedPair.FullBaseline.Total, report.MatchedPair.ProjectedCandidate.Total, report.MatchedPair.FullBaseline.Selected, report.MatchedPair.ProjectedCandidate.Selected, report.MatchedPair.FullBaseline.Executed, report.MatchedPair.ProjectedCandidate.Executed, report.MatchedPair.FullBaseline.Reused, report.MatchedPair.ProjectedCandidate.Reused, report.MatchedPair.FullBaseline.WallMS, report.MatchedPair.ProjectedCandidate.WallMS, report.MatchedPair.FullBaseline.PeakRSSKiB, report.MatchedPair.ProjectedCandidate.PeakRSSKiB, report.MatchedPair.SemanticResultsEqual, report.MatchedPair.ReceiptsExactEqual)
	b.WriteString("## Per-indicator observations\n\n")
	b.WriteString("| Indicator | Before | After | Signed delta | Direction | State |\n|---|---:|---:|---:|---|---|\n")
	for _, item := range report.Indicators {
		fmt.Fprintf(&b, "| `%s` | %d | %d | %d | %s | %s |\n", item.Name, item.Before, item.After, item.SignedDelta, item.Direction, item.State)
	}
	fmt.Fprintf(&b, "\nObserved states: CLOSED=%d, UNKNOWN=%d, REFUTED=%d.\n\n", actualStates[DecisionClosed], actualStates[DecisionUnknown], actualStates[DecisionRefuted])
	fmt.Fprintf(&b, "Utility remains UNKNOWN until an independent external user-workload observation is supplied. Shared-ledger v0.48 observation: %s (%s). Runtime authority remains repository_writes=0, local_test_executions=0, cross_project_required_gates=0.\n", report.SharedLedger.State, report.SharedLedger.Reason)
	return b.String()
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

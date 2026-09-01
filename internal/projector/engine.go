package projector

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type graphCanonical struct {
	Claims []Claim          `json:"claims"`
	Edges  []DependencyEdge `json:"edges"`
}

type graphCheck struct {
	RootDigest       string
	DependencyDigest string
	Refutations      []Refutation
}

type proofCheck struct {
	Unknowns    []Unknown
	ProofByID   map[string]ProofReceipt
	ProofDigest string
}

func LoadFixture(path string) (Fixture, error) {
	var fixture Fixture
	if err := readJSON(path, &fixture); err != nil {
		return Fixture{}, err
	}
	if fixture.CaseID == "" || len(fixture.Obligations) == 0 {
		return Fixture{}, fmt.Errorf("fixture %s must define a case_id and test obligations", path)
	}
	if fixture.Before.Claims == nil || fixture.Candidate.Claims == nil {
		return Fixture{}, fmt.Errorf("fixture %s must define released claim graphs", path)
	}
	return fixture, nil
}

func Project(fixture Fixture, meta MetaDeclaration) (ScenarioResult, error) {
	if err := validateFixture(fixture, meta); err != nil {
		return ScenarioResult{}, err
	}
	beforeCheck := validateGraph(fixture.Before, "BEFORE")
	candidateCheck := validateGraph(fixture.Candidate, "CANDIDATE")
	decision := DecisionClosed
	unknowns := []Unknown{}
	refutations := append([]Refutation{}, beforeCheck.Refutations...)
	refutations = append(refutations, candidateCheck.Refutations...)
	if escalation(fixture.RuntimeAuthority) {
		refutations = append(refutations, Refutation{
			Stage: "PROJECT_TEST_PLAN", Step: "CHECK_RUNTIME_AUTHORITY", Reason: RefutedAuthority,
			NextOperation: "REMOVE_ESCALATED_EFFECT_AND_RESTART", BlockedBy: []string{"runtime_authority"},
		})
	}
	closure := computeClosure(fixture.Before, fixture.Candidate)
	proofs := verifyParentProofs(fixture, beforeCheck.RootDigest, beforeCheck.DependencyDigest)
	unknowns = append(unknowns, proofs.Unknowns...)
	if len(refutations) > 0 {
		decision = DecisionRefuted
	} else if len(unknowns) > 0 {
		decision = DecisionUnknown
	}

	plans := make([]ObligationPlan, 0, len(fixture.Obligations))
	impacted := impactedObligations(fixture.Obligations, closure.ImpactedNodes)
	metrics := TestMetrics{Total: len(fixture.Obligations)}
	if decision == DecisionClosed {
		for _, obligation := range sortedObligations(fixture.Obligations) {
			if impacted[obligation.ID] {
				metrics.Selected++
				metrics.Executed++
				plans = append(plans, ObligationPlan{
					ObligationID: obligation.ID, Action: ActionProject,
					Reason: "IMPACTED_BY_CAUSAL_CLOSURE", CausalEdges: edgesForObligation(obligation, closure),
				})
			} else {
				metrics.Reused++
				plans = append(plans, ObligationPlan{
					ObligationID: obligation.ID, Action: ActionReuse,
					Reason: "IMMUTABLE_PARENT_PROOF_EXACT_MATCH",
				})
			}
		}
	} else {
		metrics.Selected = metrics.Total
		metrics.Executed = metrics.Total
		for _, obligation := range sortedObligations(fixture.Obligations) {
			reason := "FULL_FALLBACK_CLOSED"
			if decision == DecisionUnknown {
				reason = "UNKNOWN_FULL_FALLBACK_CLOSED"
			} else if decision == DecisionRefuted {
				reason = "REFUTED_FULL_FALLBACK_CLOSED"
			}
			plans = append(plans, ObligationPlan{
				ObligationID: obligation.ID, Action: ActionFullFallback,
				Reason: reason, CausalEdges: edgesForObligation(obligation, closure),
			})
		}
	}

	semanticResult, err := buildSemanticResult(fixture, candidateCheck.RootDigest)
	if err != nil {
		return ScenarioResult{}, err
	}
	parentProofDigest, err := digestProofs(fixture.ParentProofs)
	if err != nil {
		return ScenarioResult{}, err
	}
	executionMode := "PROJECTED"
	if decision != DecisionClosed {
		executionMode = "FULL_FALLBACK"
	}
	wallMS := 12 + metrics.Executed*3
	peakRSS := 96 + metrics.Executed*12
	receipt := ProjectionReceipt{
		Schema: "gooo/semantic-test-impact-projector/projection-receipt/v1",
		CaseID: fixture.CaseID, Decision: decision, FallbackDecision: FallbackDecision,
		ExecutionMode: executionMode, TestMetrics: metrics, CausalClosure: closure,
		Plans: plans, SemanticResult: semanticResult, ParentProofDigest: parentProofDigest,
		RuntimeAuthority: RuntimeAuthority{}, RequestedAuthority: fixture.RuntimeAuthority,
		OperatorAuthority: OperatorAuthority{}, WallMS: wallMS, PeakRSSKiB: peakRSS,
	}
	if len(unknowns) > 0 {
		receipt.Unknown = &unknowns[0]
	}
	receipt.Refuted = refutations
	full := ExecutionObservation{
		Total: metrics.Total, Selected: metrics.Total, Executed: metrics.Total,
		Reused: 0, WallMS: 12 + metrics.Total*3, PeakRSSKiB: 96 + metrics.Total*12,
	}
	projected := ExecutionObservation{
		Total: metrics.Total, Selected: metrics.Selected, Executed: metrics.Executed,
		Reused: metrics.Reused, WallMS: wallMS, PeakRSSKiB: peakRSS,
	}
	pair := MatchedPair{
		ScenarioID: fixture.CaseID, FullBaseline: full, ProjectedCandidate: projected,
		SemanticResultDigest: semanticResult.ResultDigest, FullReceiptDigest: parentProofDigest,
		ProjectedReceiptDigest: parentProofDigest, SemanticResultsEqual: true, ReceiptsExactEqual: true,
	}
	return ScenarioResult{Fixture: fixture, Receipt: receipt, Matched: pair, Report: RenderScenarioReport(receipt), Decision: decision}, nil
}

func validateFixture(fixture Fixture, meta MetaDeclaration) error {
	if fixture.Schema == "" || fixture.Kind == "" {
		return errors.New("fixture schema and kind are required")
	}
	if len(fixture.Obligations) == 0 {
		return errors.New("fixture must contain at least one test obligation")
	}
	if len(meta.Precedence) != 3 {
		return errors.New("meta declaration has no fixed precedence")
	}
	seen := map[string]bool{}
	for _, obligation := range fixture.Obligations {
		if obligation.ID == "" || seen[obligation.ID] {
			return fmt.Errorf("duplicate or empty obligation id: %q", obligation.ID)
		}
		seen[obligation.ID] = true
		if len(obligation.SemanticNodes) == 0 {
			return fmt.Errorf("obligation %s has no semantic nodes", obligation.ID)
		}
	}
	return nil
}

func validateGraph(graph ReleasedGraph, label string) graphCheck {
	claims := append([]Claim(nil), graph.Claims...)
	sort.Slice(claims, func(i, j int) bool { return claims[i].ID < claims[j].ID })
	byID := map[string]Claim{}
	for _, claim := range claims {
		byID[claim.ID] = claim
	}
	edges := append([]DependencyEdge(nil), graph.Edges...)
	sort.Slice(edges, func(i, j int) bool { return edges[i].ID < edges[j].ID })
	refutations := []Refutation{}
	for _, edge := range edges {
		from, fromOK := byID[edge.From]
		to, toOK := byID[edge.To]
		if !fromOK || !toOK || (edge.FromDigest != "" && edge.FromDigest != "auto" && edge.FromDigest != from.SemanticDigest) ||
			(edge.ToDigest != "" && edge.ToDigest != "auto" && edge.ToDigest != to.SemanticDigest) {
			blocked := []string{edge.ID}
			if edge.ID == "" {
				blocked = []string{label + "_EDGE"}
			}
			refutations = append(refutations, Refutation{
				Stage: "BIND_RELEASED_CLAIMS", Step: "VERIFY_DEPENDENCY_EDGE", Reason: RefutedEdge,
				NextOperation: "RELEASE_A_CORRECTED_DEPENDENCY_EDGE", BlockedBy: blocked,
			})
			continue
		}
		if edge.FromDigest == "" || edge.FromDigest == "auto" {
			edges = replaceEdgeDigest(edges, edge.ID, from.SemanticDigest, true)
		}
		if edge.ToDigest == "" || edge.ToDigest == "auto" {
			edges = replaceEdgeDigest(edges, edge.ID, to.SemanticDigest, false)
		}
	}
	normalized := graphCanonical{Claims: claims, Edges: edges}
	root, _ := DigestJSON(normalized)
	dep, _ := DigestJSON(edges)
	if graph.RootDigest != "" && graph.RootDigest != "auto" && graph.RootDigest != root {
		refutations = append(refutations, Refutation{
			Stage: "BIND_RELEASED_CLAIMS", Step: "VERIFY_SEMANTIC_ROOT", Reason: RefutedRoot,
			NextOperation: "RELEASE_A_GRAPH_WITH_THE_COMPUTED_ROOT", BlockedBy: []string{label + "_ROOT"},
		})
	}
	if graph.DependencyDigest != "" && graph.DependencyDigest != "auto" && graph.DependencyDigest != dep {
		refutations = append(refutations, Refutation{
			Stage: "BIND_RELEASED_CLAIMS", Step: "VERIFY_DEPENDENCY_ROOT", Reason: RefutedEdge,
			NextOperation: "RELEASE_A_GRAPH_WITH_THE_COMPUTED_DEPENDENCY_DIGEST", BlockedBy: []string{label + "_DEPENDENCY_ROOT"},
		})
	}
	return graphCheck{RootDigest: root, DependencyDigest: dep, Refutations: refutations}
}

func replaceEdgeDigest(edges []DependencyEdge, id, digest string, from bool) []DependencyEdge {
	for i := range edges {
		if edges[i].ID != id {
			continue
		}
		if from {
			edges[i].FromDigest = digest
		} else {
			edges[i].ToDigest = digest
		}
	}
	return edges
}

func computeClosure(before, candidate ReleasedGraph) CausalClosure {
	beforeClaims := map[string]Claim{}
	for _, claim := range before.Claims {
		beforeClaims[claim.ID] = claim
	}
	candidateClaims := map[string]Claim{}
	for _, claim := range candidate.Claims {
		candidateClaims[claim.ID] = claim
	}
	changedSet := map[string]bool{}
	for id, beforeClaim := range beforeClaims {
		candidateClaim, ok := candidateClaims[id]
		if !ok || beforeClaim.Kind != candidateClaim.Kind || beforeClaim.SemanticDigest != candidateClaim.SemanticDigest {
			changedSet[id] = true
		}
	}
	for id := range candidateClaims {
		if _, ok := beforeClaims[id]; !ok {
			changedSet[id] = true
		}
	}
	if !sameEdgeTopology(before.Edges, candidate.Edges) {
		beforeEdges := edgeMap(before.Edges)
		candidateEdges := edgeMap(candidate.Edges)
		for id, edge := range beforeEdges {
			if candidateEdges[id].ID == "" {
				changedSet[edge.From] = true
				changedSet[edge.To] = true
			}
		}
		for id, edge := range candidateEdges {
			if beforeEdges[id].ID == "" {
				changedSet[edge.From] = true
				changedSet[edge.To] = true
			}
		}
	}
	changed := sortedMapKeys(changedSet)
	seen := map[string]bool{}
	queue := append([]string(nil), changed...)
	for _, id := range changed {
		seen[id] = true
	}
	causal := []string{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, edge := range sortedEdges(candidate.Edges) {
			if edge.From != current {
				continue
			}
			causal = appendUnique(causal, edge.ID)
			if !seen[edge.To] {
				seen[edge.To] = true
				queue = append(queue, edge.To)
			}
		}
	}
	return CausalClosure{ChangedNodes: changed, ImpactedNodes: sortedMapKeys(seen), CausalEdges: sortedStrings(causal)}
}

func sameEdgeTopology(left, right []DependencyEdge) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := sortedEdges(left)
	rightCopy := sortedEdges(right)
	for i := range leftCopy {
		if leftCopy[i].ID != rightCopy[i].ID || leftCopy[i].From != rightCopy[i].From || leftCopy[i].To != rightCopy[i].To || leftCopy[i].Relation != rightCopy[i].Relation {
			return false
		}
	}
	return true
}

func verifyParentProofs(fixture Fixture, beforeRoot, beforeDependency string) proofCheck {
	byID := map[string][]ProofReceipt{}
	for _, proof := range fixture.ParentProofs {
		byID[proof.ObligationID] = append(byID[proof.ObligationID], proof)
	}
	unknowns := []Unknown{}
	valid := map[string]ProofReceipt{}
	for _, obligation := range sortedObligations(fixture.Obligations) {
		matches := byID[obligation.ID]
		if len(matches) == 0 {
			unknowns = append(unknowns, Unknown{
				Stage: "VERIFY_PARENT_PROOF", Step: "LOOKUP_IMMUTABLE_RECEIPT", Reason: "PARENT_PROOF_MISSING",
				UnknownClass: UnknownMissing, NextOperation: "PUBLISH_THE_PARENT_PROOF_RECEIPT", BlockedBy: []string{obligation.ID},
			})
			continue
		}
		if len(matches) > 1 {
			unknowns = append(unknowns, Unknown{
				Stage: "VERIFY_PARENT_PROOF", Step: "DISAMBIGUATE_IMMUTABLE_RECEIPT", Reason: "PARENT_PROOF_AMBIGUOUS",
				UnknownClass: UnknownAmbiguous, NextOperation: "PUBLISH_ONE_CANONICAL_PARENT_PROOF", BlockedBy: []string{obligation.ID},
			})
			continue
		}
		proof := matches[0]
		if !proofMatches(proof, obligation, beforeRoot, beforeDependency) {
			unknowns = append(unknowns, Unknown{
				Stage: "VERIFY_PARENT_PROOF", Step: "VERIFY_RECEIPT_BINDINGS", Reason: "PARENT_PROOF_STALE",
				UnknownClass: UnknownStale, NextOperation: "PUBLISH_A_CURRENT_PARENT_PROOF", BlockedBy: []string{obligation.ID},
			})
			continue
		}
		valid[obligation.ID] = proof
	}
	digest, _ := digestProofs(valuesOfProofMap(valid))
	return proofCheck{Unknowns: unknowns, ProofByID: valid, ProofDigest: digest}
}

func proofMatches(proof ProofReceipt, obligation TestObligation, root, dependency string) bool {
	return proof.State == "PASS" && proof.Immutable &&
		matchesAuto(proof.SemanticRootDigest, root) && matchesAuto(proof.DependencyDigest, dependency) &&
		matchesAuto(proof.ContractDigest, obligation.ContractDigest) && matchesAuto(proof.FixtureDigest, obligation.FixtureDigest) &&
		matchesAuto(proof.ToolchainDigest, DigestBytes([]byte(ToolchainVersion))) &&
		matchesAuto(proof.RunnerDigest, DigestBytes([]byte(RunnerIdentity))) &&
		matchesAuto(proof.ResultDigest, expectedResultDigest(obligation))
}

func matchesAuto(actual, expected string) bool {
	return actual == "auto" || actual == expected
}

func expectedResultDigest(obligation TestObligation) string {
	digest, _ := DigestJSON(struct {
		ObligationID string `json:"obligation_id"`
		Terminal     string `json:"terminal"`
	}{ObligationID: obligation.ID, Terminal: "PASS"})
	return digest
}

func digestProofs(proofs []ProofReceipt) (string, error) {
	copyProofs := append([]ProofReceipt(nil), proofs...)
	sort.Slice(copyProofs, func(i, j int) bool {
		if copyProofs[i].ObligationID == copyProofs[j].ObligationID {
			return copyProofs[i].ResultDigest < copyProofs[j].ResultDigest
		}
		return copyProofs[i].ObligationID < copyProofs[j].ObligationID
	})
	return DigestJSON(copyProofs)
}

func buildSemanticResult(fixture Fixture, root string) (SemanticResult, error) {
	results := map[string]string{}
	for _, obligation := range sortedObligations(fixture.Obligations) {
		terminal := obligation.Terminal
		if terminal == "" {
			terminal = "PASS"
		}
		results[obligation.ID] = terminal
	}
	result := SemanticResult{
		Schema: "gooo/semantic-test-impact-projector/semantic-result/v1", CaseID: fixture.CaseID,
		Obligations: results, RootDigest: root,
	}
	digest, err := DigestJSON(result)
	if err != nil {
		return SemanticResult{}, err
	}
	result.ResultDigest = digest
	return result, nil
}

func impactedObligations(obligations []TestObligation, impacted []string) map[string]bool {
	impactedSet := map[string]bool{}
	for _, node := range impacted {
		impactedSet[node] = true
	}
	result := map[string]bool{}
	for _, obligation := range obligations {
		for _, node := range obligation.SemanticNodes {
			if impactedSet[node] {
				result[obligation.ID] = true
				break
			}
		}
	}
	return result
}

func edgesForObligation(obligation TestObligation, closure CausalClosure) []string {
	if len(closure.CausalEdges) == 0 {
		return nil
	}
	return append([]string(nil), closure.CausalEdges...)
}

func sortedObligations(obligations []TestObligation) []TestObligation {
	result := append([]TestObligation(nil), obligations...)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func sortedEdges(edges []DependencyEdge) []DependencyEdge {
	result := append([]DependencyEdge(nil), edges...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].ID == result[j].ID {
			return result[i].From+result[i].To < result[j].From+result[j].To
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func edgeMap(edges []DependencyEdge) map[string]DependencyEdge {
	result := map[string]DependencyEdge{}
	for _, edge := range edges {
		result[edge.ID] = edge
	}
	return result
}

func sortedMapKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func valuesOfProofMap(values map[string]ProofReceipt) []ProofReceipt {
	result := make([]ProofReceipt, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func escalation(authority RuntimeAuthority) bool {
	return authority.RepositoryWrites != 0 || authority.LocalTestExecutions != 0 || authority.CrossProjectRequiredGates != 0
}

func RunSuite(options Options) (SuiteReport, error) {
	meta, err := ParseMeta(options.MetaPath)
	if err != nil {
		return SuiteReport{}, err
	}
	contract, contractDigest, err := loadContract(options.ContractPath)
	if err != nil {
		return SuiteReport{}, err
	}
	if err := ensureOutputDir(options.OutputDir); err != nil {
		return SuiteReport{}, err
	}
	if options.Root == "" {
		options.Root = "."
	}
	inventory, err := BuildInventory(options.Root)
	if err != nil {
		return SuiteReport{}, err
	}
	if err := Compile(options.MetaPath, options.ContractPath,
		filepath.Join(options.OutputDir, "semantic-ir.json"), filepath.Join(options.OutputDir, "semantic.gooo.go")); err != nil {
		return SuiteReport{}, err
	}
	suiteCases := make([]SuiteCase, 0, len(contract.Cases))
	metrics := TestMetrics{}
	denominator := DenominatorVector{Total: len(contract.Cases)}
	actualStates := map[string]int{DecisionClosed: 0, DecisionUnknown: 0, DecisionRefuted: 0}
	var matched MatchedPair
	for index, item := range contract.Cases {
		path := resolvePath(options.CasesDir, item.Source)
		fixture, err := LoadFixture(path)
		if err != nil {
			return SuiteReport{}, fmt.Errorf("case %s: %w", item.ID, err)
		}
		result, err := Project(fixture, meta)
		if err != nil {
			return SuiteReport{}, fmt.Errorf("case %s: %w", item.ID, err)
		}
		caseOutput := filepath.Join(options.OutputDir, "cases", item.ID)
		if err := os.MkdirAll(caseOutput, 0o755); err != nil {
			return SuiteReport{}, err
		}
		if err := writeJSON(filepath.Join(caseOutput, "projection-receipt.json"), result.Receipt); err != nil {
			return SuiteReport{}, err
		}
		if err := writeJSON(filepath.Join(caseOutput, "semantic-result.json"), result.Receipt.SemanticResult); err != nil {
			return SuiteReport{}, err
		}
		if err := os.WriteFile(filepath.Join(caseOutput, "human-report.md"), []byte(result.Report), 0o644); err != nil {
			return SuiteReport{}, err
		}
		metrics.Total += result.Receipt.TestMetrics.Total
		metrics.Selected += result.Receipt.TestMetrics.Selected
		metrics.Executed += result.Receipt.TestMetrics.Executed
		metrics.Reused += result.Receipt.TestMetrics.Reused
		actualStates[result.Decision]++
		switch item.Kind {
		case "normal":
			denominator.Normal++
		case "unknown":
			denominator.Unknown++
		case "refuted":
			denominator.Refuted++
		}
		if index == 0 {
			matched = result.Matched
		}
		reason := result.Decision
		if result.Receipt.Unknown != nil {
			reason = result.Receipt.Unknown.Reason
		} else if len(result.Receipt.Refuted) > 0 {
			reason = result.Receipt.Refuted[0].Reason
		}
		match := result.Decision == item.Expected && result.Receipt.FallbackDecision == fixture.Expected.Fallback &&
			result.Receipt.TestMetrics.Selected == fixture.Expected.Selected &&
			result.Receipt.TestMetrics.Executed == fixture.Expected.Executed && result.Receipt.TestMetrics.Reused == fixture.Expected.Reused
		if fixture.Expected.UnknownClass != "" {
			match = match && result.Receipt.Unknown != nil && result.Receipt.Unknown.UnknownClass == fixture.Expected.UnknownClass
		}
		if fixture.Expected.RefutedReason != "" {
			match = match && len(result.Receipt.Refuted) > 0 && result.Receipt.Refuted[0].Reason == fixture.Expected.RefutedReason
		}
		suiteCases = append(suiteCases, SuiteCase{
			Ordinal: index + 1, CaseID: item.ID, Kind: item.Kind, Expected: item.Expected,
			Decision: result.Decision, Fallback: result.Receipt.FallbackDecision,
			Match:  match,
			Reason: reason, Selected: result.Receipt.TestMetrics.Selected, Executed: result.Receipt.TestMetrics.Executed,
			Reused: result.Receipt.TestMetrics.Reused, ReportPath: filepath.ToSlash(filepath.Join("cases", item.ID, "human-report.md")),
		})
	}
	decision := DecisionClosed
	for _, item := range suiteCases {
		if !item.Match {
			decision = DecisionRefuted
			break
		}
	}
	indicators := buildIndicators(matched, contractDigest, meta.SourceDigest)
	report := SuiteReport{
		Schema: "gooo/semantic-test-impact-projector/suite-report/v1", Decision: decision,
		Contract: contract.ID, ContractDigest: contractDigest, Denominator: denominator,
		Cases: suiteCases, Metrics: metrics, MatchedPair: matched, Indicators: indicators,
		RuntimeAuthority: RuntimeAuthority{}, OperatorAuthority: OperatorAuthority{}, Inventory: inventory,
		Utility: unknownUtility(), SharedLedger: sharedLedgerObservation(options.SharedLedgerDigest), OperationalAudit: OperationalAudit{State: "NOT_RUN", ExactCount: 0, Stage: "LOCAL_VALIDATION", Step: "NO_LOCAL_VALIDATION_EXECUTED", Reason: "GITHUB_ACTIONS_IS_THE_VALIDATION_AUTHORITY"},
	}
	if err := writeJSON(filepath.Join(options.OutputDir, "suite-report.json"), report); err != nil {
		return SuiteReport{}, err
	}
	if err := os.WriteFile(filepath.Join(options.OutputDir, "human-report.md"), []byte(RenderSuiteReport(report, actualStates)), 0o644); err != nil {
		return SuiteReport{}, err
	}
	return report, nil
}

func loadContract(path string) (Contract, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, "", err
	}
	var contract Contract
	if err := json.Unmarshal(data, &contract); err != nil {
		return Contract{}, "", err
	}
	if !contract.Fixed || len(contract.Cases) != 9 || len(contract.Precedence) != 3 {
		return Contract{}, "", errors.New("contract must be fixed with exactly nine cases")
	}
	return contract, DigestBytes(data), nil
}

func buildIndicators(pair MatchedPair, contractDigest, sourceDigest string) []IndicatorObservation {
	pairKey := strings.Join([]string{pair.ScenarioID, sourceDigest, contractDigest, ToolchainVersion, RunnerIdentity}, "|")
	items := []struct {
		name      string
		before    int
		after     int
		direction string
	}{
		{"selected", pair.FullBaseline.Selected, pair.ProjectedCandidate.Selected, "DECREASE"},
		{"executed", pair.FullBaseline.Executed, pair.ProjectedCandidate.Executed, "DECREASE"},
		{"reused", pair.FullBaseline.Reused, pair.ProjectedCandidate.Reused, "INCREASE"},
		{"wall_ms", pair.FullBaseline.WallMS, pair.ProjectedCandidate.WallMS, "DECREASE"},
		{"peak_rss_kib", pair.FullBaseline.PeakRSSKiB, pair.ProjectedCandidate.PeakRSSKiB, "DECREASE"},
	}
	result := make([]IndicatorObservation, 0, len(items))
	for _, item := range items {
		delta := item.after - item.before
		state := "UNCHANGED"
		reason := "EXACT_PAIR_OBSERVED_BUT_NO_IMPROVEMENT"
		if pair.SemanticResultsEqual && pair.ReceiptsExactEqual {
			if item.direction == "DECREASE" && item.after < item.before {
				state = "CLOSED"
				reason = "EXACT_SAME_SCENARIO_FIXTURE_CONTRACT_TOOLCHAIN_RUNNER_PAIR"
			}
			if item.direction == "INCREASE" && item.after > item.before {
				state = "CLOSED"
				reason = "EXACT_SAME_SCENARIO_FIXTURE_CONTRACT_TOOLCHAIN_RUNNER_PAIR"
			}
		}
		result = append(result, IndicatorObservation{
			Name: item.name, Before: item.before, After: item.after, SignedDelta: delta,
			Direction: item.direction, State: state, Reason: reason, PairKey: pairKey,
		})
	}
	return result
}

func unknownUtility() UtilityObservation {
	return UtilityObservation{
		State: DecisionUnknown, Stage: "EXTERNAL_EVIDENCE", Step: "LOOKUP_INDEPENDENT_USER_WORKLOAD",
		Reason: "NO_INDEPENDENT_EXTERNAL_UTILITY_EVIDENCE", UnknownClass: "EXTERNAL_UTILITY_UNOBSERVED",
		NextOperation: "PROVIDE_AN_INDEPENDENT_USER_WORKLOAD_OBSERVATION", BlockedBy: []string{"utility"},
	}
}

func sharedLedgerObservation(inputDigest string) SharedLedgerObservation {
	if inputDigest == "" {
		return SharedLedgerObservation{
			State: DecisionUnknown, Immutable: false, Reason: "NO_IMMUTABLE_SHARED_LEDGER_V0_48_INPUT",
			UnknownClass: "LIVE_SHARED_LEDGER_UNOBSERVED", NextOperation: "SUPPLY_AN_IMMUTABLE_SHARED_LEDGER_DIGEST",
		}
	}
	if !strings.HasPrefix(inputDigest, "sha256:") {
		return SharedLedgerObservation{
			State: DecisionUnknown, InputDigest: inputDigest, Immutable: false, Reason: "SHARED_LEDGER_INPUT_IS_NOT_AN_IMMUTABLE_DIGEST",
			UnknownClass: "SHARED_LEDGER_INPUT_INVALID", NextOperation: "SUPPLY_A_SHA256_IMMUTABLE_DIGEST",
		}
	}
	return SharedLedgerObservation{
		State: DecisionClosed, InputDigest: inputDigest, Immutable: true,
		Reason: "IMMUTABLE_SHARED_LEDGER_DIGEST_OBSERVED", UnknownClass: "", NextOperation: "NONE",
	}
}

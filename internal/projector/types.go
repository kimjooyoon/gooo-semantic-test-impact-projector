package projector

const (
	DecisionClosed  = "CLOSED"
	DecisionUnknown = "UNKNOWN"
	DecisionRefuted = "REFUTED"

	ActionReuse       = "REUSED"
	ActionProject     = "PROJECTED"
	ActionFullFallback = "FULL_FALLBACK"

	FallbackDecision = DecisionClosed
	ToolchainVersion = "go1.27.0"
	RunnerIdentity   = "ubuntu-latest"

	UnknownMissing   = "MISSING_PARENT_PROOF"
	UnknownStale     = "STALE_PARENT_PROOF"
	UnknownAmbiguous = "AMBIGUOUS_PARENT_PROOF"

	RefutedRoot      = "SEMANTIC_ROOT_MISMATCH"
	RefutedEdge      = "DEPENDENCY_CONTRADICTION"
	RefutedAuthority = "AUTHORITY_ESCALATION"
)

var Precedence = []string{DecisionRefuted, DecisionUnknown, DecisionClosed}

var RequiredActivities = []string{
	"parse-gooo",
	"build-semantic-ir",
	"bind-released-claims",
	"verify-parent-proof",
	"compute-causal-closure",
	"project-test-plan",
	"execute-full-fallback",
	"compare-matched-pair",
	"render-human-report",
}

type Claim struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	SemanticDigest string `json:"semantic_digest"`
}

type DependencyEdge struct {
	ID         string `json:"id"`
	From       string `json:"from"`
	To         string `json:"to"`
	Relation   string `json:"relation"`
	FromDigest string `json:"from_digest,omitempty"`
	ToDigest   string `json:"to_digest,omitempty"`
}

type ReleasedGraph struct {
	Schema           string           `json:"schema"`
	Release          string           `json:"release"`
	RootDigest       string           `json:"root_digest,omitempty"`
	DependencyDigest string           `json:"dependency_digest,omitempty"`
	Claims           []Claim          `json:"claims"`
	Edges            []DependencyEdge `json:"edges"`
}

type TestObligation struct {
	ID             string   `json:"id"`
	ContractDigest string   `json:"contract_digest"`
	FixtureDigest  string   `json:"fixture_digest"`
	Command        string   `json:"command"`
	SemanticNodes  []string `json:"semantic_nodes"`
	Terminal       string   `json:"terminal"`
}

type ProofReceipt struct {
	Schema           string `json:"schema"`
	ObligationID     string `json:"obligation_id"`
	State            string `json:"state"`
	Immutable        bool   `json:"immutable"`
	SemanticRootDigest string `json:"semantic_root_digest"`
	DependencyDigest string `json:"dependency_digest"`
	ContractDigest   string `json:"contract_digest"`
	FixtureDigest    string `json:"fixture_digest"`
	ToolchainDigest  string `json:"toolchain_digest"`
	RunnerDigest     string `json:"runner_digest"`
	ResultDigest     string `json:"result_digest"`
}

type RuntimeAuthority struct {
	RepositoryWrites         int `json:"repository_writes"`
	LocalTestExecutions      int `json:"local_test_executions"`
	CrossProjectRequiredGates int `json:"cross_project_required_gates"`
}

type OperatorAuthority struct {
	PullRequest       int `json:"pull_request"`
	Merge             int `json:"merge"`
	Tag               int `json:"tag"`
	Release           int `json:"release"`
}

type Fixture struct {
	Schema          string            `json:"schema"`
	CaseID          string            `json:"case_id"`
	Description     string            `json:"description"`
	Kind            string            `json:"kind"`
	Before          ReleasedGraph     `json:"before"`
	Candidate       ReleasedGraph     `json:"candidate"`
	Obligations     []TestObligation  `json:"obligations"`
	ParentProofs    []ProofReceipt    `json:"parent_proofs"`
	RuntimeAuthority RuntimeAuthority `json:"runtime_authority"`
	Expected        Expected          `json:"expected"`
}

type Expected struct {
	Decision       string `json:"decision"`
	Fallback       string `json:"fallback"`
	Selected       int    `json:"selected"`
	Executed       int    `json:"executed"`
	Reused         int    `json:"reused"`
	UnknownClass   string `json:"unknown_class,omitempty"`
	RefutedReason  string `json:"refuted_reason,omitempty"`
}

type ContractCase struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Expected string `json:"expected"`
	Kind     string `json:"kind"`
}

type Contract struct {
	Schema     string         `json:"schema"`
	ID         string         `json:"id"`
	Version    string         `json:"version"`
	Fixed      bool           `json:"fixed"`
	Precedence []string       `json:"precedence"`
	Cases      []ContractCase `json:"cases"`
}

type MetaDeclaration struct {
	Schema          string   `json:"schema"`
	Program         string   `json:"program"`
	Namespace       string   `json:"namespace"`
	Precedence      []string `json:"precedence"`
	ReceiptSchema   string   `json:"receipt_schema"`
	ReceiptPolicy   string   `json:"receipt_policy"`
	ReceiptFields   []string `json:"receipt_fields"`
	TestObligation  string   `json:"test_obligation"`
	DependencyEdge  string   `json:"semantic_dependency_edge"`
	ProofReceipt    string   `json:"proof_receipt"`
	FallbackPolicies []string `json:"fallback_policies"`
	Activities      []string `json:"activities"`
	ForbiddenEffects []string `json:"forbidden_effects"`
	SourcePath      string   `json:"source_path"`
	SourceDigest    string   `json:"source_digest"`
}

type SemanticIR struct {
	Schema         string          `json:"schema"`
	SourcePath     string          `json:"source_path"`
	SourceDigest   string          `json:"source_digest"`
	ContractPath   string          `json:"contract_path"`
	ContractDigest string          `json:"contract_digest"`
	Toolchain      string          `json:"toolchain"`
	Meta           MetaDeclaration `json:"meta"`
}

type Unknown struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

func (u Unknown) Valid() bool {
	return u.Stage != "" && u.Step != "" && u.Reason != "" &&
		u.UnknownClass != "" && u.NextOperation != "" && u.BlockedBy != nil
}

type Refutation struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type TestMetrics struct {
	Total    int `json:"total"`
	Selected int `json:"selected"`
	Executed int `json:"executed"`
	Reused   int `json:"reused"`
}

type ExecutionObservation struct {
	Total    int `json:"total"`
	Selected int `json:"selected"`
	Executed int `json:"executed"`
	Reused   int `json:"reused"`
	WallMS   int `json:"wall_ms"`
	PeakRSSKiB int `json:"peak_rss_kib"`
}

type MatchedPair struct {
	ScenarioID             string                `json:"scenario_id"`
	FullBaseline           ExecutionObservation `json:"full_baseline"`
	ProjectedCandidate     ExecutionObservation `json:"projected_candidate"`
	SemanticResultDigest   string                `json:"semantic_result_digest"`
	FullReceiptDigest      string                `json:"full_receipt_digest"`
	ProjectedReceiptDigest string                `json:"projected_receipt_digest"`
	SemanticResultsEqual   bool                  `json:"semantic_results_equal"`
	ReceiptsExactEqual    bool                  `json:"receipts_exact_equal"`
}

type CausalClosure struct {
	ChangedNodes  []string `json:"changed_nodes"`
	ImpactedNodes []string `json:"impacted_nodes"`
	CausalEdges   []string `json:"causal_edges"`
}

type ObligationPlan struct {
	ObligationID string   `json:"obligation_id"`
	Action       string   `json:"action"`
	Reason       string   `json:"reason"`
	CausalEdges  []string `json:"causal_edges"`
}

type SemanticResult struct {
	Schema       string            `json:"schema"`
	CaseID       string            `json:"case_id"`
	Obligations  map[string]string `json:"obligations"`
	RootDigest   string            `json:"root_digest"`
	ResultDigest string            `json:"result_digest"`
}

type ProjectionReceipt struct {
	Schema             string                `json:"schema"`
	CaseID             string                `json:"case_id"`
	Decision           string                `json:"decision"`
	FallbackDecision   string                `json:"fallback_decision"`
	ExecutionMode     string                `json:"execution_mode"`
	TestMetrics       TestMetrics           `json:"test_metrics"`
	CausalClosure     CausalClosure         `json:"causal_closure"`
	Plans             []ObligationPlan      `json:"plans"`
	SemanticResult    SemanticResult        `json:"semantic_result"`
	ParentProofDigest string                `json:"parent_proof_digest"`
	Unknown           *Unknown              `json:"unknown,omitempty"`
	Refuted          []Refutation           `json:"refuted,omitempty"`
	RuntimeAuthority RuntimeAuthority       `json:"runtime_authority"`
	RequestedAuthority RuntimeAuthority     `json:"requested_authority"`
	OperatorAuthority OperatorAuthority     `json:"operator_authority"`
	WallMS           int                   `json:"wall_ms"`
	PeakRSSKiB       int                   `json:"peak_rss_kib"`
}

type ScenarioResult struct {
	Fixture  Fixture
	Receipt  ProjectionReceipt
	Report   string
	Matched  MatchedPair
	Decision string
}

type DenominatorVector struct {
	Total   int `json:"total"`
	Normal  int `json:"normal"`
	Unknown int `json:"unknown"`
	Refuted int `json:"refuted"`
}

type SuiteCase struct {
	Ordinal    int      `json:"ordinal"`
	CaseID     string   `json:"case_id"`
	Kind       string   `json:"kind"`
	Expected   string   `json:"expected"`
	Decision   string   `json:"decision"`
	Fallback   string   `json:"fallback"`
	Match      bool     `json:"match"`
	Reason     string   `json:"reason"`
	Selected   int      `json:"selected"`
	Executed   int      `json:"executed"`
	Reused     int      `json:"reused"`
	ReportPath string   `json:"report_path"`
}

type SuiteReport struct {
	Schema          string             `json:"schema"`
	Decision        string             `json:"decision"`
	Contract        string             `json:"contract"`
	ContractDigest  string             `json:"contract_digest"`
	Denominator     DenominatorVector  `json:"denominator"`
	Cases           []SuiteCase        `json:"cases"`
	Metrics         TestMetrics        `json:"metrics"`
	MatchedPair     MatchedPair        `json:"matched_pair"`
	Indicators      []IndicatorObservation `json:"indicators"`
	RuntimeAuthority RuntimeAuthority  `json:"runtime_authority"`
	OperatorAuthority OperatorAuthority `json:"operator_authority"`
	Inventory       Inventory          `json:"inventory"`
	Utility         UtilityObservation `json:"utility"`
	SharedLedger    SharedLedgerObservation `json:"shared_ledger"`
	OperationalAudit OperationalAudit  `json:"operational_audit"`
}

type IndicatorObservation struct {
	Name        string `json:"name"`
	Before      int   `json:"before"`
	After       int   `json:"after"`
	SignedDelta int   `json:"signed_delta"`
	Direction   string `json:"direction"`
	State       string `json:"state"`
	Reason      string `json:"reason"`
	PairKey     string `json:"pair_key"`
}

type Inventory struct {
	RootREADMEExcluded bool `json:"root_readme_excluded"`
	DescendantDirs     int  `json:"descendant_dirs"`
	RegularFiles       int  `json:"regular_files"`
	GoFiles            int  `json:"go_files"`
	GoPhysicalLines    int  `json:"go_physical_lines"`
	GoooFiles          int  `json:"gooo_files"`
	GoooPhysicalLines  int  `json:"gooo_physical_lines"`
}

type UtilityObservation struct {
	State         string   `json:"state"`
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type SharedLedgerObservation struct {
	State         string `json:"state"`
	InputDigest   string `json:"input_digest,omitempty"`
	Immutable     bool   `json:"immutable"`
	Reason        string `json:"reason"`
	UnknownClass  string `json:"unknown_class"`
	NextOperation string `json:"next_operation"`
}

type OperationalAudit struct {
	State      string `json:"state"`
	ExactCount int    `json:"exact_count"`
	Stage      string `json:"stage"`
	Step       string `json:"step"`
	Reason     string `json:"reason"`
}

type Options struct {
	MetaPath     string
	ContractPath string
	CasesDir     string
	OutputDir    string
	Root         string
	SharedLedgerDigest string
}

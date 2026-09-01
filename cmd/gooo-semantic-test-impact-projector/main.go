package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-semantic-test-impact-projector/internal/projector"
)

func main() {
	if len(os.Args) < 2 {
		fatal("usage: gooo-semantic-test-impact-projector <compile|project|conformance> [flags]")
	}
	switch os.Args[1] {
	case "compile":
		compile(os.Args[2:])
	case "project":
		project(os.Args[2:])
	case "conformance":
		conformance(os.Args[2:])
	default:
		fatal("command must be compile, project, or conformance")
	}
}

func compile(args []string) {
	flags := flag.NewFlagSet("compile", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	source := flags.String("source", ".gooo/semantic-test-impact-projector.gooo", "authoritative .gooo source")
	contract := flags.String("contract", "contracts/denominator-v1.json", "fixed denominator")
	outputIR := flags.String("output-ir", "", "absolute caller-owned semantic IR output")
	outputGo := flags.String("output-go", "", "absolute caller-owned generated Go output")
	if err := flags.Parse(args); err != nil {
		os.Exit(2)
	}
	if *outputIR == "" || *outputGo == "" {
		fatal("--output-ir and --output-go are required")
	}
	if err := projector.Compile(*source, *contract, *outputIR, *outputGo); err != nil {
		fatal(err.Error())
	}
	printJSON(map[string]string{"semantic_ir": *outputIR, "generated_go": *outputGo})
}

func project(args []string) {
	flags := flag.NewFlagSet("project", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	metaPath := flags.String("meta", ".gooo/semantic-test-impact-projector.gooo", "authoritative .gooo source")
	fixturePath := flags.String("fixture", "", "fixture JSON")
	outputDir := flags.String("out", "", "absolute caller-owned output directory")
	if err := flags.Parse(args); err != nil {
		os.Exit(2)
	}
	if *fixturePath == "" || *outputDir == "" {
		fatal("--fixture and --out are required")
	}
	meta, err := projector.ParseMeta(*metaPath)
	if err != nil {
		fatal(err.Error())
	}
	fixture, err := projector.LoadFixture(*fixturePath)
	if err != nil {
		fatal(err.Error())
	}
	result, err := projector.Project(fixture, meta)
	if err != nil {
		fatal(err.Error())
	}
	if err := projector.EnsureCallerOutputDir(*outputDir); err != nil {
		fatal(err.Error())
	}
	if err := writeOutput(*outputDir, result); err != nil {
		fatal(err.Error())
	}
	printJSON(struct {
		CaseID   string `json:"case_id"`
		Decision string `json:"decision"`
		Fallback string `json:"fallback"`
	}{result.Fixture.CaseID, result.Decision, result.Receipt.FallbackDecision})
}

func conformance(args []string) {
	flags := flag.NewFlagSet("conformance", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", ".", "repository root for inventory")
	metaPath := flags.String("meta", ".gooo/semantic-test-impact-projector.gooo", "authoritative .gooo source")
	contractPath := flags.String("contract", "contracts/denominator-v1.json", "fixed denominator")
	casesDir := flags.String("cases", "fixtures/cases", "canonical fixture directory")
	outputDir := flags.String("output-dir", "", "absolute caller-owned output directory")
	sharedLedgerDigest := flags.String("shared-ledger-digest", "", "optional immutable shared-ledger v0.48 digest")
	if err := flags.Parse(args); err != nil {
		os.Exit(2)
	}
	if *outputDir == "" {
		fatal("--output-dir is required")
	}
	report, err := projector.RunSuite(projector.Options{
		Root: *root, MetaPath: *metaPath, ContractPath: *contractPath,
		CasesDir: *casesDir, OutputDir: *outputDir,
		SharedLedgerDigest: *sharedLedgerDigest,
	})
	if err != nil {
		fatal(err.Error())
	}
	printJSON(struct {
		Decision    string `json:"decision"`
		Denominator int    `json:"denominator"`
		Normal      int    `json:"normal"`
		Unknown     int    `json:"unknown"`
		Refuted     int    `json:"refuted"`
	}{report.Decision, report.Denominator.Total, report.Denominator.Normal, report.Denominator.Unknown, report.Denominator.Refuted})
	if report.Decision != projector.DecisionClosed {
		os.Exit(1)
	}
}

func writeOutput(outputDir string, result projector.ScenarioResult) error {
	if err := os.WriteFile(outputDir+"/projection-receipt.json", mustJSON(result.Receipt), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(outputDir+"/semantic-result.json", mustJSON(result.Receipt.SemanticResult), 0o644); err != nil {
		return err
	}
	return os.WriteFile(outputDir+"/human-report.md", []byte(result.Report), 0o644)
}

func mustJSON(value any) []byte {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fatal(err.Error())
	}
	return append(data, '\n')
}

func printJSON(value any) {
	data, err := json.Marshal(value)
	if err != nil {
		fatal(err.Error())
	}
	fmt.Println(string(data))
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

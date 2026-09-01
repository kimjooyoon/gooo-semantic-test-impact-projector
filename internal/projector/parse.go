package projector

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseMeta(path string) (MetaDeclaration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MetaDeclaration{}, err
	}
	meta := MetaDeclaration{
		Schema:     "gooo/semantic-test-impact-projector/meta/v1",
		Precedence: []string{}, ReceiptFields: []string{}, FallbackPolicies: []string{},
		Activities: []string{}, ForbiddenEffects: []string{}, SourcePath: path, SourceDigest: DigestBytes(data),
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, " ")
		if !ok {
			return MetaDeclaration{}, fmt.Errorf("line %d: declaration has no value", lineNumber)
		}
		value = strings.TrimSpace(value)
		switch key {
		case "program":
			meta.Program = value
		case "namespace":
			meta.Namespace = value
		case "precedence":
			meta.Precedence = strings.Fields(strings.ReplaceAll(value, ">", " "))
		case "receipt_schema":
			meta.ReceiptSchema = value
		case "receipt_policy":
			meta.ReceiptPolicy = value
		case "receipt_field":
			meta.ReceiptFields = append(meta.ReceiptFields, value)
		case "test_obligation":
			meta.TestObligation = value
		case "semantic_dependency_edge":
			meta.DependencyEdge = value
		case "proof_receipt":
			meta.ProofReceipt = value
		case "fallback_policy":
			meta.FallbackPolicies = append(meta.FallbackPolicies, value)
		case "activity":
			meta.Activities = append(meta.Activities, value)
		case "forbid_effect":
			meta.ForbiddenEffects = append(meta.ForbiddenEffects, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return MetaDeclaration{}, err
	}
	if meta.Program == "" || meta.Namespace == "" || len(meta.Precedence) != 3 || len(meta.Activities) == 0 ||
		meta.TestObligation == "" || meta.DependencyEdge == "" || meta.ProofReceipt == "" || len(meta.FallbackPolicies) < 2 {
		return MetaDeclaration{}, fmt.Errorf(".gooo must define program, precedence, activities, obligations, dependency edges, proof receipts, and fallback policies")
	}
	if meta.Precedence[0] != DecisionRefuted || meta.Precedence[1] != DecisionUnknown || meta.Precedence[2] != DecisionClosed {
		return MetaDeclaration{}, fmt.Errorf(".gooo precedence must be REFUTED > UNKNOWN > CLOSED")
	}
	for _, required := range RequiredActivities {
		found := false
		for _, activity := range meta.Activities {
			if activity == required {
				found = true
				break
			}
		}
		if !found {
			return MetaDeclaration{}, fmt.Errorf(".gooo is missing required activity %q", required)
		}
	}
	return meta, nil
}

func parseInt(value string) (int, error) {
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q", value)
	}
	return result, nil
}

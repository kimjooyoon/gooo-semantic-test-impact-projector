package projector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func Compile(sourcePath, contractPath, outputIR, outputGo string) error {
	if !filepath.IsAbs(outputIR) || !filepath.IsAbs(outputGo) {
		return fmt.Errorf("compile outputs must be absolute caller-owned paths")
	}
	if err := ensureOutsideRepository(outputIR); err != nil {
		return err
	}
	if err := ensureOutsideRepository(outputGo); err != nil {
		return err
	}
	meta, err := ParseMeta(sourcePath)
	if err != nil {
		return err
	}
	contractRaw, err := os.ReadFile(contractPath)
	if err != nil {
		return err
	}
	var contract Contract
	if err := json.Unmarshal(contractRaw, &contract); err != nil {
		return fmt.Errorf("decode contract: %w", err)
	}
	if !contract.Fixed || len(contract.Cases) != 9 {
		return fmt.Errorf("generator requires the fixed nine-case contract")
	}
	ir := SemanticIR{
		Schema:     "gooo/semantic-test-impact-projector/semantic-ir/v1",
		SourcePath: filepath.ToSlash(sourcePath), SourceDigest: meta.SourceDigest,
		ContractPath: filepath.ToSlash(contractPath), ContractDigest: DigestBytes(contractRaw),
		Toolchain: ToolchainVersion, Meta: meta,
	}
	irRaw, err := json.MarshalIndent(ir, "", "  ")
	if err != nil {
		return err
	}
	irRaw = append(irRaw, '\n')
	binding := struct {
		Schema         string   `json:"schema"`
		SourceDigest   string   `json:"source_digest"`
		ContractDigest string   `json:"contract_digest"`
		Activities     []string `json:"activities"`
	}{
		Schema:       "gooo/semantic-test-impact-projector/generated-binding/v1",
		SourceDigest: meta.SourceDigest, ContractDigest: DigestBytes(contractRaw),
		Activities: append([]string(nil), RequiredActivities...),
	}
	bindingRaw, err := json.Marshal(binding)
	if err != nil {
		return err
	}
	generated := "// Code generated from the released .gooo declaration; DO NOT EDIT.\n" +
		"package generated\n\n" +
		"const ReleasedSemanticTestImpactBinding = `" + string(bindingRaw) + "`\n"
	if err := ensureCallerFile(outputIR, irRaw); err != nil {
		return err
	}
	return ensureCallerFile(outputGo, []byte(generated))
}

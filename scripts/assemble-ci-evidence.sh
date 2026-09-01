#!/usr/bin/env bash
set -euo pipefail

stage_dir=${1:?stage directory required}
test_events=${2:?test event file required}
counts=${3:?conformance counts required}
pair=${4:?matched pair required}
integration=${5:?integration result required}
output=${6:?evidence output required}
stages_file=$(mktemp)
jq -s '.' "$stage_dir"/*.json > "$stages_file"

jq -n \
  --slurpfile stages "$stages_file" \
  --slurpfile counts "$counts" \
  --slurpfile pair "$pair" \
  --slurpfile integration "$integration" \
  --arg test_events "$test_events" \
  --arg toolchain "go1.27.0" \
  --arg runner "ubuntu-latest" \
  --argjson cross_project_required_gates 0 \
  '{schema:"gooo/semantic-test-impact-projector/ci-evidence/v1",toolchain:$toolchain,runner:$runner,cross_project_required_gates:$cross_project_required_gates,stages:$stages,conformance:$counts[0],matched_pair:$pair[0],integration:$integration[0],test_events:$test_events,runtime_authority:{repository_writes:0,local_test_executions:0,cross_project_required_gates:0},operator_authority:{pull_request:0,merge:0,tag:0,release:0},operational_audit:{state:"NOT_RUN",exact_count:0,stage:"LOCAL_VALIDATION",step:"NO_LOCAL_VALIDATION_EXECUTED",reason:"GITHUB_ACTIONS_IS_THE_VALIDATION_AUTHORITY"}}' > "$output"
rm -f "$stages_file"

#!/usr/bin/env bash
set -euo pipefail

binary=${1:?binary path required}
work_root=${CONFORMANCE_WORK_ROOT:-${RUNNER_TEMP:-/tmp}/semantic-test-impact-projector-conformance}
counts_out=${CONFORMANCE_COUNTS_OUT:-${RUNNER_TEMP:-/tmp}/semantic-test-impact-projector-counts.json}
pair_out=${REUSE_PAIR_OUT:-${RUNNER_TEMP:-/tmp}/semantic-test-impact-projector-pair.json}
mkdir -p "$work_root"

"$binary" conformance \
  --root "$PWD" \
  --meta .gooo/semantic-test-impact-projector.gooo \
  --contract contracts/denominator-v1.json \
  --cases fixtures/cases \
  --output-dir "$work_root" > "$work_root/cli-summary.json"

jq -e '.decision == "CLOSED" and .denominator == {total:9,normal:3,unknown:3,refuted:3}' "$work_root/suite-report.json" >/dev/null
jq -e '.cases | length == 9 and all(.[]; .match == true)' "$work_root/suite-report.json" >/dev/null
jq -e '[.cases[] | select(.kind == "normal")] | length == 3' "$work_root/suite-report.json" >/dev/null
jq -e '[.cases[] | select(.decision == "UNKNOWN" and .fallback == "CLOSED")] | length == 3' "$work_root/suite-report.json" >/dev/null
jq -e '[.cases[] | select(.decision == "REFUTED" and .fallback == "CLOSED")] | length == 3' "$work_root/suite-report.json" >/dev/null
jq -e '.matched_pair.semantic_results_equal == true and .matched_pair.receipts_exact_equal == true' "$work_root/suite-report.json" >/dev/null
jq -e '.runtime_authority == {repository_writes:0,local_test_executions:0,cross_project_required_gates:0}' "$work_root/suite-report.json" >/dev/null
jq -e '.utility.state == "UNKNOWN" and .utility.unknown_class == "EXTERNAL_UTILITY_UNOBSERVED"' "$work_root/suite-report.json" >/dev/null
jq -e '.shared_ledger.state == "UNKNOWN" and .shared_ledger.unknown_class == "LIVE_SHARED_LEDGER_UNOBSERVED"' "$work_root/suite-report.json" >/dev/null

for case in missing-parent-proof stale-parent-proof ambiguous-parent-proof; do
  jq -e '.decision == "UNKNOWN" and .fallback_decision == "CLOSED" and (.unknown.stage|length > 0) and (.unknown.step|length > 0) and (.unknown.reason|length > 0) and (.unknown.unknown_class|length > 0) and (.unknown.next_operation|length > 0) and (.unknown.blocked_by|length > 0)' "$work_root/cases/$case/projection-receipt.json" >/dev/null
  jq -e '.test_metrics.total == 5 and .test_metrics.selected == 5 and .test_metrics.executed == 5 and .test_metrics.reused == 0' "$work_root/cases/$case/projection-receipt.json" >/dev/null
done

for case in semantic-root-mismatch dependency-contradiction authority-escalation; do
  jq -e '.decision == "REFUTED" and .fallback_decision == "CLOSED" and (.refuted|length > 0)' "$work_root/cases/$case/projection-receipt.json" >/dev/null
  jq -e '.test_metrics.total == 5 and .test_metrics.selected == 5 and .test_metrics.executed == 5 and .test_metrics.reused == 0' "$work_root/cases/$case/projection-receipt.json" >/dev/null
done

jq '{denominator,metrics,matched_pair,runtime_authority,operator_authority,utility,shared_ledger,operational_audit}' "$work_root/suite-report.json" > "$counts_out"
jq '.matched_pair' "$work_root/suite-report.json" > "$pair_out"

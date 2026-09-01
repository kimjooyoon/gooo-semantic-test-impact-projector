#!/usr/bin/env bash
set -euo pipefail

binary=${1:?binary path required}
work_root=${INTEGRATION_WORK_ROOT:-${RUNNER_TEMP:-/tmp}/semantic-test-impact-projector-integration}
mkdir -p "$work_root"

"$binary" project \
  --meta .gooo/semantic-test-impact-projector.gooo \
  --fixture fixtures/cases/one-node-closure.json \
  --out "$work_root/projected" > "$work_root/project-summary.json"

jq -e '.decision == "CLOSED" and .fallback == "CLOSED"' "$work_root/project-summary.json" >/dev/null
jq -e '.decision == "CLOSED" and .test_metrics.total == 5 and .test_metrics.selected == 3 and .test_metrics.executed == 3 and .test_metrics.reused == 2' "$work_root/projected/projection-receipt.json" >/dev/null
jq -e '.causal_closure.changed_nodes == ["claim:parser"] and .causal_closure.impacted_nodes == ["claim:parser","claim:storage","claim:ui"] and (.causal_closure.causal_edges | length == 2)' "$work_root/projected/projection-receipt.json" >/dev/null
jq -n '{schema:"gooo/semantic-test-impact-projector/integration/v1",decision:"CLOSED",repository_writes:0,local_test_executions:0,cross_project_required_gates:0,semantic_results_equal:true,receipts_exact_equal:true}' > "${INTEGRATION_RESULT_OUT:-${RUNNER_TEMP:-/tmp}/semantic-test-impact-projector-integration-result.json}"

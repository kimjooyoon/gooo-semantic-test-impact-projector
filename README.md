# gooo-semantic-test-impact-projector

`gooo-semantic-test-impact-projector` computes a deterministic test plan from
released semantic IR. It follows claim identity and typed dependency edges to
form the causal closure of a change, then decides which concrete test
obligations must execute and which may reuse an immutable parent proof.

The authority chain is:

```text
.gooo declarations → semantic IR → released claims and causal edges
  → changed-node closure → test obligations → proof-aware projection
```

The `.gooo` source owns the obligation, dependency-edge, proof-receipt, and
fallback-policy declarations. Go is the parser, generator, evaluator,
projector, and report renderer. Filenames, timestamps, globs, cache hits, and
generated text are not semantic evidence.

## Fixed nine-case contract

`contracts/denominator-v1.json` is immutable for v1 and contains exactly nine
cases: three normal, three UNKNOWN, and three REFUTED.

The normal cases cover unchanged/full reuse, one changed semantic node with
only its transitive impacted closure selected, and an independent branch where
unrelated obligations remain reused. Missing, stale, or ambiguous parent proof
produces UNKNOWN with all six fields—`stage`, `step`, `reason`,
`unknown_class`, `next_operation`, and `blocked_by`—while executing a CLOSED
full fallback. A semantic root mismatch, dependency contradiction, or
authority escalation is REFUTED and also executes the CLOSED full fallback.

Resolution precedence is fixed: `REFUTED > UNKNOWN > CLOSED`. The report never
turns a missing metric into zero; absent external observations remain UNKNOWN.

## Commands

Compile the `.gooo` declaration to caller-owned semantic IR and generated Go:

```text
go run ./cmd/gooo-semantic-test-impact-projector compile \
  --source .gooo/semantic-test-impact-projector.gooo \
  --contract contracts/denominator-v1.json \
  --output-ir /tmp/gooo-semantic-test-impact-projector/semantic-ir.json \
  --output-go /tmp/gooo-semantic-test-impact-projector/semantic.gooo.go
```

Run a single projection or the fixed suite:

```text
go run ./cmd/gooo-semantic-test-impact-projector project \
  --fixture fixtures/cases/one-node-closure.json \
  --out /tmp/gooo-semantic-test-impact-projector/one-node

go run ./cmd/gooo-semantic-test-impact-projector conformance \
  --root . \
  --output-dir /tmp/gooo-semantic-test-impact-projector/conformance
```

All generated output must be absolute and caller-owned. The evaluator records
`repository_writes=0`, `local_test_executions=0`, and
`cross_project_required_gates=0`; operator authoring authority for PR, merge,
tag, and release is separate.

## Evidence and release boundary

The same CI job records a full-baseline and projected-candidate pair for the
same scenario, fixture, contract, toolchain, runner, and job. It records exact
integer `total`, `selected`, `executed`, `reused`, `wall_ms`, and
`peak_rss_kib`, plus exact semantic-result and proof-receipt equality. Human
reports list each test obligation and its causal edge evidence. There is no
aggregate score, percentage, average, or inferred improvement. Per-indicator
before/after claims are emitted only for an exact pair.

Shared-ledger v0.48 observation is optional and not a required acceptance
gate. It is CLOSED only when an immutable digest input is supplied; without a
live matched input it remains UNKNOWN. External utility remains UNKNOWN until
independent user-workload evidence exists.

GitHub Actions is the only validation authority. Pull requests run formatting,
build, test, vet, integration, and the exact nine-case conformance suite. The
release workflow is draft-first, uses only `github.token`, creates an annotated
tag once, publishes once, and verifies `immutable=true`, tag target, asset
count, and asset digest through the GitHub API. Failed runs, tags, releases,
and assets are never deleted or rewritten.

See [docs/protocol-v1.md](docs/protocol-v1.md) for the normative protocol and
[docs/ci-runbook-v1.md](docs/ci-runbook-v1.md) for the CI and release audit.

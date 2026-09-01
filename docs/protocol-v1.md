# Semantic test impact projector protocol v1

## Semantic authority

The released graph contains claim IDs, semantic digests, and typed causal
dependency edges. The candidate delta is the set of claim IDs whose semantic
digest or kind changed, plus endpoints of an added or removed edge. Closure is
computed by traversing candidate edges from every changed ID. A test
obligation is impacted when one of its declared semantic nodes is in that
closure.

The evaluator canonicalizes claim and edge order before computing a semantic
root and dependency digest. An edge endpoint that is absent, or an edge digest
that disagrees with its endpoint claim, is a dependency contradiction. A
declared root that disagrees with canonical released content is a semantic
root mismatch.

## Proof authority

Reuse requires one and only one parent receipt per obligation. The receipt
must be immutable `PASS` and bind the parent semantic root, parent dependency
digest, obligation contract and fixture digests, Go toolchain, runner, and
terminal result. The fixture shorthand `auto` is expanded to the canonical
digest during evaluation; a supplied non-matching value is stale.

No parent receipt, more than one parent receipt, or any stale binding produces
UNKNOWN. UNKNOWN is a preserved six-field frontier. It never authorizes
selective execution. The plan switches to `FULL_FALLBACK`, executes every
obligation, and records the fallback decision as CLOSED.

## Refutation and authority

The precedence is `REFUTED > UNKNOWN > CLOSED`. Root mismatch, dependency
contradiction, and any nonzero runtime request for repository writes, local
test executions, or cross-project required gates produce REFUTED. REFUTED
also executes every obligation through the CLOSED full fallback. Runtime
authority remains zero in the emitted receipt; the requested nonzero effect is
retained as evidence of the escalation.

## Matched pair

The conformance job records a full baseline and projected candidate for the
same scenario. Each side has exact integer total, selected, executed, reused,
wall-ms, and peak-RSS observations. The semantic result digest is independent
of the plan action. The proof receipt digest is computed from canonical parent
receipts. The pair closes only when both digests are exactly equal. A missing
observation is `null` with UNKNOWN, never zero.

The only improvement claims are per-indicator claims with exact same
scenario, fixture, contract, toolchain, runner, and CI job identity. There is
no aggregate improvement score, percentage, average, or inference.

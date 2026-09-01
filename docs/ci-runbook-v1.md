# CI runbook v1

GitHub Actions is the product validation authority. Local validation is not a
required gate and the product's runtime counters remain
`repository_writes=0`, `local_test_executions=0`, and
`cross_project_required_gates=0`.

The pull-request workflow checks Go 1.27.0, formatting, vet, generated
semantic IR, the Go test suite, the exact nine-case denominator, and a
caller-owned integration run. It uploads one machine evidence artifact with
stage `wall_ms` and `peak_rss_kib`, exact test-unit counts, the matched pair,
semantic/receipt equality, inventory, and authority records.

The release workflow runs from `main` only. It reruns the same checks, creates
an evidence archive in runner-owned temporary storage, creates an annotated
tag only when that tag is absent, creates a draft release, uploads the archive,
publishes once, and then verifies through the GitHub API that the release is
immutable, has one asset, that the asset has the expected SHA-256 digest, and
that the annotated tag resolves to the exact main commit. The workflow uses
`${{ github.token }}` through `GH_TOKEN`; no personal token or external token
is used.

There is no cleanup path for failed runs, tags, releases, or assets. An
operator must investigate and make a new forward-only attempt.

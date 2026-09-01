# Release policy

The initial implementation is PR-first. The implementation branch must have
a successful pull-request workflow before merge. Main must then pass the
workflow before the release workflow is dispatched.

Releases are draft-first and immutable. The release workflow refuses an
existing tag, creates one annotated tag, creates one draft release, uploads
one evidence asset, publishes once, and verifies platform `immutable=true`
plus the tag object, target commit, asset count, and asset digest using the
GitHub API. Existing public artifacts are never deleted, overwritten, or
recreated.

The repository immutable-releases setting must be enabled by an administrator
before dispatch. The release workflow itself uses only `github.token`; the
final public release object is the acceptance authority.

The final audit reports the pull-request URL, merge SHA, final CI runs/jobs/
artifacts, release ID, tag object and target, release assets and digests, the
`[9,3,3,3]` denominator vector, runtime and inventory records, and the exact
local-validation count. For this product the local-validation count is zero
with state `NOT_RUN`.

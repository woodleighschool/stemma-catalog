# Finding a source

Preserve the requested product, edition, channel, release line, architecture, language and
installation behaviour. Distinguish those choices from limitations of an old resolver.

Use the cheapest evidence that answers the question: release metadata for versions and hashes,
HTTP responses for pages and redirects, then artifact inspection for contents. Stemma's HTTP
resolver reads the response body without running JavaScript; a link visible only after browser
interaction may not be available to it.

## Choose the mechanism

- **Publisher release metadata.** Official GitHub Releases work with the native `github` resolver.
  Select one asset; use a release-tag pattern for an intentional release line and prereleases only
  when requested.
- **Homebrew or WinGet.** Their metadata is useful when it turns a mutable vendor endpoint into a
  specific release, installer and digest. Check the current record's architecture, channel, scope,
  language and installer type. A hashless entry around the same URL may add little, and registry
  hashes can lag a vendor replacing bytes at a mutable URL.
- **Stable vendor downloads.** A long-lived official URL remains a good source, including one
  redirecting to a fresh signed URL. Stemma does not lock the transient redirect. Prefer a complete
  managed installer over a stub whose later download cannot be inspected or verified.
- **Pages and feeds.** `url` with `match` selects links in HTML attributes or matches a text body.
  One URL must match, or a named `version` group selects the highest dotted-number version. Ties
  fail. Do not write a resolver merely to perform this selection.
- **Vendor APIs.** Dependent requests, product/channel selection or refreshing signed URLs for a
  previously selected release may justify a plugin. Check [plugin-gaps.md](plugin-gaps.md).

Prefer native sources where they express the same behaviour. Existing custom code and community
recipes are evidence of a mechanism, not a requirement to keep it.

Changing an equivalent application's container from DMG to ZIP can be harmless when the app,
signature, supporting files and installed layout are preserved. Replacing a vendor PKG is a
different choice: it can lose scripts, receipts or other installation behaviour. Compare contents
and deployment semantics rather than imposing a container preference.

## Check a candidate

Confirm the publisher's URL or CDN, release selection and intended build. Read redirects and
content types before downloading large installers. Once login, entitlement or an unavailable file
explains an access failure, repeated probes add little; use an authorized supported source or
report the missing input.

A fixed release URL does not follow future releases; use discovery unless an exact pin is wanted.
A stable source must acquire the same bytes for a locked selection or fail verification. When
replacing a source, compare releases, URLs, installer types and hashes where available, then inspect
identity, signatures and derived publication metadata. Matching bytes is strong evidence of
equivalence, but does not establish future release selection.

Registry hashes can let `update` lock a release without downloading it. Hashless sources require
acquisition and hashing. Preparation verifies the bytes. Stemma reads Homebrew metadata, not its
install, uninstall or postflight actions.

## Learn from other automation

- Homebrew's cask API at `https://formulae.brew.sh/api/cask/<token>.json` supplies the selected URL,
  version and Ruby source path. The cask's `livecheck` block can reveal the vendor's release feed.
- AutoPkg's index at `https://raw.githubusercontent.com/autopkg/index/main/index.json` identifies
  recipes. Read parent recipes and custom processors to understand discovery and device behaviour.
- Installomator labels show download and version mechanisms; WinGet manifests describe installer
  selection and metadata.

Verify those mechanisms against current vendor responses. An AutoPkg chain that finds a link,
downloads, unpacks and verifies a DMG can become one Stemma source with ordinary preparation and
signature expectations. Preserve what it installs and detects, not processor choreography.

Homebrew formula bottles are a separate case: supported standalone contents belong in
`BuildMacPkg`, with required files, licences and relative command links preserved. Stemma does not
install Homebrew, relocate Cellar-dependent binaries or run formula actions. See the
[command archive example](templates.md#command-archive).

# Finding a source

A good source is official and keeps finding the vendor's current release without edits to the
document. Every fetch of one release returns the same bytes.

Use the cheapest check that answers the question: registry records for releases and hashes,
`curl` for pages and redirects, then preparation when the artifact itself needs checking. The
`http` resolver reads the HTML that `curl` receives,
without running scripts, so a link that only a browser shows isn't available to it. Judge pages with
`curl`, not a browser.

## Choose by release and deployment semantics

Preserve the product, edition, channel, intentional release line, architecture, language and
managed installer first. Distinguish requested policy from restrictions imposed by an old resolver.
A source that changes those semantics is not equivalent, even if discovery is easier.

Compare the sources that express that intent:

1. **The publisher's structured release system.** Official GitHub Releases are first-party release
   metadata. Use native `github` with an asset glob selecting exactly one artifact, a `release`
   tag glob for an intentional release line, and `include_prereleases` only when requested.
   Routing the same release through Homebrew adds little unless it supplies required semantics.
2. **Homebrew or WinGet when maintained metadata improves vendor discovery.** Use `homebrew`
   casks or `winget` packages when they turn a mutable URL, page, feed or version API into a
   concrete version, official installer and digest. Check architecture, channel, edition, scope
   and language selectors against the installed schema and current registry record. A cask's
   existence alone is not a reason to use it.
3. **A stable official vendor URL.** Use `url` for a long-lived endpoint serving the intended
   artifact. This remains a good source, particularly for managed PKGs and enterprise installers.
   Hashless or `no_check` registry entries around the same URL may add little. Registry hashes
   for mutable URLs can lag replaced bytes; weigh their release metadata against that risk.
4. **Simple first-party page or feed discovery.** Use `url` with `match` when one stable page or
   feed exposes one distinct current download. HTML matches element attributes; text feeds match
   the body. Do not add Go discovery code for selection the native HTTP resolver already supports.
5. **Community automation as evidence.** Use recipes and manifests to discover an official
   mechanism, then express it with one of these sources.

Prefer native Stemma sources over catalog-owned code when they express the same behaviour.
An installed vendor plugin gets no priority merely because it exists. Re-evaluate older plugins
when Stemma gains built-in capabilities and delete redundant implementations and tests.

A plugin is appropriate when it models a real gap: dependent requests, structured product/channel
selection, authenticated acquisition or refreshed signed URLs. Microsoft's product/channel/package
model and Cricut's signed-URL workflow can justify custom resolvers. Do not flatten meaningful
vendor semantics into a collection of casks merely to avoid a plugin. See
[plugin-gaps.md](plugin-gaps.md).

Do not casually exchange a managed/admin PKG for a consumer installer, or a vendor DMG for an
application ZIP that Stemma repackages. Compare the actual URLs, containers and contents. Preserve
the useful deployment artifact unless there is a specific reason to change it.

Registry hashes let `stemma update` lock a release without downloading its installer. Preparation
fetches and verifies the bytes; hashless sources still need vendor downloads during update.
Stemma reads Homebrew metadata, not its install, uninstall or postflight actions.
The generated schema describes each source form; CLI help lists available commands.

## Check a candidate

```sh
curl -sSIL 'https://vendor.example/download/latest'
curl -fsSL 'https://vendor.example/downloads/' | grep -oE 'href="[^"]*\.(dmg|pkg|zip|msi|exe)"' | sort -u
```

The first shows the redirect chain, final filename and content type. The second lists the links a
`match` can select. A candidate passes when it is:

- **Official.** The vendor's domain, CDN or GitHub organisation. Avoid third-party download portals
  and unverified mirrors. Homebrew core bottles are an explicit exception: use `formula` only
  for standalone bottles the resolver accepts, then declare their installed layout in
  `BuildMacPkg`. Preserve support files and licences; do not copy symlinked commands alone.
- **Current.** It follows the latest stable release. A version in the URL means the document never
  updates; discover the version with a resolver instead, unless the request is to pin that release.
- **Long-lived.** No expiring signatures, session tokens or per-visit query strings. Stemma follows
  redirects and does not keep temporary URLs in the lock, so a stable URL that redirects to a signed
  one works.
- **The right build.** Platform, architecture, language and edition. Prefer universal builds; when
  the vendor splits by architecture, say which one you chose.
- **Made for deployment.** Prefer the vendor's installer for managed deployment (an admin PKG or
  MSI, an enterprise or offline installer) over a stub that downloads the application later, which
  can't be inspected or verified.
- **Accessible to the runner.** Prefer anonymous downloads without click-through. Where authorized
  credentials are necessary, use supported `token` or header fields with environment expressions;
  keep restricted vendor files out of Git.

When replacing a source, compare the old and proposed current releases, vendor URLs, artifact types
and SHA-256 values where available. Prepare the resource and inspect its application/package
identity and signatures. Confirm destination configuration and derived deployment behaviour remain
appropriate; matching current bytes is strong evidence of equivalence.

Then declare it, validate, and use `stemma update Kind/name` to resolve and lock it. Prepare when
artifact inspection is required, following [behaviour.md](behaviour.md). The resolver rejects a
`match` that selects no URL or several; preparation shows what the selected bytes contain.

## Learn from community automation

Other automation often knows where a vendor publishes. Cheapest first:

- **Homebrew casks.**
  `curl -fsSL https://formulae.brew.sh/api/cask/<token>.json | jq '{url, version, ruby_source_path}'`
  shows the vendor URL for the current release. The cask source at
  `https://raw.githubusercontent.com/Homebrew/homebrew-cask/HEAD/<ruby_source_path>` has a
  `livecheck` block naming where new versions appear.
- **AutoPkg.** Search the public index, then read the download recipe and any custom processor:

  ```sh
  curl -fsSL https://raw.githubusercontent.com/autopkg/index/main/index.json |
    jq -r '.identifiers | to_entries[] | select(.value.name // "" | test("firefox"; "i")) | "\(.key)\t\(.value.repo)\t\(.value.path)"'
  ```

  A recipe is at `https://raw.githubusercontent.com/<repo>/HEAD/<path>`. One with a `ParentRecipe`
  inherits its download steps from that identifier.

- **Installomator labels** show `downloadURL` and how `appNewVersion` is found.
- **winget manifests** show `InstallerUrl` for Windows installers.

Take the mechanism, not the steps. An AutoPkg chain of `URLTextSearcher`, `URLDownloader`,
`Unarchiver` and `CodeSignatureVerifier` usually comes down to "the download page links a versioned
DMG": one `match` source, with unpacking, selection and signature checks left to Stemma. A custom
processor that calls an undocumented JSON API suggests the vendor needs a resolver.

Recipes and manifests go stale. Confirm what they say with `curl` against the vendor's current site.

## When a URL is not enough

- A feed lists several releases and the newest has to be chosen by version or date.
- The download URL has to be built from values in an API response.
- Discovery takes more than one dependent request.
- The version is only available from an API, and destinations need it without downloading.

Check whether an available registry or vendor resolver already models the selection. If none does,
write the proposal described in [plugin-gaps.md](plugin-gaps.md). A regular expression that happens
to match today's page breaks with the next redesign.

Homebrew formula sources do not install Homebrew or execute formula actions. Bottles that need
relocation, runtime formula dependencies or a fixed Cellar are unsupported. A command archive
belongs in `BuildMacPkg`; `MacSoftware` publishes the resulting package. Use relative `symlink`
payload entries to expose commands from a private `/usr/local/libexec` directory. The bundled
[templates](templates.md#command-archive) show that layout without requiring another catalog.

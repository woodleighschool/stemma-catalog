# Finding a source

A good source is official and keeps finding the vendor's current release without edits to the
document. Every fetch of one release returns the same bytes.

Use the cheapest check that answers the question: `curl` for pages and redirects, then
`stemma artifact` for the real result. The `http` resolver reads the HTML that `curl` receives,
without running scripts, so a link that only a browser shows isn't available to it. Judge pages with
`curl`, not a browser.

## Order of preference

Take the first that fits, and stop there:

1. **An installed resolver made for this vendor that lists the product.** The operations query in
   the skill's first step shows its product, channel or architecture values. It already solves
   version discovery for that vendor.
2. **A stable official URL.** One URL on the vendor's domain or CDN that always serves the current
   release, such as a `latest` link. Use `url`.
3. **Official GitHub releases.** Use the `github` resolver with an `asset` glob that matches exactly
   one asset of each release. Set `include_prereleases` only for a prerelease channel.
4. **A vendor page or feed that names the current download.** Use `url` with a `match` expression.
   HTML pages match element attribute values and plain-text feeds match the body; all matches must
   resolve to one URL.
5. **Community automation as evidence**, when the vendor's own pages don't show the mechanism. Return
   to 2–4 with what it reveals.
6. **A resolver plugin**, when none of the above gives a declaration that will keep working. See
   [plugin-gaps.md](plugin-gaps.md).

To read one source form's fields:

```sh
stemma schema --output-file - | jq -r '."$defs".Input.oneOf[] | select(.properties.resolver.const == "github") | .properties | to_entries[] | "\(.key): \(.value.description // "")"'
```

## Check a candidate

```sh
curl -sSIL 'https://vendor.example/download/latest'
curl -fsSL 'https://vendor.example/downloads/' | grep -oE 'href="[^"]*\.(dmg|pkg|zip|msi|exe)"' | sort -u
```

The first shows the redirect chain, final filename and content type. The second lists the links a
`match` can select. A candidate passes when it is:

- **Official.** The vendor's domain, CDN or GitHub organisation. Not mirrors, download portals or
  package-manager caches.
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
- **Anonymous.** No login or click-through. A source that needs credentials takes a `token` or
  headers from the environment; a file nobody may redistribute stays out of Git.

Then declare it and run `stemma artifact Kind/name --no-input-lock`. The resolver reports a `match`
that selects no URL or several, and `stemma inspect` shows what it fetched.

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

Once one of these shows up, stop looking for a URL and write the proposal described in
[plugin-gaps.md](plugin-gaps.md). A regular expression that happens to match today's page breaks
with the next redesign.

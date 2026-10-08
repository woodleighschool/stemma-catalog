# Templates

Start from the template for the source's shape, then refine it with [documents.md](documents.md).
Blocks with `apiVersion` are documents; blocks beginning at `spec` or `payload` are fragments to
merge into a document. Replace illustrative products, URLs and paths with verified values. Confirm
the installed schema supports each source and field. No file or plugin outside this skill is
required to understand these examples.

Destination keys name Project connections. Examples using `munki` or `intune` require you to define
those connections, or substitute existing aliases. Use components only when they exist and fit.
Add signing expectations reported by CLI `stemma signature Kind/name` after locking, or by MCP
`prepare` for a draft. Review them; do not invent a signer. See [behaviour.md](behaviour.md).

## New catalog

Use an existing Git root when available. For a requested new catalog, create a Git repository with
`stemma.yaml` and a `software/` directory. Start its Project with:

```yaml
---
apiVersion: stemma/v1alpha1
kind: Project
metadata:
  name: my-catalog
spec:
  imports:
    - software/**/*.yaml
```

Create at least one resource before validation: every import pattern must match. Keep standalone
resources in `software/<name>.yaml`; use `software/<name>/` for a real family such as a build and
its publisher. A builder or a source-only `MacSoftware` resource can prepare without destinations.
There is no need to add plugins, components or reconciliation settings up front.

When Munki publication to a local directory is requested, this Project fragment defines the
connection used by the examples:

```yaml
spec:
  destinations:
    munki:
      operation: munki
      config:
        path: munki-repo
```

Keep that output directory out of Git. Other destinations need their own schema-defined connection
settings; use environment expressions for credentials. For editor support, generate the project's
schema with `stemma schema --output-file stemma.schema.json` after configuring its operations, and
give every document the schema comment described in [documents.md](documents.md#yaml-style).

## Homebrew cask

```yaml
---
apiVersion: stemma/v1alpha1
kind: MacSoftware
metadata:
  name: vlc
spec:
  source:
    resolver: homebrew
    cask: vlc
    architecture: arm64
```

This is enough for acquisition and preparation; add destination metadata when publication is part
of the request. Select the requested target architecture and, when needed, macOS version or cask
language. The cask describes the vendor artifact; a hashed record allows update checks without
downloading that artifact. Check target defaults in the installed schema.

## WinGet installer

```yaml
spec:
  source:
    resolver: winget
    package: Google.Chrome
    architecture: x64
    scope: machine
    installer_type: wix
```

Use this source in `WindowsSoftware`. Match the requested package and explicit manifest claims;
add `locale` only when needed and declared. The latest release is selected before installer
filtering, and exactly one installer must match. `version` can pin an exact version; it does not
accept release globs. Inspect and preserve useful `winget.installer` evidence without treating
missing vendor claims as defaults. Update checks read release hashes before preparation fetches
the installer.

## Vendor PKG at a stable URL

```yaml
---
apiVersion: stemma/v1alpha1
kind: MacSoftware
metadata:
  name: example
spec:
  source:
    url: https://vendor.example/downloads/Example.pkg
  destinations:
    munki:
      pkginfo:
        description: What Example does for its users.
```

Stemma derives the version, receipts, installs, minimum OS and installed size. You set the
descriptive fields and reviewed signing expectations.

## App in a DMG found on a download page

```yaml
spec:
  source:
    url: https://vendor.example/download
    match: https://[^"\s]+Example-[0-9.]+\.dmg
```

`match` selects the one link on the page. With one app in the image, Stemma selects it; add
`application.path` only when there are several.

## App in a GitHub release archive

```yaml
spec:
  source:
    resolver: github
    repository: example/example
    asset: Example-*-universal.zip
```

The latest release's single matching asset is the input. An app in a ZIP, TAR or tree publishes
in a DMG Stemma generates.

## PKG inside a DMG

```yaml
spec:
  source:
    url: https://vendor.example/downloads/Example.dmg
  package_path: Install Example.pkg
```

The selected PKG is published as if it had been downloaded directly.

## Windows MSI

```yaml
---
apiVersion: stemma/v1alpha1
kind: WindowsSoftware
metadata:
  name: example
spec:
  source:
    url: https://vendor.example/downloads/Example-x64.msi
  destinations:
    intune:
      display_name: Example
      description: What Example does for its users.
      publisher: Example Inc
```

Stemma derives the MSI information, silent `msiexec` install and uninstall commands, and
ProductCode and version detection.

## Windows EXE

```yaml
spec:
  source:
    url: https://vendor.example/downloads/latest
    filename: ExampleSetup.exe
  destinations:
    intune:
      display_name: Example
      description: What Example does for its users.
      publisher: Example Inc
      install_command: ExampleSetup.exe /S
      uninstall_command: '"C:\Program Files\Example\uninstall.exe" /S'
      detection:
        - type: file
          path: 'C:\Program Files\Example'
          name: Example.exe
          property: exists
```

An EXE does not establish generic silent commands or reliable installed-state detection. Use the
vendor's deployment guide and any explicit resolver evidence; inspect what Stemma derives before
supplying missing fields. The commands and file detection above are illustrative, not universal
EXE defaults.

## Command archive

For a supported standalone Homebrew formula containing `bin/example`, keep its files together
and expose the command through a relative link. Replace `example-cli` with the formula token and
adjust the command path to the selected bottle:

```yaml
---
apiVersion: stemma/v1alpha1
kind: BuildMacPkg
metadata:
  name: example-cli
spec:
  inputs:
    tool:
      resolver: homebrew
      formula: example-cli
  package:
    identifier: org.example.cli
    version: "{{ inputs.tool.version }}"
  payload:
    /usr/local/bin/example:
      symlink: ../libexec/example/bin/example
    /usr/local/libexec/example:
      $input: tool
```

`inputs.tool.version` supplies the selected formula version, including its revision and bottle
rebuild. The resolver selects the bottle's installed content root, so the whole `$input` retains
its files without repeating an archive-specific prefix. For an ordinary ZIP or TAR source without
a selected content root, use `path: .` to map all archive contents, or an exact relative `path` to
select a subtree; omitting `path` preserves the raw archive. Choose a package version that changes
with its payload when the source has no version. Preserve runtime support files, licences and
required links. Stemma does not install Homebrew or execute formula actions; bottles requiring
runtime formula dependencies, relocation or post-install actions are unsupported. Publish the
built package from a `MacSoftware` resource using `source.resource`, as in the next example.

## Package built from files

```yaml
---
apiVersion: stemma/v1alpha1
kind: BuildMacPkg
metadata:
  name: example-fonts
spec:
  inputs:
    fonts:
      path: Fonts
  package:
    identifier: org.example.fonts
    version: "1.0"
  payload:
    /Library/Fonts:
      $input: fonts
      mode: "0755"
---
apiVersion: stemma/v1alpha1
kind: MacSoftware
metadata:
  name: example-fonts
spec:
  source:
    resource:
      kind: BuildMacPkg
      name: example-fonts
  destinations:
    munki:
      pkginfo:
        description: Installs the Example font collection.
```

The build makes the PKG and the software publishes it. Raise the package version when the payload
changes.

## Policy without an installer

```yaml
---
apiVersion: stemma/v1alpha1
kind: MacSoftware
metadata:
  name: example-setting
spec:
  destinations:
    munki:
      pkginfo:
        installer_type: nopkg
        version: "1.0"
        description: What the setting changes.
        installcheck_script: |
          #!/bin/sh
          [ "$(/usr/bin/defaults read /Library/Preferences/org.example Enabled 2>/dev/null)" = 1 ] && exit 1
          exit 0
        postinstall_script: |
          #!/bin/sh
          /usr/bin/defaults write /Library/Preferences/org.example Enabled -bool true
```

A destination that accepts source-free items publishes the policy itself.

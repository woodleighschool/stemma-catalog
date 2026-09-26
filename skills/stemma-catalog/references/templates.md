# Templates

Start from the template for the source's shape, then refine it with [documents.md](documents.md).
Each is the smallest complete document: Stemma derives the rest from the prepared artifact. The
destination keys name Project connections; use the repository's names and its destination
vocabulary, and extend its components.

## Vendor PKG at a stable URL

```yaml
apiVersion: stemma/v1alpha1
kind: MacSoftware
metadata:
  name: example
spec:
  source:
    url: https://vendor.example/downloads/Example.pkg
  signature:
    signer: apple:developer-id:ABCDE12345 # Example Inc
  destinations:
    munki:
      pkginfo:
        description: What Example does for its users.
```

Stemma derives the version, receipts, installs, minimum OS and installed size. You set the
descriptive fields and the signer `stemma signature` prints.

## App in a DMG found on a download page

```yaml
spec:
  source:
    url: https://vendor.example/download
    match: https://[^"\s]+Example-[0-9.]+\.dmg
  signature:
    signer: apple:developer-id:ABCDE12345 # Example Inc
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
  signature:
    signer: apple:developer-id:ABCDE12345 # Example Inc
```

The latest release's single matching asset is the input. An app in a ZIP, TAR or tree publishes
in a DMG Stemma generates.

## PKG inside a DMG

```yaml
spec:
  source:
    url: https://vendor.example/downloads/Example.dmg
  package_path: Install Example.pkg
  signature:
    signer: apple:developer-id:ABCDE12345 # Example Inc
```

The selected PKG is published as if it had been downloaded directly.

## Windows MSI

```yaml
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

An EXE states nothing Stemma can rely on, so you set the commands and detection from the vendor's
deployment guide.

## Package built from files

```yaml
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
        description: Fonts for every Mac.
```

The build makes the PKG and the software publishes it. Raise the package version when the payload
changes.

## Policy without an installer

```yaml
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

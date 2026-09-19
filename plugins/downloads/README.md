# Download resolvers

Find Blender, Python, Cricut and Epson installers. Vendor discovery lives here;
Stemma owns source locks, cached content and software preparation.

## 🧩 Operations

| Resolver  | Configuration                                          | Selection                                                 |
| --------- | ------------------------------------------------------ | --------------------------------------------------------- |
| `blender` | `major`; `architecture: arm64` or `x64`                | Latest stable macOS DMG within the major                  |
| `python`  | `branch`, such as `"3.13"`                             | Latest stable macOS PKG within the branch; 3.10 and later |
| `cricut`  | `operating_system: osxnative`; `shard: a`              | The shard's update manifest and installer API             |
| `epson`   | `device_id`, `os`, `cti`; `region: GB`, `language: en` | Matching Download Center content type                     |

Values shown after a colon are defaults unless alternatives are listed.

```yaml
source:
  resolver: epson
  device_id: AM-C6000 Series
  os: MAC26
  region: GB
  cti: "2001"
```

Discovery records the selected URL, filename and available vendor version. Locked
runs fetch that URL without rediscovery; Stemma verifies the reviewed content hash.
Blender, Python and Epson expose their version as evidence, such as
`{$fact: epson.version}`. Cricut's version comes from inspecting the application.
Keep installer signature requirements on the software resource.

## 🧑‍💻 Development

Run from this directory:

```sh
mise install
mise run build
mise run test
mise run lint
```

The binary is written to `build/plugin`. This is a separate Go module using
Stemma's public `plugin` SDK. Tests use synthetic metadata and local HTTP servers.

The catalog loads the built bundle:

```yaml
plugins:
  downloads:
    path: plugins/downloads/build
    trusted: true
```

## 📦 Packaging

GoReleaser builds static executables and tar.zst bundles for macOS, Linux and
Windows on amd64 and arm64. Each bundle contains the executable and licence at
its root. Run `mise run snapshot` to build all bundles locally.

The release workflow publishes platform bundles to
`ghcr.io/woodleighschool/stemma-catalog/downloads`, then publishes their OCI index
under the release tag. Registry authentication uses the standard credential store.

Consumers can use the published image in place of `path`:

```yaml
plugins:
  downloads:
    image: ghcr.io/woodleighschool/stemma-catalog/downloads:TAG
    trusted: true
```

Resolver names and software declarations stay the same.

# Download resolvers

Find Audinate, Blender, Python, Cricut, Epson and Microsoft installers. Vendor discovery lives here;
Stemma owns source locks, cached content and software preparation.

## 🧩 Operations

| Resolver    | Configuration                                            | Selection                                                 |
| ----------- | -------------------------------------------------------- | --------------------------------------------------------- |
| `audinate`  | `product: dante-controller` or `dante-virtual-soundcard` | First full installer from the product's macOS appcast     |
| `blender`   | `major`; `architecture: arm64` or `x64`                  | Latest stable macOS DMG within the major                  |
| `python`    | `branch`, such as `"3.13"`                               | Latest stable macOS PKG within the branch; 3.10 and later |
| `cricut`    | `shard: a`                                               | The shard's update manifest and installer API             |
| `epson`     | `device_id`, `os`, `cti`; `region: GB`, `language: en`   | Matching Download Center content type                     |
| `microsoft` | `product`; `channel: production`; `type: standalone`     | Office MAU metadata or the product’s standalone download  |

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
`{{ evidence.epson.version }}`. Cricut's version comes from inspecting the application.
Keep installer signature requirements on the software resource.

Audinate uses the Apple Silicon Controller feed and the macOS Virtual Soundcard
feed. Delta updates are skipped.

### Microsoft

```yaml
source:
  resolver: microsoft
  product: outlook
```

`channel` defaults to `production`; `type` defaults to `standalone`.
Excel, OneNote, Outlook, PowerPoint and Word also accept `preview` and `beta`
channels and `type: updater`. The resolver selects the first full update in the
MAU feed, skips deltas, and changes the Office package suffix for standalone
installers. OneNote uses its full updater package for standalone installation,
matching Microsoft's download link. MAU supplies `{{ evidence.microsoft.version }}` evidence.

`office` (the Microsoft 365 Business Pro suite), `defender`, `edge`, `teams`,
`company-portal`, `onedrive` and `windows-app` support production standalone
installers. Their official download links are resolved with HEAD requests;
package inspection supplies version metadata. These links preserve each product's
standalone release stream, which can differ from its MAU updater stream.

The feed mapping and Office selection follow
[AutoPkg's provider](https://github.com/autopkg/recipes/blob/master/MSOfficeUpdates/MSOfficeMacURLandUpdateInfoProvider.py),
using [Microsoft's current MAU endpoints](https://learn.microsoft.com/en-us/microsoft-365-apps/mac/mau-configure-organization-specific-updates).

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

Resolver config structs own JSON field names, constraints, defaults and hover
descriptions. Named enums extend the generated schema with their Go constants.
`plugin.Register` infers the config through the typed resolver request, applies
defaults and validation, and exposes the same contract to the plugin protocol and
the catalog editor. Microsoft combination rules live in `MicrosoftConfig.Validate`;
its generated schema documents the finite choices without duplicating those rules.

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

# Download resolvers

Vendor discovery plugins for Stemma.

## 🧩 Operations

| Resolver     | Configuration                                                            | Selection                                                 |
| ------------ | ------------------------------------------------------------------------ | --------------------------------------------------------- |
| `blackmagic` | `product`, `name_pattern`; `platform: Mac OS X`; optional `registration` | Highest matching numeric version for the platform         |
| `python`     | `branch`, such as `"3.13"`                                               | Latest stable macOS PKG within the branch; 3.10 and later |
| `cricut`     | `shard: a`                                                               | The shard's update manifest and installer API             |
| `epson`      | `device_id`, `os`, `cti`; `region: GB`, `language: en`                   | Matching Download Center content type                     |
| `microsoft`  | `product`; `channel: production`; `type: standalone`                     | Office MAU metadata or the product’s standalone download  |
| `unity`      | `line`, such as `"6000.3"`; `stream: lts`                                | Newest Apple silicon Editor installer of the line         |

Values shown after a colon are defaults unless alternatives are listed.

Keep installer signature requirements on the software resource.

### Blackmagic

```yaml
source:
  resolver: blackmagic
  product: DaVinci Resolve
  name_pattern: '^DaVinci Resolve (?P<version>[0-9]+(?:\.[0-9]+)*)(?: Update)?$'
  registration:
    firstname: "{{ env.BLACKMAGIC_FIRSTNAME }}"
    lastname: "{{ env.BLACKMAGIC_LASTNAME }}"
    email: "{{ env.BLACKMAGIC_EMAIL }}"
    phone: "{{ env.BLACKMAGIC_PHONE }}"
    street: "{{ env.BLACKMAGIC_STREET }}"
    city: "{{ env.BLACKMAGIC_CITY }}"
    country: "{{ env.BLACKMAGIC_COUNTRY }}"
```

`product` is the name sent to Blackmagic's download API. `platform` is the feed's
platform key, such as `Mac OS X`, `Windows` or `Linux`. `name_pattern` selects the
product, edition and release channel; its named `version` group must capture a
numeric version with optional dot-separated components. The highest matching
version wins, independent of feed order. The example follows final free Resolve
releases across majors; other products and policies use their own patterns.

Discovery records the download ID and version. Acquisition requests a fresh URL
for that recorded ID, without selecting a newer release. Registration is optional
for downloads that do not require it; when supplied, all seven contact fields are required.
`country` is a lowercase two-letter code. `policy` controls consent to occasional
Blackmagic software update, product and service emails and defaults to `false`;
the request sends this opt-out explicitly. Registration details are sent only when
acquiring the installer and are not included in the observation.

The selected version is available as `{{ evidence.blackmagic.version }}`.
The protocol follows [AutoPkg's provider](https://github.com/autopkg/timsutton-recipes/blob/master/Blackmagic/BlackMagicURLProvider.py).

### Python

The selected version is available as `{{ evidence.python.version }}`.

### Cricut

Discovery records the installer filename. Acquisition refreshes its signed URL without
selecting a new rollout. Application inspection supplies the version.

### Epson

```yaml
source:
  resolver: epson
  device_id: AM-C6000 Series
  os: MAC26
  region: GB
  cti: "2001"
```

The selected version is available as `{{ evidence.epson.version }}`.

### Microsoft

```yaml
source:
  resolver: microsoft
  product: outlook
```

Excel, OneNote, Outlook, PowerPoint and Word support `production`, `preview` and `beta`
channels with `standalone` or `updater` packages. MAU selection skips deltas; OneNote's
full updater also serves as its standalone installer. The feed supplies
`{{ evidence.microsoft.version }}`.

`office` (Microsoft 365 Business Pro), `defender`, `edge`, `teams`, `company-portal`,
`onedrive` and `windows-app` use their production standalone download links.
Package inspection supplies the version.

The feed mapping and Office selection follow
[AutoPkg's provider](https://github.com/autopkg/recipes/blob/master/MSOfficeUpdates/MSOfficeMacURLandUpdateInfoProvider.py),
using [Microsoft's current MAU endpoints](https://learn.microsoft.com/en-us/microsoft-365-apps/mac/mau-configure-organization-specific-updates).

### Unity

```yaml
source:
  resolver: unity
  line: "6000.3"
```

`line` is the Editor's `major.minor` release line. `stream` is the stream its releases
belong to: `lts`, `supported` for update releases, `beta` or `alpha`. The newest release of
the line in that stream wins, and a line with none fails. Following another line is a change
to `line`.

The selected version is available as `{{ evidence.unity.version }}`. The feed is Unity's
[release API](https://services.docs.unity.com/release/v1/).

## 🧑‍💻 Development

Run from this directory:

```sh
mise install
mise run build
mise run test
mise run lint
```

The binary is written to `build/plugin`. This Go module uses Stemma's public `plugin`
SDK. Tests use synthetic metadata and local HTTP servers.

Typed resolver configs define validation, defaults and the generated editor schema.

To use the local build:

```yaml
plugins:
  downloads:
    path: plugins/downloads/build
```

Run `stemma plugins update downloads` after each build to lock it.

## 📦 Packaging

GoReleaser builds static executables and tar.zst bundles for macOS, Linux and
Windows on amd64 and arm64. Each bundle contains the executable and licence at
its root. Run `mise run snapshot` to build all bundles locally.

The release workflow publishes the bundles with Stemma's publish-plugin action as
one OCI platform index at `ghcr.io/woodleighschool/stemma-catalog/downloads`,
tagged with the release.

Consumers can use the published image in place of `path`:

```yaml
plugins:
  downloads:
    image: ghcr.io/woodleighschool/stemma-catalog/downloads:TAG
```

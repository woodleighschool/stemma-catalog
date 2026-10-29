# Fleet agent packaging

Native Fleet component discovery and macOS payload preparation for Stemma.

## 🧩 Operations

`fleetd` resolves `orbit`, `osqueryd` or `desktop` for macOS. `channel` defaults to
`stable` and also accepts a published component version. Discovery verifies Fleet's
TUF metadata and records the versioned target, length and SHA-512. Acquisition
downloads exactly those bytes and verifies them against the lock.

`fleetd/v1alpha1/FleetMacPayload` consumes those components and emits `payload`
(TAR) and `postinstall` (script) artifacts. `BuildMacPkg` owns the receipt,
release version and PKG assembly. `MacSoftware` publishes the resulting installer.

Supply `fleet_url` and `enroll_secret` through environment expressions, or set
`use_system_configuration: true` when a macOS configuration profile supplies
enrollment. The built payload and PKG contain any supplied enrollment secret;
source locks and artifact evidence contain no enrollment settings.

`enable_scripts`, `disable_updates` and `desktop` default to false.
`orbit_channel`, `osqueryd_channel` and `desktop_channel` default to `stable`.
The generated schema describes the full configuration.

## 📦 Deployment

Keep the receipt `com.fleetdm.orbit.base.pkg` when replacing a Fleet-generated
package. Files install beneath `/opt/orbit` and the hook restarts
`com.fleetdm.orbit`. Existing databases, host identity and keychain entries remain
owned by Orbit. Preparation never executes components or installer hooks.

Use an explicit increasing `BuildMacPkg.package.version` for each deployment,
including configuration-only changes. Intune can detect that receipt and version
when the `MacSoftware` declaration has no application selector. Setting
`disable_updates: true` makes MDM package releases own the component versions.

Publish the installer to Intune and Jamf through `MacSoftware`. Jamf package
publication does not create or reset install policies unless the declaration requests them.

Releases publish OCI bundles at `ghcr.io/woodleighschool/stemma-catalog/fleetd`.
Catalogs select a released image tag and digest; local paths are for development.

## 🧑‍💻 Development

```sh
mise run //plugins/fleetd:build
```

`stemma update` reviews the current channel selections; `prepare` builds from the
lock. Increase the package version when releasing changed selections or settings.
For local development, select `path: plugins/fleetd/build` in the Project. Rebuild
and run `stemma plugins update fleetd` after source changes. Tests use synthetic
signed metadata and inert binaries.

Run the plugin's `test`, `lint`, `vulncheck` and `release-check` Mise tasks for
verification. `snapshot` creates portable executables and bundles for macOS,
Linux and Windows on amd64 and arm64.

## 📚 Upstream

The bootstrap TUF root, certificate bundle and installation contract come from
[Fleet](https://github.com/fleetdm/fleet/tree/91e9755e1c955cafbfcfdc5eeab7ce5c91a5865c/orbit).
The pinned source revision is `91e9755e1c955cafbfcfdc5eeab7ce5c91a5865c`.
Fleet's licence is retained in `internal/fleetd/assets/FLEET-LICENSE` and included
in plugin bundles. TUF validates root rotation from the embedded bootstrap;
component releases are selected independently of this source revision.

`certs.pem` seeds osquery's public CA trust store. Fleet's
[nightly workflow](https://github.com/fleetdm/fleet/blob/main/.github/workflows/update-certs.yml)
proposes bundle updates upstream. Starting with Orbit 1.62.1,
[startup refreshes the installed bundle](https://github.com/fleetdm/fleet/blob/orbit-v1.62.1/orbit/pkg/cabundle/cabundle.go)
from the copy embedded in Orbit, even with self-updates disabled. MDM-delivered
Orbit releases therefore carry CA updates. The seed remains in the package
because Orbit leaves a missing bundle missing. Older Orbit versions do not
perform this refresh.

Set `minimum_os` from [Fleet's supported operating systems](https://fleetdm.com/docs/get-started/faq#what-host-operating-systems-does-fleet-support).

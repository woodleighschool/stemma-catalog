# Resource documents

Use the catalog's conventions for files, naming, categories, audiences and YAML style. In a new
catalog, start with a Project importing the resource files; add shared components only when
several resources need the same values. Resource identity comes from kind and name, not the file
path. Moving a document changes the base for its relative inputs and assets.

The schema describes the available kinds, selectors and destination fields. An existing vendor
installer or application belongs in a software kind; files assembled into a package belong in a
builder. Source-free policies need a destination that supports them. See [templates.md](templates.md).

## Composition and publication

`extends` uses a Project component: maps merge, while lists and null replace inherited values.
`source.resource` consumes another resource's output and is a build dependency. Destination
relationships such as Munki `requires` or Intune `dependencies` order publications; they do not
supply build inputs. Name related resources by kind and name.

Destination keys identify connections in the Project. Keep metadata in the destination's
vocabulary, including native Munki pkginfo keys. Describe the product and set its intended
publication policy. Reuse appropriate vendor copy; assignments and management fields carry the
deployment context. Omit values that derivation already supplies correctly.

`prepare --json` reports `publications` without destination credentials. An entry with `metadata`
shows declared and derived fields, values to clear and their origins. An entry with `deferred`
explains why a preview is unavailable. `plan` compares this managed metadata with remote state.

Omitted fields preserve remote values unless Stemma derives them. Explicit values override
inference, supplied lists replace their collections, and supported nulls clear fields. Deleting
a declaration does not generally clear the remote value. For example, `exclude: []` beside an
`include` explicitly clears exclusions.

## Installation, version and detection

A vendor PKG keeps its payload and installer behaviour. By default, its package metadata supplies
the version and receipts. `application.path` or `application.bundle_id` selects an app supplying
version, detection and icon evidence; it does not remove other payloads from the PKG. A disk
image or archive containing one app selects it automatically.

Review that selection against the product. Receipts establish package installation state;
application `installs` entries establish that the installed product is present at the expected
version. Components can be conditional on OS or architecture, so requiring every observed receipt
may repeatedly request installation of a correctly installed suite. Select the evidence that
represents the intended state; optional components need not be mandatory detection items.

For several applications, name inspected subjects and use `"{{ facts.<name> }}"` as `installs`
items. Each supplies its installed path, bundle identifier and comparison version. Native
`installs` entries remain available when detection differs from those facts. A DMG folder can be
copied with `items_to_copy`; its nested apps are detected at the resulting installed paths.

Detection and the managed version are separate. Use the installer, a selected app or an explicit
component/source expression for the release version; Stemma does not combine app versions.
When the managed version stays the same, changed installer content replaces that publication.
Munki uses detection versions to decide which Macs need it; changed bytes alone do not trigger
reinstallation. [Templates](templates.md#suite-detection-and-component-version) show the selectors.

An app's `architectures` fact describes its main executable. An Intel-only app needs Rosetta on
Apple silicon; the catalog decides how to provide it. An absent fact means unknown. Likewise,
static inspection cannot establish behaviours implemented only by installer scripts.

## Wrappers and assembled packages

An executable installer app needs its documented unattended invocation. A `BuildMacPkg` can
carry it in `scripts` and invoke it from `postinstall`; `MacSoftware` publishes that package.
Select archive contents with `$input` and `path: .`, and locate script resources beside `$0`.
Verify vendor inputs through the build's `signatures`; the builder's own package is unsigned.

Build paths are exact and direct input facts stop at the packages they contain. If a build needs
to select a nested package by glob or use facts from inside it, select it in a software resource
without destinations and consume that resource as the build input. This preserves the vendor
artifact without duplicating selection logic in scripts.

A wrapper receipt records the wrapper installation, which may not establish the product's
continued presence. Use vendor documentation and corroborating automation to establish installed
paths, versions, minimum OS and removal. Prefer native detection when it expresses those facts;
use `version_script` when the installed version requires a script. Check absent, older, current
and newer installations where runtime verification is needed.

## Signatures and artwork

Declare the expectations reported by `stemma signature` or trial MCP `prepare`. An entry without
`subject` covers a sole signing subject; identify subjects when there are several. Review
unsigned observations explicitly. Invalid or unsupported signatures are not unsigned inputs.
Software consuming an unsigned package build declares no signer for that output.

Use suitable existing artwork or extract it with `stemma icon`. An input can be selected with
`--input vendor`; `--path` chooses among apps using the inspected subject path, including inside
packages. `--from PATH` reads a local application, installer or image without a resource source
or lock. Declare the icon according to the target catalog's layout.

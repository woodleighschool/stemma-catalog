# How Stemma behaves

Use the installed CLI's `--help` and generated schema for its current interface. A connected
Stemma MCP server also provides `describe`. These are different interfaces: MCP tool names are
not automatically CLI commands.

## CLI and MCP

| Purpose                                           | CLI                                                                        | MCP                                                                                  |
| ------------------------------------------------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| Discover fields                                   | `stemma schema --output-file -`; add `--builtins` when there is no Project | `describe`, optionally selecting `kind`, `resolver`, `destination` or `field`        |
| Resolve current inputs and write their lock       | `stemma update Kind/name`                                                  | `update` with `resources: ["Kind/name"]`                                             |
| Prepare locked inputs                             | `stemma prepare Kind/name`                                                 | `check` prepares affected locked resources since a revision                          |
| Trial current sources without writing input locks | `stemma artifact Kind/name --no-input-lock` materializes an artifact       | `prepare` with `resources: ["Kind/name"]` reports artifacts and signing expectations |
| Derive signing expectations from locked inputs    | `stemma signature Kind/name`                                               | Included in trial `prepare`                                                          |
| Inspect a builder input before it can build       | `stemma inspect Kind/name --input vendor [--path Installer.app] --json`    | `inspect` with `resource`, `input` and optional `path`                               |
| Check a change                                    | `stemma prepare --changed-since REV`, then `stemma validate`               | `check` with `since: REV`                                                            |

There is no CLI `describe` or `check`. `stemma mcp` serves an existing project; the skill does not
configure or connect it automatically. Use the CLI when MCP is unavailable. Trial preparation
can download installers: use `update` for an update check that only needs release and digest
changes. `artifact --no-input-lock` still enforces declared signatures, whereas MCP `prepare`
derives signing expectations for a draft. Neither refreshes plugin locks.

## The lockfile

- `update` resolves sources and records them in `stemma.lock.yaml`, preserving unselected resources'
  entries. Homebrew records and WinGet manifests with hashes let it pin a release without fetching
  its installer, even with an empty artifact cache. Hashless Homebrew casks still need their vendor
  bytes. Preparation fetches locked bytes when needed and verifies the recorded digest.
- CLI `prepare`, `signature`, `icon`, `plan` and `apply`, and MCP `check` and `icon`, use locked inputs.
  A missing or stale entry fails with "run stemma update". `--offline` requires the verified cached
  inputs; it does not refresh sources. Plugin tags and local paths use lock entries;
  digest-pinned images select their code directly.
- MCP `prepare` resolves current sources without writing the input lock. Its reported changes are
  what an update at that moment would record; a later update can observe a newer release.
- CLI `prepare --changed-since REV` and MCP `check` check the lock and prepare what the change
  affects among the resources a run without selectors takes. MCP `check` also validates every
  document. Each revision uses its own plugin
  implementations. Changes to resource or
  input resolver operation identities affect preparation and downstream consumers. Destination
  metadata and destination operation identities do not select resources for preparation.

## Environment values

`{{ env.NAME }}` is read where the value is used. Plain `stemma validate` does not evaluate it;
`validate --resolved` does and prints resolved configuration, so avoid it when values are secret.
Acquisition reads credentials for the inputs it needs. When Stemma reports a missing value you don't
have, stop and report it; never set a placeholder or dummy value to get past it.

## Composition

`extends` names a Project component. Maps merge recursively; lists and `null` replace what the
component set. MCP `describe` lists existing components; otherwise read the Project. A new catalog
does not need components until it has useful shared defaults.

## What comes from the artifact

Software kinds inspect the prepared artifact and derive what it states: version, identifiers,
receipts and installed applications, detection, minimum OS, installed size and signer evidence.
CLI `prepare --json` and MCP `prepare` report output subjects. To discover builder input facts,
use `inspect` with the input name and optional path. CLI input inspection uses the lock unless
`--no-input-lock` is set; MCP `inspect` reads current sources. Neither builds the selected resource
or writes the lockfile. Destinations turn artifact facts into
their own fields. An explicit value replaces a derived one. Destination
metadata can also use expressions:

- `facts` for inspected subjects, such as `{{ facts.app.app.version }}`;
- `evidence` for metadata the resolver or verification supplied, such as
  `{{ evidence['vendor.release'].version }}`;
- `inputs` in a package build, for its named inputs' facts and versions.

## Dependencies

- A build dependency is an input that names another resource, such as `source.resource`. The
  producer is prepared first, and a change to it prepares its consumers too.
- A publication dependency is destination metadata, such as Munki `requires` or Intune
  `dependencies`, naming other resources. It orders publication within that destination and
  prepares nothing.

## Profiles and suspend

A run without selectors takes the resources that declare no profile and aren't suspended.

- `profiles` names an opt-in workload: resources that run by themselves, but on one prepared machine
  or on their own schedule. `--profile NAME` runs that workload instead of the rest; repeat it to add
  profiles. `--profile` and selectors can't be combined.
- `suspend: true` keeps a resource out of every run that doesn't name it, including runs of its
  profiles. Use it for software whose files or download link the repository can't carry, or that is
  paused.

A selector runs either kind by name. A build that a selected resource references is prepared
whatever profile it declares. Both keep their lock entries between runs, are still validated, and
are skipped by `reconcile`. `prepare --changed-since` and MCP `check` leave both out; CLI
`--profile NAME` compares that profile instead.

Choose by how the resource is run, not by its source or resolver. A stable URL without release
metadata still updates by itself and needs neither; a local path needs a profile only when ordinary
runners don't have what it points at.

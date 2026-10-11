# Stemma interfaces and evidence

Use the chosen executable's help and generated schema, or a connected MCP server's `describe`,
for the operations available to the task. CLI and MCP names are not interchangeable.

## CLI and MCP

| Purpose                                           | CLI                                                          | MCP                                                  |
| ------------------------------------------------- | ------------------------------------------------------------ | ---------------------------------------------------- |
| Discover fields                                   | `schema --output-file -`; add `--builtins` without a Project | `describe` with kind, resolver, destination or field |
| Resolve sources and write their lock              | `update Kind/name`                                           | `update` with `resources`                            |
| Prepare locked resources                          | `prepare Kind/name --json`                                   | `check` prepares changes since a revision            |
| Trial current sources without writing input locks | `artifact Kind/name --no-input-lock`                         | `prepare` with `resources`                           |
| Derive signing expectations                       | `signature Kind/name`                                        | Included in trial `prepare`                          |
| Inspect an input without building                 | `inspect Kind/name --input vendor --json`                    | `inspect` with resource and input                    |
| Inspect a local file                              | `inspect PATH --json`                                        | Use an available local CLI                           |
| Compare declarations to the lock                  | `validate --locked`                                          | `check` with `locked: true`                          |
| Check affected preparation                        | `prepare --changed-since REV`                                | `check` with `since: REV`                            |

CLI commands above follow `stemma`. There is no CLI `describe` or `check`. MCP `prepare` reports
artifacts and signing expectations; CLI `artifact` materializes an output and enforces declared
signatures. Both trial paths can download files and leave input locks unchanged; neither refreshes
plugin locks. Follow a trial with `update` and frozen preparation when the result needs a reviewed
lock.

## Inspection

`prepare --json` reports output subjects. `inspect --input NAME` reads a named resource input
without building it, using locked sources unless `--no-input-lock` is set. MCP `inspect` reads
current sources. `--path` selects a subject inside the artifact.

`inspect --read PATH` reads one entry; `artifact --read PATH` does the same for a prepared output.
Use `Distribution`, `Core.pkg/Scripts/postinstall` or a payload path inside a PKG. Paths can enter
packages inside images or archives. Directories list their children; `.` is the root. Stemma
reads DMG contents without mounting through hdiutil, so a hdiutil licence prompt does not establish
that Stemma cannot inspect an image.

Facts describe static contents. Preparation never executes installer hooks, and a successful
inspection does not prove an unattended installer invocation works.

## Locks and change checks

`update` records source selections under `stemma.lock/`, preserving unselected entries. Trusted
release hashes can avoid downloading installers during update; preparation fetches and verifies
locked content when needed. CLI `prepare`, `signature`, `icon`, `plan` and `apply`, and MCP `check`
and `icon`, reject missing or stale inputs. `--offline` requires verified cached inputs.
Plugin tags and local paths use locks; digest-pinned images identify their code directly.

`validate --locked` compares declarations without acquiring content or checking signatures and
digests. It reports environment-dependent inputs it cannot compare as unresolved.

`prepare --changed-since REV` checks locks and prepares affected resources and build consumers.
Each revision uses its own plugins. Resource-kind and input-resolver changes affect preparation;
destination metadata and destination implementation changes do not. Prepare metadata edits by
name to see their publications. MCP `check` also validates every document.

## Expressions and environment

- `facts` contains inspected subjects, addressed by path or a declared subject name, such as
  `{{ facts.editor.app.version }}`.
- `evidence` contains namespaced resolver and verification metadata. Use the fields the operation
  actually returns, such as `{{ evidence['vendor.release'].version }}`.
- `inputs` in a build exposes its named inputs' facts and versions.

`{{ env.NAME }}` is evaluated where it is used. Plain `validate` does not evaluate it;
`validate --resolved` does and prints configuration, which may expose secrets. Use available
credentials only for the operation that needs them; never supply dummy values to bypass a missing
one. Keep private inputs and credentials outside Git.

## Profiles and suspension

An unselected run takes resources with no profile that are not suspended. `--profile NAME`
selects an opt-in workload; it cannot be combined with resource selectors. `suspend: true` excludes
an item from unselected runs, including profiles. A resource selector can run either explicitly.
Referenced build inputs are prepared regardless of profile.

Both retain their locks and remain subject to validation. `reconcile` and default change checks
skip them; CLI change checks can select a profile. The catalog decides workload policy. Neither
an unversioned URL nor a local path inherently requires suspension or a profile.

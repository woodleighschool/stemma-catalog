---
name: stemma-catalog
description: Use when creating a Stemma catalog or adding, updating, reviewing or reorganising its software, including finding a vendor's stable download, choosing a source or resolver, checking prepared artifacts, writing resource YAML, or assessing a resolver plugin gap.
---

# Stemma catalog

Turn a software request into checked resources that install the intended product and follow its
release channel. Use registry metadata for release checks and prepare artifacts to verify delivery.

## Rules

- Repository and user instructions come first.
- The installed Stemma's schema and command help are the CLI authority. Use
  `stemma schema --builtins --output-file -` before a project exists, or
  `stemma schema --output-file -` inside one to include its plugins and connections. If a Stemma
  MCP server is connected, its `describe` tool offers focused field discovery. There is no CLI
  `describe` command. See [references/behaviour.md](references/behaviour.md) for the tool differences.
- When resources exist, read one of the same kind for local conventions. Do not assume its
  components, connections or plugins exist elsewhere, or carry over a workaround without a reason.
- The vendor is the authority. Community automation such as AutoPkg shows where a vendor publishes;
  use what it reveals, not how it does it.
- Stemma reads environment values only where it uses them; never set a placeholder or dummy value to
  get past a missing one. Never read or print credential files.
- Keep plugin dependency updates separate from unrelated software changes.
- Keep work within the requested scope. `plan` reads destinations; `apply` and `reconcile` publish.
  Publishing, commits and pushes require an explicit request.

## Workflow

1. **Orient.** Read any repository instructions and [references/behaviour.md](references/behaviour.md).
   Check the installed commands and schema using the repository's tool runner when it has one.
   For a new catalog, use the Project and layout in [references/templates.md](references/templates.md);
   add connections, components and plugins only as needed. If Stemma is unavailable, report that
   executable validation is unavailable rather than guessing its capabilities.
2. **Pin down the request.** Product and publisher, platform, release channel, architecture, and
   whether this is a vendor installer, a package built from files, or a policy without an installer.
   Find these out; ask only when the evidence can't settle something that changes the result.
3. **Choose the source** with [references/discovery.md](references/discovery.md). Preserve release
   and deployment semantics, then prefer a native resolver when it expresses the same behaviour.
   Reassess existing custom resolvers rather than treating them as precedent.
4. **Draft the document** from the matching template in
   [references/templates.md](references/templates.md), refined with
   [references/documents.md](references/documents.md). Consult the schema or MCP `describe` for the
   fields you use; examples are starting points, not evidence of installed capabilities.
5. **Resolve and review.** Run `stemma update Kind/name` to record the
   selected sources. Review the lock changes, including a newly created lockfile. An update-only
   request can end here. If local policy reserves lock updates for a maintainer, report that step
   as pending; MCP `prepare` can trial a draft without writing its input lock when available.
6. **Prepare when required.** CLI `stemma prepare Kind/name` uses the lock and rejects stale or
   missing inputs. For a new resource or a signer change, use `stemma signature Kind/name`, review
   its signed or unsigned observations, and declare the expectations before preparation. Compare
   artifact evidence with the installation, detection and removal contract in
   [references/documents.md](references/documents.md). Create a declared icon with
   `stemma icon Kind/name` when needed. MCP `prepare` instead resolves current sources and reports
   artifacts and signing expectations without writing the lock; follow it with `update` and frozen
   verification before calling the result locked.
7. **Check the change.** Run any repository formatter on changed files and `stemma validate`.
   With an existing comparison revision, run `stemma prepare --changed-since REV` or MCP `check`
   with `since: REV`; use the actual merge base or target branch, not an assumed `origin/main`.
   For a new catalog without that history, prepare the changed resources directly.
8. **Raise gaps.** When no source expresses the vendor cleanly, or Stemma can't state an ordinary
   policy, follow [references/plugin-gaps.md](references/plugin-gaps.md) instead of working around
   it. When Stemma fails on a vendor file that looks valid, put the tool, its arguments and the error
   in the report, leave out what it would have produced and carry on with the rest of the request;
   debugging Stemma is separate work.

## Report

End with a line or two for each of:

- the resource (`Kind/name` and file) and what it delivers;
- the source, and why it won over the alternatives you checked;
- the selected release and digest; when prepared, its artifact version, identifiers and signer;
- what you verified and how, and what you could not verify;
- open items: gaps, failed tools with their errors, proposed plugins and decisions for a maintainer.

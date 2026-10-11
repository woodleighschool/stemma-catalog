---
name: stemma-catalog
description: Create, maintain or review a Stemma software catalog, including vendor source discovery, resource composition, artifact inspection and resolver gaps.
---

# Stemma catalog

Express the requested software and deployment behaviour in a catalog, then check the prepared
artifact and publication metadata. The catalog's instructions supply its policies: platforms,
layout, naming, audiences, categories and defaults. Examples in this skill do not establish them.

## Work from the request

Read the Project and a relevant resource when they exist. Establish the product, edition, release
channel, architecture and installation behaviour. Resolve ordinary choices from evidence; ask
when the remaining alternatives would deliver different products or policies.

Use the Stemma executable or MCP server available for the task. Its command help and schema
establish capabilities: `stemma schema --builtins --output-file -` works without a Project;
inside one, omit `--builtins` to include configured plugins and connections. MCP `describe`
provides focused field discovery.
See [behaviour.md](references/behaviour.md) for command, lock and expression semantics.

## Find and describe the software

Use [discovery.md](references/discovery.md) to choose a source that follows the intended release
and preserves installation behaviour. Vendor documentation and artifacts establish what is
shipped; AutoPkg and other community automation can reveal discovery and deployment mechanisms.

Use [documents.md](references/documents.md) for composition and detection, and
[templates.md](references/templates.md) for source and packaging examples. Add only the
components, plugins and destinations the catalog needs. Inspect inputs before writing
expressions that depend on their facts.

When existing operations cannot express the request, use
[plugin-gaps.md](references/plugin-gaps.md) to identify the missing contract. A catalog workaround
should not disguise a Stemma defect or replace the requested product with an easier download.

## Verify the affected behaviour

- `stemma update Kind/name` resolves and locks sources. Review the resulting changes. An
  update-only request can finish here. Batch independent lookups and resource selectors when useful.
- For new software or changed signing expectations, use `stemma signature Kind/name` and declare
  the reviewed signer or unsigned observations. `stemma prepare Kind/name --json` verifies the
  locked input, prepares it and reports publication metadata without connecting to destinations.
  MCP `prepare` can trial a draft and derive signing expectations without writing its input lock.
- Check what the installer delivers, how the destination detects its presence and version, and
  how removal works. Preparation inspects contents; it does not run installer hooks. Extract a
  declared icon with `stemma icon Kind/name` when artwork is needed.
- Format changed files and run `stemma validate`. For acquisition or packaging changes,
  `stemma prepare --changed-since REV` or MCP `check` with `since: REV` uses the actual comparison
  revision. These skip profiled and suspended resources by default and do not select metadata-only
  changes; prepare those items by name to check their previews.

`plan` reads live destinations; `apply` and `reconcile` publish.

Report the delivery and source choices, meaningful verification and unresolved behaviour.
Use a compact table for batches when helpful; include versions or digests when they explain a
change. Identify tool failures with the operation and error, without claiming the failed check
succeeded.

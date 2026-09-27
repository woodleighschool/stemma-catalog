---
name: stemma-catalog
description: Use when adding, updating, reviewing or reorganising software in a Stemma catalog (a Git repository with a stemma.yaml Project), including finding a vendor's stable download, choosing a Stemma source or resolver, checking what Stemma prepares, writing or ordering resource YAML, or deciding whether a vendor needs its own resolver plugin.
---

# Stemma catalog

Turn a software request into a Stemma resource that follows the vendor's own release channel, uses
what the installed Stemma supports and reads cleanly in review.

Take the cheapest path to a checked result. Stemma's MCP tools (`stemma mcp`) answer most questions
in seconds, so draft the document as soon as there is a candidate source and let `prepare` test it,
rather than reading further.

## Rules

- Repository and user instructions come first.
- Stemma is the authority on what a document can declare, including what its plugins add. Ask
  `describe` rather than relying on neighbouring documents or memory: without arguments it lists the
  resource kinds, resolvers with the values they accept, destinations and components; name a kind,
  resolver or destination for its fields, and a `field` to narrow to one block.
- Existing resources show local conventions, such as components and naming; one document of the same
  kind is enough. Don't carry over a resolver, field or workaround because another document has it.
- The vendor is the authority. Community automation such as AutoPkg shows where a vendor publishes;
  use what it reveals, not how it does it.
- Stemma reads environment values only where it uses them; never set a placeholder or dummy value to
  get past a missing one. Never read or print credential files.
- Keep plugin dependency updates separate from unrelated software changes.
- Stay local. Publishing (`plan`, `apply`, `reconcile`), commits and pushes need an explicit
  request.

## Workflow

1. **Orient.** Read the repository's agent instructions and
   [references/behaviour.md](references/behaviour.md). Ask `describe` for the overview, then for the
   kind you need, and read one existing document of that kind.
2. **Pin down the request.** Product and publisher, platform, release channel, architecture, and
   whether this is a vendor installer, a package built from files, or a policy without an installer.
   Find these out; ask only when the evidence can't settle something that changes the result.
3. **Find the source** with [references/discovery.md](references/discovery.md). Stop at the first
   candidate that passes its checks.
4. **Draft the document** from the matching template in
   [references/templates.md](references/templates.md), refined with
   [references/documents.md](references/documents.md). Ask `describe` for a resolver's or
   destination's fields when you set more than the template shows.
5. **Prepare it.** `prepare` fetches the sources as they are now without writing the lockfile, and
   reports each artifact's version, inspected applications, packages or installers, and the
   `signatures` block to declare. It also checks destination metadata. Compare the result with what
   the vendor publishes, fix the document and prepare again until they match, then add the
   `signatures` block.
6. **Lock it.** `update` records the sources in the lockfile, and `icon` creates a declared icon
   that the repository doesn't have yet. If the repository leaves lock updates to a person, or the
   vendor is unreachable from here, stop before this step and list the resources that need it.
7. **Check the change** as pull requests do: run the repository's formatter, then `check` since
   the branch the change merges into, such as `origin/main`.
8. **Raise gaps.** When no source expresses the vendor cleanly, or Stemma can't state an ordinary
   policy, follow [references/plugin-gaps.md](references/plugin-gaps.md) instead of working around
   it. When Stemma fails on a vendor file that looks valid, put the tool, its arguments and the error
   in the report, leave out what it would have produced and carry on with the rest of the request;
   debugging Stemma is separate work.

## Report

End with a line or two for each of:

- the resource (`Kind/name` and file) and what it delivers;
- the source, and why it won over the alternatives you checked;
- the prepared artifact: version, identifiers and signer;
- what you verified and how, and what you could not verify;
- open items: gaps, failed tools with their errors, proposed plugins and decisions for a maintainer.

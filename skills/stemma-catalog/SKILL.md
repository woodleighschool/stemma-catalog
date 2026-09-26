---
name: stemma-catalog
description: Use when adding, updating, reviewing or reorganising software in a Stemma catalog (a Git repository with a stemma.yaml Project), including finding a vendor's stable download, choosing a Stemma source or resolver, checking what Stemma prepares, writing or ordering resource YAML, or deciding whether a vendor needs its own resolver plugin.
---

# Stemma catalog

Turn a software request into a Stemma resource that follows the vendor's own release channel, uses
what the installed Stemma supports and reads cleanly in review.

Take the cheapest path to a checked result. Stemma's commands answer most questions in seconds, so
draft the document as soon as there is a candidate source and let `stemma validate` and
`stemma artifact` test it, rather than reading further.

## Rules

- Repository and user instructions come first.
- Learn what Stemma supports from the installed binary with the queries in step 1.
  `stemma operations` and `stemma schema` print hundreds of kilobytes; filter them with `jq` rather
  than reading them whole.
- Existing resources show local conventions, such as components and naming; one document of the same
  kind is enough. Don't carry over a resolver, field or workaround because another document has it.
- The vendor is the authority. Community automation such as AutoPkg shows where a vendor publishes;
  use what it reveals, not how it does it.
- Run `stemma` the way the repository does, such as through its tool manager. Commands read
  environment values only where they use them; never set a placeholder or dummy value to get past a
  missing one. Never read or print credential files. Keep `stemma validate --resolved` output
  private: it can contain secrets.
- Leave plugins out of software changes. Pull request checks refuse changed plugins, which are
  reviewed and verified on their own.
- Stay local. `plan`, `apply`, `reconcile`, commits and pushes need an explicit request.

## Workflow

1. **Orient.** Read the repository's agent instructions, `stemma.yaml`, one existing document of the
   kind you need and [references/behaviour.md](references/behaviour.md). Then list the resource
   kinds, resolvers and destinations with their fields, and every source form:

   ```sh
   stemma operations | jq -r '.operations[] | [.kind, (.resource.kind // .name), (.config_schema.properties // {} | to_entries | map(.key + (if .value.enum then "=" + (.value.enum | map(tostring) | join("|")) else "" end)) | join(" "))] | @tsv'
   stemma schema --output-file - | jq -r '."$defs".Input.oneOf[] | [.properties.resolver.const // "(implicit)", (.properties | keys - ["resolver"] | join(" "))] | @tsv'
   ```

2. **Pin down the request.** Product and publisher, platform, release channel, architecture, and
   whether this is a vendor installer, a package built from files, or a policy without an installer.
   Find these out; ask only when the evidence can't settle something that changes the result.
3. **Find the source** with [references/discovery.md](references/discovery.md). Stop at the first
   candidate that passes its checks.
4. **Draft the document** from the matching template in
   [references/templates.md](references/templates.md), refined with
   [references/documents.md](references/documents.md).
5. **Check what Stemma prepares**, without touching the lockfile:

   ```sh
   stemma validate
   stemma inspect "$(stemma artifact Kind/name --no-input-lock)"
   ```

   Compare the version, selected application or installer, identifiers and minimum OS with what the
   vendor publishes. When they differ, fix the document and run the checks again.

6. **Lock and sign.** `stemma update Kind/name` records the source in the lockfile.
   `stemma signature Kind/name` then prints the verified signer; add it to the document. If the repository leaves lock updates to a person, or the vendor is unreachable from
   here, stop before this step and list the resources that need it.
7. **Check the change** as pull requests do, against the branch they merge into:

   ```sh
   stemma validate
   stemma prepare --changed-since origin/main
   ```

8. **Raise gaps.** When no source expresses the vendor cleanly, or Stemma can't state an ordinary
   policy, follow [references/plugin-gaps.md](references/plugin-gaps.md) instead of working around
   it. When a Stemma command fails on a vendor file that looks valid, put the exact command and error
   in the report, leave out what the command would have produced and carry on with the rest of the
   request; debugging Stemma is separate work.

## Report

End with a line or two for each of:

- the resource (`Kind/name` and file) and what it delivers;
- the source, and why it won over the alternatives you checked;
- the prepared artifact: version, identifiers and signer;
- what you verified and how, and what you could not verify;
- open items: gaps, failed commands with their errors, proposed plugins and decisions for a
  maintainer.

# How Stemma behaves

The installed binary is the authority on fields: read them with `stemma operations` and
`stemma schema`. This page covers what the schema can't say.

## The lockfile

- `stemma update [Kind/name]` resolves sources and records them in `stemma.lock.yaml`. It is the
  only command that writes input entries.
- Every other command uses the lockfile as it is. An input without a current entry fails with
  "run stemma update", and plugins must match their entries.
- `stemma artifact Kind/name --no-input-lock` resolves the sources as they are now without writing
  anything: use it to try a draft.
- `stemma prepare --changed-since REV` prepares only what a change affects, as pull request checks
  do. Destination metadata never counts, so a description edit prepares nothing. A changed plugin
  fails it: plugin changes are reviewed and verified on their own.

## Environment values

`{{ env.NAME }}` is read only by a command that uses the value: `validate` reads none, and `update`
or `artifact` read only those of the resources they fetch. When a command reports a missing value
you don't have, stop and report it; never set a placeholder or dummy value to get past it.

## Composition

`extends` names a Project component. Maps merge recursively; lists and `null` replace what the
component set. `stemma validate --resolved` prints the merged result, which can contain secrets.

## What comes from the artifact

Software kinds inspect the prepared artifact and derive what it states: version, identifiers,
receipts and installed applications, detection, minimum OS, installed size and signer evidence.
Destinations turn these into their own fields. An explicit value replaces a derived one. Destination
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

## Suspend

`suspend: true` keeps a resource declared and validated but out of every run that doesn't name it.
Its lock entries stay as they are. Use it for software whose files the repository can't carry, or
that is paused.

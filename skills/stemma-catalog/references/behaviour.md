# How Stemma behaves

Stemma's `describe` is the authority on fields. This page covers what a field list can't say.

## The lockfile

- `update` resolves sources and records them in `stemma.lock.yaml`. It is the only tool that
  writes input entries, and it keeps other resources' entries.
- `icon` and `check` use the lockfile as it is. An input without a current entry fails with "run
  stemma update", and plugins must match their entries.
- `prepare` resolves the sources as they are now and writes nothing: use it to try a draft. Its
  input changes are what `update` would record.
- `check` rejects plugin changes, prepares what a change affects since a revision, then validates
  the catalog, as pull request checks do. Destination metadata never counts, so a description edit prepares nothing. A
  changed plugin fails it: plugin changes are reviewed and verified on their own.

## Environment values

`{{ env.NAME }}` is read only where the value is used: validation reads none, and `prepare` or
`update` read only those of the resources they fetch. When Stemma reports a missing value you don't
have, stop and report it; never set a placeholder or dummy value to get past it.

## Composition

`extends` names a Project component. Maps merge recursively; lists and `null` replace what the
component set. `describe` lists what each component sets.

## What comes from the artifact

Software kinds inspect the prepared artifact and derive what it states: version, identifiers,
receipts and installed applications, detection, minimum OS, installed size and signer evidence.
`prepare` reports these as the artifact's subjects. Destinations turn them into their own fields. An explicit value replaces a derived one. Destination
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

# Writing resource documents

## Kind, name and file

- Pick the kind from what is delivered: an existing vendor installer or application uses the
  platform's software kind, files you assemble use the package builder, and a policy with no
  installer uses the software kind's source-free form where the destination supports one.
- `describe` gives a kind's or destination's fields, one line each with `*` marking required ones,
  their types, allowed values, defaults and descriptions. Name a `field`, such as `pkginfo` or
  `targets.include`, to narrow to one block rather than reading the whole destination.
- `metadata.name` is the resource's identity in every destination, and renaming it creates a new
  item. Use the product's plain name in lowercase kebab case, such as `google-chrome`. Add a major
  version only when majors install side by side as separate products.
- Place the file as the repository's instructions say. Otherwise use one resource per file, named
  after it; documents that belong together, such as platform editions or a build and the software
  that publishes it, can share a folder.

## Declare what you mean

- Extend a Project component when it holds the shared defaults for this kind of item, and don't
  repeat what it sets; `describe` lists what each component sets.
- Let Stemma select the application or installer. `prepare`'s evidence names the one it chose, even
  when helpers are nested inside it; set a selection only when that choice is wrong, and prefer
  identifiers that survive upgrades, such as a bundle ID or product code, over paths that contain a
  version.
- Let Stemma derive what the artifact states: version, identifiers, detection, receipts, removal
  method, installed size, minimum OS. Set a derived field only to correct it, even when neighbouring
  documents set it. When a derived value comes from the wrong application or file, fix the selection
  instead.
- Leave out a field the destination fills in from other fields, as its description says, unless
  that default is wrong for this software. Munki, for example, blocks on the applications in
  `installs` when `blocking_applications` is absent.
- Set descriptive fields yourself: display name, description, publisher or developer, category. Base
  the description on the vendor's wording, cut to what an end user needs.
- Declare explicit signing expectations wherever the kind supports them. `stemma signature` reports
  each subject with its signer or `unsigned: true`. Review intentionally unsigned sources before
  declaring them; report invalid or unsupported signatures rather than omitting the assertion.
- Declare an icon by name (`icon: <name>` publishes `icons/<name>.png`) and create it with `icon`,
  unless the repository already has artwork for it.
- Refer to other catalog resources by `kind` and `name`, not by native names. `source.resource`
  consumes another resource's output, which is a build dependency. Destination relationships, such
  as Munki `requires` or Intune `dependencies`, are publication dependencies. Don't use one for the
  other.
- Keep destination metadata in the destination's vocabulary, such as Munki pkginfo keys.
- Supply secrets through environment expressions such as `"{{ env.VENDOR_TOKEN }}"`. Keep licensed
  installers, private files and credentials out of Git; a resource that needs files the repository
  can't carry sets `suspend: true`.

## Omitted destination fields

A destination field you omit keeps its current value unless Stemma derives it. A supplied list
replaces the whole collection, an empty list clears it and a supported `null` clears a value. So:

- deleting a field from the YAML doesn't clear it on the destination; set `null` or `[]`;
- declare a managed block whole: beside an `include`, `exclude: []` states that nothing is excluded
  instead of leaving exclusions to the destination;
- add a field when you mean to own its value, not to repeat the value you expect.

## Gaps

If an ordinary policy needs a no-op source, a weakened or removed verification, duplicated values or
a copied private file to pass validation, stop and report the gap with the smallest Stemma change
that would express it. See [plugin-gaps.md](plugin-gaps.md).

## YAML style

- Two-space indentation with sequences indented under their key, one final newline and no trailing
  whitespace. Run the repository's formatter on the files you change.
- Quote only values YAML would misread: versions such as `"1.0"`, octal modes such as `"0644"`,
  expressions, and strings starting with a special character. Write a regular expression plain, or
  in single quotes when it needs quoting, so its backslashes stay single. No anchors or aliases.
- Write long descriptions as wrapped plain text or a folded block, matching neighbouring files.
- Comment only what a reviewer needs and the YAML can't say, such as the name behind a signer or
  group ID.
- Leave unrelated documents as they are.

## Field order

Fields go in the order Stemma uses them, so a document reads from what is fetched to where it is
published.

A resource document: `apiVersion`, `kind`, `metadata`, `suspend`, `spec`.

A resource `spec`:

1. `extends`
2. what is acquired: `source` or `inputs`
3. what is selected or built from it, such as `package_path`, `application`, `content`,
   `setup_file`, `payload`, `scripts` and the built `package`
4. what is verified: `signatures`. A package build verifies an input before it builds, so its
   `signatures` follows `inputs`
5. what is published: `minimum_os`, `icon`, `subjects`, then `destinations` last

The Project `spec`: `imports`, `plugins`, `components`, `destinations`, `reconcile`. A destination
connection: `operation`, then `config`.

Within a block:

- The field that says what something is comes first, such as `resolver`, `url`, `path`, `resource`,
  `type`, `operation`, `intent` or `label_name`. The fields that refine it follow in the order they
  apply: `url` before `match`, `include` before `exclude`.
- Destination metadata runs from what the item is to what happens to old versions: identity and
  description, requirements, installation, detection, relationships, assignment, retention.
- Maps keyed by name, such as destinations, inputs, components, plugins and payload paths, are
  alphabetical, unless the names have an order of use, such as `preinstall` before `postinstall`.
- An embedded native document keeps its own convention: Munki pkginfo keys are sorted, as Munki
  writes them.
- Lists keep their order.

A kind or field not named here follows the same sequence: acquisition, selection, verification,
publication.

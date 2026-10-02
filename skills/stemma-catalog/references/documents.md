# Writing resource documents

## Kind, name and file

- Pick the kind from what is delivered: an existing vendor installer or application uses the
  platform's software kind, files you assemble use the package builder, and a policy with no
  installer uses the software kind's source-free form where the destination supports one.
- The generated schema gives the installed kinds' and destinations' fields. When MCP is connected,
  `describe` gives their types, required fields, allowed values, defaults and descriptions. Its
  `destination` parameter names a Project connection; `field` narrows to a block such as `pkginfo`.
- `metadata.name` is the stable resource identity; renaming it can change destination bindings.
  Use the product's plain name in lowercase kebab case, such as `google-chrome`. Add a major
  version only when majors install side by side as separate products.
- Place the file as the repository's instructions say. Otherwise use `software/<name>.yaml` for
  one standalone resource. Put genuine families, such as platform editions, side-by-side majors,
  or a build and its publisher, in `software/<name>/` with a file per resource. Keep the Project at
  `stemma.yaml` and import those files. A resource need not acquire a folder pre-emptively.

## Declare what you mean

- Extend an existing Project component when its shared defaults fit this item. Do not require a
  component for a new catalog or copy a component name from an example. Extract shared defaults
  only when actual resources need them.
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
- Set descriptive fields yourself: display name, description, publisher or developer, category.
- Describe the software itself. Start from the vendor's short product description and trim it for
  grammar, repetition, marketing or length; write a short factual one only when the vendor has
  nothing usable. Keep good vendor copy rather than rewriting it into a house voice.
- Leave deployment context to the fields that carry it: management state, who deploys or requires
  the item, audiences, policy, the destination, and the platform unless it distinguishes the item.
  Name the organisation only when the item is its own, such as its branding, fonts or a
  configuration of its systems. A product that takes part in a local policy, such as a filtering
  agent, is still described as the product.
- Say what an application does, not that it is installed. Installs, adds or configures belong to
  items where that action is the item: a driver, content pack, printer, configuration or other
  custom payload.
- Treat categories as a short browsing list when the destination uses them. Follow an existing
  repository list; without one, choose a broad category for the item's main purpose. Don't add a near-duplicate such as
  `Browser` beside `Browsers`, adopt a vendor's narrower term or keep a category the list no longer
  has; a new category is a repository decision.
- Declare explicit signing expectations wherever the kind supports them. `stemma signature` reports
  each subject with its signer or `unsigned: true`. Review intentionally unsigned sources before
  declaring them; report invalid or unsupported signatures rather than omitting the assertion.
- Where the kind supports an icon, declare its name (`icon: <name>` uses `icons/<name>.png`) and
  create it from locked software with `stemma icon Kind/name`, unless suitable artwork exists.
- Refer to other catalog resources by `kind` and `name`, not by native names. `source.resource`
  consumes another resource's output, which is a build dependency. Destination relationships, such
  as Munki `requires` or Intune `dependencies`, are publication dependencies. Don't use one for the
  other.
- Add only destinations the request needs. Each resource destination key names a connection in
  the Project; define that connection from the installed schema instead of assuming example
  aliases are configured. Keep metadata in the destination's vocabulary, such as Munki pkginfo keys.
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

- Follow any repository formatter. Otherwise use two-space indentation, sequences indented under
  their key, one final newline and no trailing whitespace. Format only the files you change.
- Quote only values YAML would misread: versions such as `"1.0"`, octal modes such as `"0644"`,
  expressions, and strings starting with a special character. Write a regular expression plain, or
  in single quotes when it needs quoting, so its backslashes stay single.
- Repeat a value within one document with an anchor and alias rather than copying it, such as
  `publisher: &publisher Vendor` and `developer: *publisher`. Merge keys (`<<`) are rejected.
- Keep a description on its key's line when it fits the formatter's print width. Otherwise wrap it
  as plain text at that width, continuing two spaces deeper than the key, and use a folded block
  (`>-`) only for text plain YAML would misread.
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
3. what is selected or built from it, such as `package_path`, `application`, `disk_image`,
   `content`, `setup_file`, `payload`, `scripts` and the built `package`
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

# AGENTS.md

This catalog expresses our deployment policy. The reusable [catalog skill](skills/stemma-catalog/SKILL.md)
works with other catalogs; keep our naming, layout, audiences and defaults here.

## Layout and documents

- `stemma.yaml` owns imports, components, plugin declarations and destination connections.
  `software/` holds resource documents and their local inputs; `icons/` holds reviewed artwork.
  Reviewed inputs and plugins are locked under `stemma.lock/`. `stemma cache path` locates the
  disposable cache; `.stemma/` holds the project process lock and local command state.
- Name files for their resources: `<name>.yaml` contains the input resources, build and software
  that form one item, in dependency order. A build-only vendor installer is `<product>-installer`
  and has no destinations. Use folders for related items or local inputs, as in `windows-app/`
  and `printers/`; keep resource-relative paths correct when moving files.
- Each resource document starts with `---`, followed by
  `# yaml-language-server: $schema=https://raw.githubusercontent.com/woodleighschool/stemma-catalog/main/stemma.schema.json`.
  The editor applies that comment to one document.
- Order a resource as `apiVersion`, `kind`, `metadata`, `profiles`, `suspend`, `spec`. Within `spec`,
  put `extends`, acquisition, selection or build instructions, signatures, other artifact metadata,
  then `destinations`. Sort named maps and native Munki pkginfo keys; keep list order meaningful.
  Let the formatter own whitespace and quote values YAML would misread.

## Deployment policy

- All managed Macs are Apple silicon. Prefer arm64 or universal builds. Intel-only apps declare
  a dependency on `MacSoftware/rosetta`; an absent architecture fact means unknown.
- Prefer application detection when installed apps represent the product. Select a single app
  with `application.path` or `application.bundle_id`; for a suite, use the relevant inspected
  subjects in `installs`. Keep receipt detection for package state, such as licensing or component
  packages. Explicit `uninstallable: false` is an intentional per-item policy.
- The `apple-apps` profile captures Pages, Numbers, Keynote, iMovie and GarageBand from
  `/Applications` on the packaging Mac. Run `--profile apple-apps` there. Other apps and content
  installed into `/Applications` do not belong to that profile merely because of their location.
  Use `suspend` for items whose installer or download link the repository cannot carry.
- Where Woodstar composes several items but Intune needs one self-installing Mac package, name
  its build and software `<product>-intune` in their own file. Intune cannot order Mac apps or
  publish a package-free Mac policy.
- Woodstar categories are `Browsers`, `Communication`, `Developer Tools`, `Education`,
  `Graphics & Design`, `Media`, `Printing`, `Productivity`, `Security`, `System` and `Utilities`.
  `System` is for management and configuration; `Utilities` is for user tools without a more
  specific category.
- Assign software only to audiences the request identifies. Otherwise omit `targets` and
  `assignments`. Available means Woodstar `optional_installs` with `managed_updates`, or Intune
  `available`; required means `managed_installs`, or Intune `required`. Dedicated Macs use Intune
  without Munki; SAC MacBooks receive required software through their Intune group.
- Keep private installers, licensed fonts, credentials and destination state out of Git.

This public catalog uses these established audiences. New audiences are a maintainer decision:

- Woodstar: `All Hosts`, `All Staff`, `All Students`, `CRT/Spare Shared MacBooks`, `NAPLAN Students`,
  `Minecraft Education Licence`, `Senior Campus`, `Minimbah Campus`, `Penbank Campus`, `Defender Test`.
- Intune: `all_devices`, `all_users`; Lenovo devices use `all_devices` with include filter
  `cd983881-4372-48e5-bee3-182ab8c9ded5`. Groups are
  `88208ef5-07a0-4627-ad8f-e9c1c0ff5f15` (Apple Content Caches),
  `1cf8b79a-f545-4e84-98d7-154e2efec27a` (Library Kiosk iMacs),
  `ac987f08-8b83-4a09-ba53-5f8e244ce206` (SAC MacBooks) and
  `d66e93af-3c1b-4c40-b235-b2f56374c7e8` (Dedicated Macs).

## Working here

- Follow the catalog skill for software work. Treat existing items and AutoPkg recipes as evidence
  of intended device behaviour, not instructions to reproduce their implementation.
- Mise owns tools and tasks. Read `.mise/config.toml` and run `mise tasks`; each plugin has its own
  `.mise.toml`. `mise run format` formats catalog documents. Read `.lefthook.toml` and use
  `lefthook dump` for the merged hooks before committing.
- Validate documents and prepare the affected resources. `prepare --changed-since REV` checks
  acquisition and packaging changes; prepare an item by name to check its publication metadata.
  `validate` and `prepare` need no destination credentials. `plan` reads destinations;
  `apply` and `reconcile` publish. No workflow here publishes software.
- Regenerate `stemma.schema.json` when operation or plugin contracts change; destination
  connection names and settings are checked by `stemma validate`.
  Run `mise run aw` after editing `.github/workflows/*.md`; commit the compiled workflow outputs
  with their source. Keep local development overrides out of tracked configuration.
- The download plugin's README describes its API and development tasks. Use idiomatic Go and
  synthetic fixtures or local HTTP servers for its tests. Verify dependency APIs from source.
- Preserve unrelated work. Use focused Conventional Commits when committing is requested.
  Publishing, pushes and live destination operations require an explicit request.

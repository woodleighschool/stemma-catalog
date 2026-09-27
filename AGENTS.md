# AGENTS.md

This catalog describes our real applications while Stemma remains unreleased. Expect the documents and the sibling `../stemma` implementation to change together.

- Follow the [stemma-catalog skill](skills/stemma-catalog/SKILL.md) when adding, changing or reviewing software.
- Keep standalone managed items in `software/<slug>.yaml`, one item per file. Use folders for real groups, such as platform editions, an application with its dedicated configuration, a product suite, campus printer configurations, or package builds with their inputs. Use `microsoft-365/` for the productivity suite, `windows-app/` for the app and RDS configuration, and `printers/` for campus configurations. Keep Mail2Outlook and Epson drivers standalone. Independent policy items have their own files even when they depend on another application. Keep resource-relative asset paths correct when moving documents. The root `kind: Project` owns imports and connections.
- All managed Macs are Apple silicon: use the arm64 or universal build and don't ask about Intel. Target `All Hosts` with `optional_installs` and `managed_updates` unless the request names a narrower audience. New Woodstar targets use only the `All Hosts`, `All Staff` and `All Students` labels.
- Treat the AutoPkg chains in `../autopkg` as evidence. Preserve their intended device behaviour; remove processor choreography that exists only to expose or copy artifacts.
- Use `mise tasks` and read `.mise/config.toml`. Run Stemma through `mise exec --`, which loads `.env`. Lefthook extends the shared organisation hooks; read `.lefthook.toml` and use `lefthook dump` when merged hook behaviour matters. Run `mise run format` before finishing a change.
- Finish a change with `stemma validate` and `stemma prepare --changed-since origin/main`. Pull requests run both without destination credentials.
- Keep private installers, licensed fonts, credentials and destination state out of Git. No live destinations, commits, pushes or deployment without explicit instruction.

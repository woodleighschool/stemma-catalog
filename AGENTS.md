# AGENTS.md

This catalog describes our real applications while Stemma remains unreleased. Expect the documents and the sibling `../stemma` implementation to change together.

- Keep standalone managed items in `software/<slug>.yaml`, one item per file. Use folders for real groups, such as platform editions, an application with its dedicated configuration, a product suite, campus printer configurations, or package builds with their inputs. Use `microsoft-365/` for the productivity suite, `windows-app/` for the app and RDS configuration, and `printers/` for campus configurations. Keep Mail2Outlook and Epson drivers standalone. Independent policy items have their own files even when they depend on another application. Keep resource-relative asset paths correct when moving documents. The root `kind: Project` owns imports and connections.
- Treat the AutoPkg chains in `../autopkg` as evidence. Preserve their intended device behaviour; remove processor choreography that exists only to expose or copy artifacts.
- Challenge the schema when it makes an ordinary policy awkward. Raise a gap rather than inventing a no-op source, weakening verification or copying private inputs to make validation pass.
- Keep metadata in native destination vocabulary. A software dependency is not an acquisition step dependency.
- Use `mise tasks` and read `.mise/config.toml`. Lefthook extends the shared organisation hooks; read `.lefthook.toml` and use `lefthook dump` when merged hook behaviour matters. Run `mise run format` before finishing a change.
- Keep private installers, licensed fonts, credentials and destination state out of Git. No live destinations, downloads, commits, pushes or deployment without explicit instruction.

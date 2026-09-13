# AGENTS.md

This catalog describes our real applications while Stemma remains unreleased. Expect the documents and the sibling `../stemma` implementation to change together.

- Own one managed item per `MacSoftware` or `WindowsSoftware` document. Use lowercase kind filenames within each application family; related documents of the same kind share a YAML stream. Keep `BuildMacPkg` documents beside their consumers. Keep its payload and scripts beside it. The root `kind: Project` owns imports and connections.
- Treat the AutoPkg chains in `../autopkg` as evidence. Preserve their intended device behaviour; remove processor choreography that exists only to expose or copy artifacts.
- Challenge the schema when it makes an ordinary policy awkward. Raise a gap rather than inventing a no-op source, weakening verification or copying private inputs to make validation pass.
- Keep metadata in native destination vocabulary. A software dependency is not an acquisition step dependency.
- Use `mise tasks` and read `.mise/config.toml`.
- Keep private installers, licensed fonts, credentials and destination state out of Git. No live destinations, downloads, commits, pushes or deployment without explicit instruction.

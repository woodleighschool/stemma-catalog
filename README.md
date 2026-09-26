# stemma-catalog

Our software catalog, its Stemma plugins and an agent skill for Stemma catalogs.

## 📚 Catalog

`stemma.yaml` owns imports, shared settings and destination connections.
`software/` contains the managed items and their package inputs; `icons/` contains
reviewed artwork. Standalone items use `software/<slug>.yaml`; folders group
related product editions, application configuration, suites and package inputs.
`microsoft-365/`, `windows-app/` and `printers/` group related items; Mail2Outlook
and Epson drivers remain standalone. Resource identities do not depend on directory names.

## 🧩 Plugins

`plugins/` contains separate plugin projects with their own Mise tools and tasks.
Build the download resolvers before validating the catalog:

```sh
mise run //plugins/downloads:build
stemma schema --offline --output-file stemma.schema.json
stemma validate --offline
```

See the [download resolvers](plugins/downloads/README.md) for configuration and
packaging. Keep credentials and local tool paths in `.env` or the shell environment.

## 🤖 Agent skill

`skills/stemma-catalog` guides coding agents through adding and maintaining software
in a Stemma catalog: finding vendor sources, writing and ordering documents, checking
prepared artifacts and deciding when a vendor needs a resolver plugin. It works from
the installed Stemma's operations and schema, so it suits catalogs with other plugins.
Install it for another catalog with:

```sh
pnpm dlx skills add woodleighschool/stemma-catalog
```

## 📝 Editor schema

YAML files reference the tracked root `stemma.schema.json` through its
[raw repository URL](https://raw.githubusercontent.com/woodleighschool/stemma-catalog/main/stemma.schema.json)
in their `$schema` modelines. Run `mise run schema` after changing plugin types,
registrations or destination names, then commit the regenerated schema. The command
uses the plugins configured in `stemma.yaml`; `.stemma/` contains disposable cache.

## 🧑‍💻 Development

Mise owns the toolchain and commands; `mise install` also installs the Git hooks:

```sh
mise install
mise run format
```

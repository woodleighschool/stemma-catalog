# stemma-catalog

Our software catalog and Stemma plugins.

## 📚 Catalog

`stemma.yaml` owns imports, shared settings and destination connections.
`software/` contains the managed items and their package inputs; `icons/` contains
reviewed artwork. Resource identities do not depend on directory names.

## 🧩 Plugins

`plugins/` contains separate plugin projects with their own Mise tools and tasks.
Build the download resolvers before validating the catalog:

```sh
mise run //plugins/downloads:build
stemma validate --offline
```

See the [download resolvers](plugins/downloads/README.md) for configuration and
packaging. Keep credentials and local tool paths in `.env` or the shell environment.

# stemma-catalog

Our Stemma catalog, vendor download resolvers and a reusable catalog-maintenance skill.
The catalog is an opinionated example; [AGENTS.md](AGENTS.md) describes our deployment policy.

## 📚 Catalog

`stemma.yaml` owns imports, shared settings, plugins and destination connections.
`software/` contains the managed items and package inputs; `icons/` contains reviewed artwork.
`stemma update` records reviewed sources under `stemma.lock/`; preparation consumes those locks.
`stemma cache path` prints the disposable cache location. `.stemma/` holds local command state.

```sh
stemma update
stemma prepare
```

The `apple-apps` profile captures Apple's apps from `/Applications` on the packaging Mac.
Run `stemma update --profile apple-apps` and `stemma prepare --profile apple-apps` there.
`stemma plan` compares publications with their destinations; `stemma apply` publishes them.

## 🧩 Plugins

The Project's `plugins` declarations choose the implementations used by the catalog.
See the [download resolvers](plugins/downloads/README.md) for their configuration and development.
Keep credentials and local tool paths in the environment or ignored local configuration.

## 🤖 Agent skill

The [stemma-catalog skill](skills/stemma-catalog/SKILL.md) supports creating and maintaining
catalogs with their own policies, plugins and destinations. It covers source discovery,
resource composition, artifact inspection and publication metadata. Install it with:

```sh
pnpm dlx skills add woodleighschool/stemma-catalog
```

## 📝 Editor schema

Each YAML document references the tracked `stemma.schema.json` through its
[raw repository URL](https://raw.githubusercontent.com/woodleighschool/stemma-catalog/main/stemma.schema.json).
Regenerate it with `stemma schema --output-file stemma.schema.json` when configured operations
or their schemas change, and commit it with those changes. `mise run schema` runs that command.

## 🧑‍💻 Development

Mise owns the toolchain and commands; `mise install` also installs the Git hooks:

```sh
mise install
mise run format
```

## 🤖 Software requests

The `software-request` issue label starts the request workflow. It opens a pull request for a
settled request or asks on the issue and adds `needs-input`; a reply continues that request.
Removing and re-adding `software-request` retries a stopped run. Only repository writers can
start the workflow.

Edit `.github/workflows/*.md`, then run `mise run aw`. Commit the compiled `.lock.yml` files and
`.github/aw/actions-lock.json` with their source. CI checks for stale output; Renovate recompiles
when updating the compiler or action references.

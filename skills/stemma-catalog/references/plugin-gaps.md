# When Stemma is not enough

Most catalog problems are source problems, and most source problems have a generic answer. Before
proposing a plugin, check:

1. Does an available Homebrew, WinGet or vendor resolver select the right release and artifact?
   Prefer a source with a published digest so update checks do not need installer downloads.
2. Can a plain `url`, `github` or `http` source express discovery without depending on details of today's
   page?
3. Could an existing resolver cover it with a small extension, such as a new product value?
4. Is what's missing stable, vendor-specific discovery that other vendors don't share?

When the answer to 4 is yes and the earlier options do not fit, propose a resolver plugin with a
clear owner. A catalog may keep one locally or consume a separately maintained plugin; do not
assume plugin code or build tooling already exists. When the missing behaviour
would serve many vendors, such as a common feed format or release API, propose it as a Stemma
feature instead.

## Kinds of gap

| Gap           | Sign                                                                           | Proposal                                              |
| ------------- | ------------------------------------------------------------------------------ | ----------------------------------------------------- |
| Resolver      | The generic resolvers can't find or pin the vendor's download                  | A resolver plugin, or a new value in an installed one |
| Resource kind | Preparation fits no kind: the artifact needs assembly no kind supports         | A resource-kind plugin                                |
| Destination   | The publishing target isn't supported                                          | A destination plugin                                  |
| Stemma        | An ordinary policy needs a workaround, such as a no-op source or copied values | A Stemma change                                       |

Resolver gaps are by far the most common.

## Propose a resolver

Describe the contract. Write code only when asked.

```text
Proposed resolver: vendorname

Purpose:
  Resolve the latest stable macOS installer from Vendor's release API.

Inputs:
  product       required string
  channel       optional enum, default stable
  architecture  optional enum

Observation:
  immutable release or build identifier
  filename
  vendor version where available

Evidence:
  vendorname.version

Content and download, when the vendor supplies a digest:
  published SHA-256, stable filename and download URL

Why a plugin:
  The release API needs two dependent requests and returns no download link that a
  generic http match could select without embedding a transient URL.
```

A sound resolver contract:

- discovers exactly one release, using release records without downloading the installer when
  possible;
- supplies published content identity and a supported download handoff so Stemma can update the
  lock before acquisition; when no trusted digest exists, acknowledges that update must fetch and
  hash the artifact;
- marks the observation immutable when it always fetches the same bytes;
- returns the vendor's version as namespaced evidence that destination metadata can use;
- keeps credentials, signed URLs and per-request tokens out of observations and evidence;
- takes inputs that select a product, not URLs or patterns that restate the vendor's site.

To build it when requested, follow any repository instructions and reuse an existing plugin project
when appropriate. Otherwise create a separately owned implementation using the
[plugin documentation](https://woodleighschool.github.io/stemma/writing-plugins) for the installed
Stemma version.

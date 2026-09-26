# When Stemma is not enough

Most catalog problems are source problems, and most source problems have a generic answer. Before
proposing a plugin, check:

1. Can a plain `url` source fetch the current release?
2. Can the `github` or `http` resolver express the discovery without depending on details of today's
   page?
3. Does an installed resolver already model this vendor, or could it with a small extension, such as
   a new product value?
4. Is what's missing stable, vendor-specific discovery that other vendors don't share?

When the answer to 4 is yes, propose a repository-local resolver plugin. When the missing behaviour
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

Why a plugin:
  The release API needs two dependent requests and returns no download link that a
  generic http match could select without embedding a transient URL.
```

A sound resolver contract:

- discovers without downloading and returns an observation naming exactly one release;
- marks the observation immutable when it always fetches the same bytes;
- returns the vendor's version as namespaced evidence that destination metadata can use;
- keeps credentials, signed URLs and per-request tokens out of observations and evidence;
- takes inputs that select a product, not URLs or patterns that restate the vendor's site.

To build it, follow the repository's instructions and existing plugin projects, and the
[plugin documentation](https://woodleighschool.github.io/stemma/writing-plugins) for the installed
Stemma version.

# Gaps in Stemma or a resolver

First check whether the available operations express the intended release and deployment
behaviour. A built-in source, vendor registry or small extension to an existing resolver may
already cover it; see [discovery.md](discovery.md).

Choose the owner of the missing behaviour:

| Missing contract                                                                       | Owner           |
| -------------------------------------------------------------------------------------- | --------------- |
| Vendor-specific product selection, dependent API requests or authenticated acquisition | Resolver plugin |
| Common feed format or generally useful acquisition/inspection behaviour                | Stemma          |
| Artifact assembly that no existing builder expresses                                   | Resource kind   |
| Publication to another service                                                         | Destination     |

An installed plugin has no incumbency advantage. Remove catalog-owned discovery code when a
native operation expresses the same semantics. Conversely, keep a vendor model when flattening
it into URLs would lose product, channel or acquisition behaviour.

For a resolver proposal, state the product it selects, required configuration, discovery
mechanism and how acquisition retrieves that exact selection. Explain which available source
cannot express it. Distinguish a malformed vendor artifact from an inspection bug using the
failing file, command and error; debugging Stemma can then stay within the agreed scope.

A resolver contract should establish:

- one selected release, with its immutable identity and filename;
- a trusted content digest when the publisher supplies one, or the need to acquire and hash bytes;
- a stable download handoff, refreshing expiring URLs for the selected release when needed;
- namespaced version evidence that destinations can use;
- credentials in environment-backed configuration, outside observations and evidence.

Configuration selects products and release policies; it should not make each catalog reconstruct
the vendor's API choreography. Implement a plugin only within the requested scope, following the
[plugin contract](https://woodleighschool.github.io/stemma/writing-plugins) and relevant source.

# RFC 014: Backwards Compatibility

## Background

Contrast's components currently communicate via gRPC APIs using aTLS, where the attestation evidence is embedded inside the TLS handshake.
This makes it practically infeasible for third parties to reimplement components to fit their needs, for example a custom CLI replacement or an alternative initializer.
Even for us, the current setup complicates additional, alternative components like the planned CVM support.
The gRPC/aTLS approach additionally makes it difficult to front Contrast with a reverse proxy, which customers have expressed a desire for in the past.

For these reasons, we're currently planning to introduce, and migrate to, an HTTP API.
As a necessary precondition, this RFC defines the versioning scheme to be used by said HTTP API.
Currently, the gRPC/aTLS couplings are versioned solely through Contrast release versions, and even then backwards compatibility isn't clearly defined and can break, as the description in [this issue](https://linear.app/edgelesssys/issue/CON-113/spike-get-an-overview-of-code-that-might-break-backwards-compatibility) shows.

The goal of this RFC is to prevent a similar situation for the planned HTTP API.
It defines how endpoints should be versioned, and how components should negotiate a shared version.
As a result, Coordinators and clients of different versions will be able to communicate,
and we will be able to introduce additional functionality easily and in a properly versioned manner, while guaranteeing that old functionality isn't broken unexpectedly.

## Proposal

Ownership of the client logic used to communicate with the Coordinator moves to the SDK for the HTTP API.
This differs from the current layout, where each component owns their own, bespoke implementation of gRPC API client logic.
The SDK becomes the sole owner of the client logic, and all Contrast components currently communicating via the gRPC User API (especially the CLI and initializer) will consume these SDK functions.
Thus we eventually (once gRPC is fully gone) will end up with exactly one implementation of the protocol layer to version and test.

In the medium-term, the goals are:
- Every component still speaks gRPC/aTLS exactly as today.
- Every component can additionally speak HTTP through the SDK.
- Components communicate by the newest mutually supported HTTP API version, or fall back to gRPC, automatically.

For the split gRPC/aTLS + HTTP approach we already have a precedent in the existing codebase with the Coordinator's `POST /attest` endpoint
and the corresponding `GetAttestation` / `ValidateAttestation` functions in the SDK doing application-level attestation over plain HTTP.
We will extend these patterns to the UserAPI.

A table of currently planned HTTP API endpoints is shown below.

| Operation | gRPC (aTLS) | HTTP | Status |
| --- | --- | --- | --- |
| Capabilities | n/a | `GET /capabilities` | released |
| GetManifests | `userapi.GetManifests` | `POST /attest` / `POST /v1/attest` | released / planned |
| SetManifest | `userapi.SetManifest` | `POST /v1/manifest` | planned |
| Recover | `userapi.Recover` | `POST /v1/recover` | planned |
| NewMeshCert | `meshapi.NewMeshCert` | `POST /v1/mesh/cert` | planned |

While the design of the API and its security mechanisms will be the topic of a separate RFC, the endpoints and handling will share a common shape:
- Each handler should reuse the internal logic from the existing gRPC methods.
- For each handler, a corresponding SDK client method is exposed publicly.
- Contrast components should use the new methods if possible, or fallback to gRPC.

### Versioning scheme

The old API contract is implicitly defined by aTLS internals, the proto definitions, and the gRPC library version.
The new API contract is constituted solely by the HTTP verbs and paths, the JSON message shapes, and the error types the API might return.
This contract lives fully in a new `apitypes` package.

Additive changes are possible within an API version: new, optional fields may be added.
Non-additive, breaking changes are prohibited within minor versions, and include:
- The addition of new, mandatory fields.
- The removal of fields.
- The repurposing of fields.
- The changing of API paths.

In order to make a change like this, a new major version of the API is required.
Rather than changing such implementation details in the `apitypes` package directly and releasing the changes as a new major version, a subpackage for each API version is added, namely `apitypes/apiv1` and so forth.
The purpose of this is to allow the SDK to easily import and support past API versions.
For clearer public-facing versioning, the SDK `semver` tracks the API version.
New API versions require new major SDK versions, additive intra-version changes only require minor version bumps.

The SDK exposes each API version via the client library, for example:
```go
func (c *Client) V1() *apiv1.API {
	return apiv1.New(c.httpapi)
}
```

In this way, an arbitrary number of API versions can be exposed simultaneously, and versions can be deprecated as needed.
Consumers can thus choose to interact with a specific version of the API, for example:
```go
c := client.New(...)
c.V1().SetManifest(ctx, req)
```
However, the SDK additionally exposes "unversioned" variants of the endpoints:
```go
c := client.New(...)
c.SetManifest(ctx, req)
```
These variants autonegotiate the newest shared version between client and Coordinator before calling the corresponding versioned functions.
Generally, this mechanism is to be preferred.

### Automatic version negotiation

From the above table, note that `/capabilities` is the only newly introduced endpoint that's unversioned.
This is intentional: the endpoint serves as a way for clients to discover what versions the Coordinator speaks.

Under `GET /capabilities`, the Coordinator handles capability inquiries by returning `sdk.supportedAPIVersions`.
This is simply a list of all API versions the SDK knows about and supports, currently:
```go
var supportedAPIVersions = []string{apiv1.Version}
```
The client then simply selects the newest version both it and the Coordinator support, and makes requests via that version's functions.
The capabilities response may be cached for reuse.

In the case of the example from above, the "unversioned" `SetManifest` version looks something like this:
```go
func (c *Client) SetManifest(...) {
    version, _ := c.NegotiateAPIVersion(...)
    switch version {
    case apiv1.Version:
	    return c.V1().SetManifest(ctx, req)
    default:
	    return nil, fmt.Errorf("SetManifest is not implemented for API version %q", version)
    }
}
```

To mitigate downgrade attacks on this negotiation, two mechanisms are supported.

First, users may pin the minimum required API version in the manifest via a new `MinimumAPIVersion` field.
If the newest API version supported by both client and Coordinator is less than this field, communication via HTTP API isn't possible, and the client should throw an appropriate error.

Even though we could, at least for the time being, fall back to the gRPC API in this case, this would ignore that a pinned minimum version clearly indicates the user's desire to use the HTTP API,
thus likely meaning the user also expects the Coordinator to speak this version; moreover, the error-on-minimum-unmet behavior would need to be implemented anyways once gRPC is deprecated.
Falling back to gRPC *in the case of a pinned minimum version* would therefore be a potentially unexpected, temporary behavior and shouldn't be introduced.

Second, the hash of the Coordinator `/capabilities` response is included in `ConstructReportData`.
This allows clients to verify that the capabilities response they received during version negotiation was genuine.

### gRPC deprecation path

Once the HTTP API is fully implemented and tested, the gRPC API can be deprecated in the medium-term future, after a sufficient transition period.
The planned timeline for this looks as follows.

0. Fully implement the HTTP API.
1. Use the HTTP API by default in all our e2e tests.
2. After 2 weeks of no (API-related) e2e failures, remove the `contrast_unstable_api` tags and default to the HTTP API with the next minor release.
   Publicly announce the change in the changelog, mark as breaking change, and include a notice informing readers of the deprecation and planned removal for the second-next minor release.
   Also print an appropriate warning in the CLI whenever gRPC fallback is being used.
3. With the specified minor release, remove all gRPC related code, again marking the PR as breaking change.

### Testing strategy

A new e2e test is introduced with the purpose of testing that communication between clients and Coordinators with different supported API versions succeeds.
For each supported API version in the SDK, plus a gRPC-only case, a corresponding test case starts a Coordinator supporting all versions *up to* the specified version.
All user API functionalities (`GetManifests`, `SetManifest`, `Recover`) are tested for each supported SDK version (via the explicitly versioned SDK methods),
as well as using the automatic version negotiation paths for overlap between versions, and disjunct supported versions.

While the gRPC API is still supported, automatic negotiation failure should fall back to gRPC.
Afterward, this simply becomes an expected error case, and the gRPC-only Coordinator test case can be removed.

Additionally, the release e2e test should gain an new step in which the to-be-released deployment files are tested via an older, released version of the Contrast CLI, and vice versa.
This is intended to prevent accidental breaking changes between the *same* versions of the API in *different* Contrast releases.

## Out of scope

Coordinator-to-Coordinator traffic (for example peer recovery) can keep using aTLS for now, since both ends are always ours.

## Alternatives considered

In regards to the gRPC deprecation path, we could also consider taking this as reason to bump to Contrast `v2.0.0`.
We could do so essentially immediately after step fully implementing and testing the HTTP API.
While we would have to support `v1` with security updates for at least some time,
this would eliminate the need for the temporary gRPC and HTTP double support within the same version.

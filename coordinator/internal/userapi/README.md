# User API

The `userapi.UserAPI` service serves requests from Contrast users (data owners, workload owners and seed share owners).
A detailed description of this package's layout and responsibilities resides in the package comment.
Below are some sequence diagrams that illustrate the data flow for each request type.

## `GetManifests`

Primary audience of this RPC are data owners that want to verify the Coordinator, inspect the current manifest and retrieve the CA certificates.

```mermaid
sequenceDiagram
    participant user
    box Grey Coordinator
        participant userapi
        participant stateguard
        participant history
    end
    user->>+userapi: start aTLS handshake
    userapi->>user: finish aTLS handshake
    user->>userapi: GetManifestsRequest
    userapi->>+stateguard: GetManifests
    stateguard->>+history: read manifests and policies
    history->>-stateguard: manifests and policies
    stateguard->>-userapi: Manifests+Policies
    userapi->>-user: GetManifestsResponse
```

## `SetManifest`

This RPC allows workload owners to update the current manifest.

```mermaid
sequenceDiagram
    participant user
    box Grey Coordinator
        participant userapi
        participant stateguard
        participant history
    end
    user->>+userapi: start aTLS handshake
    userapi-->>userapi: configure handler with client pubkey
    userapi->>-user: finish aTLS handshake

    user->>+userapi: SetManifestRequest
    userapi->>+stateguard: GetState
    stateguard->>-userapi: State
    userapi-->>userapi: authorize client with<br/>workload owner keys in state
    userapi->>+stateguard: update state
    stateguard->>history: store manifest and policies
    stateguard->>history: atomically swap state
    stateguard-->>stateguard: atomically swap internal state
    stateguard->>-userapi: new state
    userapi-->>userapi: extract CA certs from new state
    userapi->>-user: SetManifestResponse
```

## Recover

This RPC allows seed share owners to recover a Coordinator that lost its internal state after a restart.

```mermaid
sequenceDiagram
    participant user
    box Grey Coordinator
        participant userapi
        participant stateguard
        participant history
    end
    user->>+userapi: start aTLS handshake
    userapi-->>userapi: configure handler with client pubkey
    userapi->>-user: finish aTLS handshake

    user->>+userapi: RecoverRequest
    userapi->>+stateguard: ResetState
    stateguard->>+history: get unverified state
    history->>-stateguard: unverified state
    stateguard->>userapi: authorize client with seedshare<br/>owner keys in unverified state
    userapi->>stateguard: seedengine + mesh CA key
    stateguard->>+history: get state with seedengine
    history->>-stateguard: verified state
    stateguard-->>stateguard: check: verifed state == unverified state
    stateguard-->>stateguard: atomically swap internal state
    stateguard->>-userapi: State
    userapi->>-user: RecoverResponse
```

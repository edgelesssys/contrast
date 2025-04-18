# User API

The `userapi.UserAPI` service serves requests from Contrast users (data owners, workload owners and seedshare owners).
A detailed description of this package's layout and responsibilities resides in the package comment.
Below are some sequence diagrams that illustrate the data flow for each request type.

## `GetManifests`

Primary audience of this RPC are data owners that want to verify the Coordinator, inspect the current manifest and retrieve the CA certificates.

```mermaid
sequenceDiagram
    participant user
    box Grey Coordinator
        participant userapi
        participant authority
        participant history
    end
    user->>+userapi: start aTLS handshake
    userapi->>user: finish aTLS handshake
    user->>userapi: GetManifestsRequest
    userapi->>+authority: GetState
    authority->>-userapi: State
    userapi->>+history: read manifests and policies
    history->>-userapi: manifests and policies
    userapi->>-user: GetManifestsResponse
```

## `SetManifest`

This RPC allows workload owners to update the current manifest.

```mermaid
sequenceDiagram
    participant user
    box Grey Coordinator
        participant userapi
        participant authority
        participant history
    end
    user->>+userapi: start aTLS handshake
    userapi-->>userapi: configure handler with client pubkey
    userapi->>-user: finish aTLS handshake

    user->>+userapi: SetManifestRequest
    userapi->>+authority: GetState
    authority->>-userapi: State
    userapi-->>userapi: authorize client with<br/>workload owner keys in state
    userapi->>history: store manifest and policies
    userapi->>+authority: update state
    authority->>history: atomically swap state
    authority-->>authority: atomically swap internal state
    authority->>-userapi: new state
    userapi-->>userapi: extract CA certs from new state
    userapi->>-user: SetManifestResponse
```

## Recover

This RPC allows seedshare owners to recover a Coordinator that lost its internal state after a restart.

```mermaid
sequenceDiagram
    participant user
    box Grey Coordinator
        participant userapi
        participant authority
        participant history
    end
    user->>+userapi: start aTLS handshake
    userapi-->>userapi: configure handler with client pubkey
    userapi->>-user: finish aTLS handshake

    user->>+userapi: RecoverRequest
    userapi->>+history: get unverified state
    history->>-userapi: unverified state
    userapi-->>userapi: authorize client with seedshare<br/>owner keys in unverified state
    userapi-->>userapi: create seedengine from request
    userapi->>+history: get state with seedengine
    history->>-userapi: verified state
    userapi-->>userapi: check: verifed state == unverified state
    userapi->>+authority: ResetState
    authority-->>authority: atomically swap internal state
    authority->>-userapi: State
    userapi->>-user: RecoverResponse
```

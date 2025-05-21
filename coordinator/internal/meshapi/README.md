# Mesh API

The `meshapi.MeshAPI` service serves requests from Contrast workloads and other Coordinators.
Below are some sequence diagrams that illustrate the data flow for each request type.

## `NewMeshCert`

This RPC is called by initializers that want to act as Contrast workloads.

```mermaid
sequenceDiagram
    participant initializer
    box Grey Coordinator
        participant stateguard
        participant meshapi
    end

    initializer->>+stateguard: start aTLS handshake
    stateguard-->>+meshapi: configure handler with state
    stateguard->>-initializer: finish aTLS handshake

    initializer->>meshapi: NewMeshCertRequest
    meshapi-->>meshapi: create cert with<br/>CA from state
    meshapi->>-initializer: NewMeshCertResponse
```

## `Recover`

This RPC is called by peer Coordinators that need to recover their internal state.
In the diagram below, this is `Coordinator1` attempting to recover from `Coordinator2`.

```mermaid
sequenceDiagram
    box Grey Coordinator1
        participant history1 as history
        participant stateguard1 as stateguard
        participant recovery1 as recovery
    end

    box Grey Coordinator2
        participant stateguard2 as stateguard
        participant meshapi2 as meshapi
    end

    # activate recovery1
    recovery1-->>+recovery1: observe stale state

    recovery1->>+stateguard1: ResetState
    stateguard1->>+history1: get unverified state
    history1->>-stateguard1: unverified state
    stateguard1->>recovery1: authorize peer

    recovery1-->>recovery1: construct aTLS validator<br/>from unverified state
    recovery1->>+stateguard2: start aTLS handshake
    stateguard2-->>+meshapi2: configure handler with state
    stateguard2->>-recovery1: finish aTLS handshake

    recovery1->>meshapi2: RecoverRequest
    meshapi2-->>meshapi2: authorize peer by manifest
    meshapi2-->>meshapi2: extract seed and mesh<br/>key from state
    meshapi2->>-recovery1: RecoverResponse

    recovery1-->>recovery1: construct seedengine from response
    recovery1->>stateguard1: seedengine + mesh CA key
    stateguard1->>+history1: get state with seedengine
    history1->>-stateguard1: verified state
    stateguard1-->>stateguard1: check:<br/>verifed state == unverified state
    stateguard1-->>stateguard1: atomically swap internal state
    stateguard1->>-recovery1: State
    deactivate recovery1
```

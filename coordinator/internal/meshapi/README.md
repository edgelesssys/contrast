# Mesh API

The `meshapi.MeshAPI` service serves requests from Contrast workloads and other Coordinators.
Below are some sequence diagrams that illustrate the data flow for each request type.

## `NewMeshCert`

This RPC is called by initializers that want to act as Contrast workloads.

```mermaid
sequenceDiagram
    participant initializer
    box Grey Coordinator
        participant authority
        participant meshapi
    end

    initializer->>+authority: start aTLS handshake
    authority-->>+meshapi: configure handler with state
    authority->>-initializer: finish aTLS handshake

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
        participant authority1 as authority
        participant enterprise1 as enterprise
    end

    box Grey Coordinator2
        participant authority2 as authority
        participant meshapi2 as meshapi
    end

    # activate enterprise1
    enterprise1-->>+enterprise1: observe stale state
    enterprise1->>+history1: get unverified state
    history1->>-enterprise1: unverified state

    enterprise1-->>enterprise1: construct aTLS validator<br/>from unverified state
    enterprise1->>+authority2: start aTLS handshake
    authority2-->>+meshapi2: configure handler with state
    authority2->>-enterprise1: finish aTLS handshake

    enterprise1->>meshapi2: RecoverRequest
    meshapi2-->>meshapi2: authorize peer by manifest
    meshapi2-->>meshapi2: extract seed and mesh<br/>key from state
    meshapi2->>-enterprise1: RecoverResponse

    enterprise1-->>enterprise1: construct seedengine from response
    enterprise1->>+history1: get state with seedengine
    history1->>-enterprise1: verified state
    enterprise1-->>enterprise1: check:<br/>verifed state == unverified state
    enterprise1->>+authority1: ResetState
    authority1-->>authority1: atomically swap internal state
    authority1->>-enterprise1: State
    deactivate enterprise1
```

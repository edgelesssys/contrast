# Contrast Coordinator

The Coordinator is the central service in a Contrast deployment.
It exposes an API to set manifests and to get the history of manifests, and it verifies workloads according to the current manifest.
The below diagram shows how the Coordinator packages interact in the enterprise version.

```mermaid
graph TB
    subgraph Coordinator
            userapi -->|update state in<br/>reset state in| stateguard
            meshapi -->|get state from| stateguard
            stateguard -->|manage latest state in| history
            history -->|store data by hash<br/>atomically update state| store
    end

    subgraph Workload
        initializer -->|NewMeshCert| meshapi
        main
    end

    subgraph Coordinator2
        enterprise2(enterprise/recovery)
        stateguard2(stateguard)
        enterprise2 -->|reset state in| stateguard2
    end

    subgraph Kubernetes
        Workload
        Coordinator
        Coordinator2
        ConfigMap
    end


    store -->|store data in<br/>receive data from| ConfigMap

    user -->|GetManifests<br/>SetManifest<br>Recover| userapi

    enterprise2 -->|Recover| meshapi
```

# RFC 009: Distributed coordinator

## Background

The Contrast Coordinator is a stateful service with a backend storage that can't be shared.
If the Coordinator pod restarts, it loses access to the secret seed and needs to be recovered manually.
This leads to predictable outages on AKS, where the nodes are replaced periodically.

## Requirements

* High-availability: node failure, pod migration, etc. must not impact the availability of the `set` or `verify` flow.
* Auto-recovery: node failure, pod migration, etc. must not require a manual recovery step.
  * A newly started Coordinator that has recovered peers must eventually recover automatically.
* Consistency: the Coordinator state must be strongly consistent.

## Design

### Persistency changes

The Coordinator uses a generic key-value store interface to read from and write to the persistent store.
In order to allow distributed read and write requests, we only need to swap the implementation for one that can be used concurrently.

```golang
type Store interface {
  Get(key string) ([]byte, error)
  Set(key string, value []byte) error
  CompareAndSwap(key string, oldVal, newVal []byte) error
  Watch(key string, callback func(newVal []byte)) error // <-- New in RFC009.
}
```

There are no special requirements for `Get` and `Set`, other than basic consistency guarantees (i.e., `Get` should return what was `Set` at some point in the past).
We can use Kubernetes resources and their `GET` / `PUT` semantics to implement them.
`CompareAndSwap` needs to do an atomic update, which is supported by the `ObjectMeta.resourceVersion`[^1] field.
We also add a new `Watch` method that facilitates reacting on manifest updates.
This can be implemented as a no-op or via `inotify(7)` on the existing file-backed store, and with the native watch mechanisms for Kubernetes objects.

[RFC 004](004-recovery.md#kubernetes-objects) contains an implementation sketch that uses custom resource definitions.
However, in order to keep the implementation simple, we implement the KV store in content-addressed `ConfigMaps`.
A few considerations on using Kubernetes objects for storage:

1. Kubernetes object names need to be valid DNS labels, limiting the length to 63 characters.
   We work around that by truncating the name and storing a map of full hashes to content.
2. The `latest` transition is not content-addressable.
   We store it under a fixed name for now (`transitions/latest`), which limits us to one Coordinator deployment per namespace.
   Should a use-case arise, we could make that name configurable.
3. Etcd has a limit of 1.5MiB per value.
   This is more than enough for policies, which are the largest objects we store right now at around 100KiB.
   However, we should make sure that the collision probability is low enough that not to many policies end up in the same config.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: "contrast-store-$(dirname)-$(truncated basename)"
  labels:
    app.kubernetes.io/managed-by: "contrast.edgeless.systems"
data:
  "$(basename)": "$(data)"
```

[^1]: <https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency>

### Recovery mode

The Coordinator is said to be in *recovery mode* if its state object (containing manifest and seed engine) is unset.
Recovery mode can be entered as follows:

* The Coordinator starts up in recovery mode.
* Receiving a watch event for the latest manifest, and the latest manifest is not the current one.
* Syncing the latest manifest during API calls and discovering a new manifest.

Recovery mode exits after the state has been set by:

* A successful [Peer recovery](#peer-recovery).
* A successful recovery through the `userapi.Recover` RPC.
* A successful initial `userapi.SetManifest` RPC.

### Service changes

Coordinators are going to recover themselves from peers that are already recovered (or set).
This means that we need to be able to distinguish Coordinators that have secrets from Coordinators that have none.
Since we're running in Kubernetes, we can leverage the built-in probe types.

We add a few new paths to the existing HTTP server for metrics, supporting the following checks (returning status code 503 on failure):

* `/probe/startup`: returns 200 when all ports are serving and [peer recovery](#peer-recovery) was attempted once.
* `/probe/liveness`: returns 200 unless the Coordinator is recovered but can't read the transaction store.
* `/probe/readiness`: returns 200 if the Coordinator is not in recovery mode.

Using these probes, we expose the Coordinators in different ways, depending on audience:

* A `cooordinator-ready` service backed by ready Coordinators should be used by initializers and verifying clients.
* A `coordinator-peers` headless service backed by ready Coordinators should be used by Coordinators to discover peers to recover from.
  Using a headless service allows getting the individual IPs of ready Coordinators, as opposed to a `ClusterIP` service that just drops requests when there are no backends.
* The `coordinator` service now accepts unready endpoints (`publishNotReadyAddresses=true`) and should be used for `set`.
  The idea here is backwards compatibility with documented workflows, see [Alternatives considered](#alternatives-considered).

### Manifest changes

In order to allow Coordinators to recover from their peers, we need to define which workloads are allowed to recover.
There are two reasons why we want to make this explicit:

1. Allowing a Coordinator with a different policy to recover is a prerequisite for automatic updates.
2. The Coordinator policy can't be embedded into the Coordinator during build, and deriving it at runtime is inconvenient.

Thus, we introduce a new field to the manifest that specifies roles available to the verified identity.
For now, the only known role is `coordinator`, but this could be extended in the future (for example: delegate CAs).

```golang
type PolicyEntry struct {
  SANs             []string
  WorkloadSecretID string `json:",omitempty"`
  Roles            []string `json:",omitempty"` // <-- New in RFC009.
}
```

During `generate`, we append the value of the `contrast.edgeless.systems/pod-role` annotation to the policy entry.
If there is no coordinator among the resources, we add a policy entry for the embedded coordinator policy.

### Peer recovery

The peer recovery process is attempted by Coordinators that are in [Recovery mode](#recovery-mode):

1. Once after startup, to speed up initialization.
2. Once as callback to the `Watch` event, .
3. Periodically from a goroutine, to reconcile missed watch events.

The process starts with a DNS request for the `coordinator-peers` name to discover ready Coordinators.
For each of the ready Coordinators, the recovering Coordinator calls a new `meshapi.Recover` method.

```proto
service MeshAPI {
  rpc NewMeshCert(NewMeshCertRequest) returns (NewMeshCertResponse);
  rpc Recover(RecoverRequest) returns (RecoverResponse); // <-- New in RFC009.
}

message RecoverRequest {}

message RecoverResponse {
  bytes Seed = 1;
  bytes Salt = 2;
  bytes RootCAKey = 3;
  bytes RootCACert = 4;
  bytes MeshCAKey = 5;
  bytes MeshCACert = 6;
  bytes LatestManifest = 7;
}
```

When this method is called, the serving Coordinator:

0. Queries the current state and attaches it to the context (this already happens automatically for all `meshapi` calls).
1. Verifies that the client identity is allowed to recover by the state's manifest (has the `coordinator` role).
2. Fills the response with values from the current state.

After receiving the `RecoverResponse`, the client Coordinator:

1. Sets up a temporary `SeedEngine` with the received parameters.
2. Verifies that the received manifest is the current latest.
   If not, it stops and enters recovery mode again.
3. Updates the state with the current manifest and the temporary seed engine.

If the recovery was successful, the client Coordinator leaves the peer recovery process.
Otherwise, it continues with the next available peer, or fails the process if none are left.

## Open issues

* TODO(burgerdev): inconsistent state if `userapi.Recover` is called while a recovered coordinator exists. Fix candidates:
  * Store the mesh cert alongside the manifest
  * Sign the latest transition with the mesh key (and resign on `userapi.Recover`).

## Alternatives considered

* TODO(burgerdev): different service structure
* TODO(burgerdev): push vs pull
* TODO(burgerdev): active mesh CA key distribution
* TODO(burgerdev): nested CAs, reuse `NewMeshCert`

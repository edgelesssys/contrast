# Recover Contrast Coordinator

This step describes how the Contrast Coordinator can be recovered after a restart.

## Applicability

This step is necessary only when the Coordinator needs to be restarted.

## Prerequisites

1. A running Contrast deployment

## How-to

This page guides you through the process of connecting to the Coordinator and restoring its state.

If the Contrast Coordinator restarts, it enters recovery mode and waits for an operator to provide key material.
For demonstration purposes, you can simulate this scenario by deleting the Coordinator pod.

```sh
kubectl delete pod -l app.kubernetes.io/name=coordinator
```

Kubernetes schedules a new pod, but that pod doesn't have access to the key material the previous pod held in memory and can't issue certificates for workloads yet.
You can confirm this by running `verify` again, or you can restart a workload pod, which should stay in the initialization phase.
However, you can recover the Coordinator using the secret seed and the seed share owner key in your working directory.

```sh
contrast recover -c "${coordinator}:1313"
```

Now that the Coordinator is recovered, all workloads should pass initialization and enter the running state.
You can now verify the Coordinator again, which should return the same manifest you set before.

:::warning

The recovery process invalidates the mesh CA certificate:
existing workloads won't be able to communicate with workloads newly spawned.
All workloads should be restarted after the recovery succeeded.

:::

### Recovering the manifest history

The Coordinator uses a key-value store backed by Kubernetes ConfigMaps to store the manifest history and policy information.
This key-value store is what allows the Coordinator to recover after a restart.
If the Coordinator StatefulSet is deleted, for example if the entire namespace is deleted, the ConfigMap store is lost and normal recovery isn't possible.
In order to recover the Coordinator state in this case, you can use the manifest history obtained by a previous `contrast verify` once the Coordinator is reapplied.

```sh
kubectl apply -n <namespace> -f verify/history.yml
contrast recover -c "${coordinator}:1313"
```

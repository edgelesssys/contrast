# Recover Contrast Coordinator

This step describes how the Contrast Coordinator can be recovered after a restart.

## Applicability

This step is necessary only when the Coordinator needs to be restarted.

## Prerequisites

1. A running Contrast deployment

## How-to

This page guides you through the process of connecting to the Coordinator and restoring its state.

### Connect to the Contrast Coordinator

The released Coordinator resource
includes a LoadBalancer definition we can use.

```sh
coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
```

:::info[Port-forwarding of Confidential Containers]

`kubectl port-forward` uses a Container Runtime Interface (CRI) method that isn't supported by the Kata shim.
If you can't use a public load balancer, you can deploy a port-forwarding pod to relay traffic to a Contrast pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: port-forwarder-coordinator
spec:
  containers:
    - name: port-forwarder
      image: "alpine/socat@sha256:..."
      args:
        - -d
        - TCP-LISTEN:1313,fork
        - TCP:coordinator:1313
      resources:
        requests:
          memory: 50Mi
        limits:
          memory: 50Mi
```

Upstream tracking issue: https://github.com/kata-containers/kata-containers/issues/1693.

:::

### Recovery

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

# Manifest update

This section guides you through the process of updating a manifest at the Contrast Coordinator.

## Applicability

If the manifest changes—for example, due to modifications in the service mesh configuration—it must be updated at the Coordinator.

## Prerequisites

A running Contrast deployment.

## How-to

First, connect to the Coordinator. Then update the manifest and restart your pods to apply the changes.

### Connect to the Coordinator

The released Coordinator resource includes a LoadBalancer definition we can use.

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
      image: alpine/socat
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

### Update the manifest

Set the changed manifest at the Coordinator with:

```sh
contrast set -c "${coordinator}:1313" deployment/
```

The Contrast Coordinator will rotate the mesh ca certificate on the manifest update. Workload certificates issued
after the manifest update are thus issued by another certificate authority and services receiving the new CA certificate chain
won't trust parts of the deployment that got their certificate issued before the update. This way, Contrast ensures
that parts of the deployment that received a security update won't be infected by parts of the deployment at an older
patch level that may have been compromised. The `mesh-ca.pem` is updated with the new CA certificate chain.

### Rolling out the update

The Coordinator has the new manifest set, but the different containers of the app are still
using the older certificate authority. The Contrast Initializer terminates after the initial attestation
flow and won't pull new certificates on manifest updates.

To roll out the update, use:

```sh
kubectl rollout restart <resource>
```

for all your application resources.
